package mcpserver

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/app/demo"
	"github.com/resetnak/cooldeck/internal/domain"
)

// connect runs the server over an in-memory transport and returns a client
// session talking to it.
func connect(t *testing.T, opts Options) *mcp.ClientSession {
	t.Helper()
	if opts.Service == nil {
		opts.Service = demo.New(demo.Options{})
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := New(opts)
	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func toolNames(t *testing.T, session *mcp.ClientSession) []string {
	t.Helper()
	res, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	return names
}

func TestReadOnlyByDefault(t *testing.T) {
	names := toolNames(t, connect(t, Options{}))

	// The read surface is the whole point; losing one silently would be worse
	// than the extra churn of an exact list.
	want := []string{
		"get_application", "get_deployment_logs", "get_instance_info",
		"get_runtime_logs", "list_applications", "list_deployments",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("tools = %v, want %v", names, want)
	}
}

func TestMutationsAreOptIn(t *testing.T) {
	names := toolNames(t, connect(t, Options{AllowMutations: true}))

	for _, want := range []string{
		"deploy_application", "restart_application", "start_application", "stop_application",
	} {
		if !slices.Contains(names, want) {
			t.Errorf("--allow-mutations did not register %q (got %v)", want, names)
		}
	}
	if len(names) != 10 {
		t.Fatalf("expected 6 read tools plus 4 mutations, got %d: %v", len(names), names)
	}
}

func TestListApplicationsReturnsStructuredSnapshot(t *testing.T) {
	session := connect(t, Options{})

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "list_applications"})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool reported an error: %+v", res.Content)
	}

	// StructuredContent round-trips through JSON, so decode rather than assert
	// on the map shape.
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var snapshot app.DashboardSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if len(snapshot.Applications) == 0 {
		t.Fatal("no applications in snapshot")
	}
	if snapshot.Applications[0].UUID == "" {
		t.Fatal("application UUID missing - the other tools take it as input")
	}
}

func TestToolErrorCarriesTitleAndSuggestion(t *testing.T) {
	session := connect(t, Options{})

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "get_application",
		Arguments: map[string]any{"application_uuid": "does-not-exist"},
	})
	if err != nil {
		// A failing tool must come back as a tool error the agent can read,
		// not as a protocol error that aborts the call.
		t.Fatalf("call tool: %v", err)
	}
	if !res.IsError {
		t.Fatal("unknown application did not produce a tool error")
	}

	text := contentText(res)
	if !strings.Contains(text, "Not found") {
		t.Errorf("error text %q lost the domain error title", text)
	}
	// The suggestion is the actionable half and the reason toolError exists at
	// all; asserting on the separator alone would pass on any stray hyphen.
	if !strings.Contains(text, "It may have been deleted") {
		t.Errorf("error text %q lost the suggestion", text)
	}
}

// stubService records what the tools ask of the service. The embedded
// interface leaves everything else nil, which is what we want: a test that
// reaches an unimplemented method should panic rather than quietly pass.
type stubService struct {
	app.Service
	lines       int
	limit       int
	hasDeadline bool
	deadline    time.Time
}

func (s *stubService) RuntimeLogs(_ context.Context, _ string, lines int) (app.LogSnapshot, error) {
	s.lines = lines
	return app.LogSnapshot{}, nil
}

func (s *stubService) Deployments(_ context.Context, _ string, limit int) ([]domain.Deployment, error) {
	s.limit = limit
	return nil, nil
}

func (s *stubService) Dashboard(ctx context.Context) (app.DashboardSnapshot, error) {
	s.deadline, s.hasDeadline = ctx.Deadline()
	return app.DashboardSnapshot{}, nil
}

func TestCountArgumentsAreDefaultedAndCapped(t *testing.T) {
	cases := []struct {
		name      string
		args      map[string]any
		wantLines int
		wantLimit int
	}{
		{"unset falls back to the default", map[string]any{}, defaultLogLines, defaultDeployments},
		{"a sane value is passed through", map[string]any{"lines": 25, "limit": 5}, 25, 5},
		// An agent asking for the whole log would spend its own context window
		// on one call, so the ceiling holds rather than the request.
		{"an oversized value is capped", map[string]any{"lines": 500000, "limit": 500000}, maxLogLines, maxDeployments},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := &stubService{}
			session := connect(t, Options{Service: service})

			logArgs := map[string]any{"application_uuid": "app-1"}
			maps.Copy(logArgs, tc.args)
			delete(logArgs, "limit")
			if _, err := session.CallTool(t.Context(), &mcp.CallToolParams{
				Name: "get_runtime_logs", Arguments: logArgs,
			}); err != nil {
				t.Fatalf("get_runtime_logs: %v", err)
			}

			deploymentArgs := map[string]any{"application_uuid": "app-1"}
			maps.Copy(deploymentArgs, tc.args)
			delete(deploymentArgs, "lines")
			if _, err := session.CallTool(t.Context(), &mcp.CallToolParams{
				Name: "list_deployments", Arguments: deploymentArgs,
			}); err != nil {
				t.Fatalf("list_deployments: %v", err)
			}

			if service.lines != tc.wantLines {
				t.Errorf("lines = %d, want %d", service.lines, tc.wantLines)
			}
			if service.limit != tc.wantLimit {
				t.Errorf("limit = %d, want %d", service.limit, tc.wantLimit)
			}
		})
	}
}

func TestEveryCallIsBoundedByATimeout(t *testing.T) {
	service := &stubService{}
	session := connect(t, Options{Service: service})

	if _, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "list_applications"}); err != nil {
		t.Fatalf("call tool: %v", err)
	}

	// An agent blocked on a stdio read cannot cancel a hung request itself, so
	// the deadline has to come from this side.
	if !service.hasDeadline {
		t.Fatal("service was called with an unbounded context")
	}
	if remaining := time.Until(service.deadline); remaining > requestTimeout {
		t.Fatalf("deadline is %s away, want at most %s", remaining, requestTimeout)
	}
}

func contentText(res *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if text, ok := c.(*mcp.TextContent); ok {
			b.WriteString(text.Text)
		}
	}
	return b.String()
}

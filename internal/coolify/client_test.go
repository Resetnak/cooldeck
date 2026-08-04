package coolify

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/credentials"
	"github.com/resetnak/cooldeck/internal/domain"
)

const testToken = "0|cooldeck-test-token-never-print"

func TestServiceConnectAndDashboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Errorf("Authorization = %q", got)
		}
		switch r.URL.Path {
		case "/api/v1/health":
			_, _ = fmt.Fprint(w, "OK")
		case "/api/v1/version":
			_, _ = fmt.Fprint(w, "v4.0.0-test")
		case "/api/v1/teams/current":
			_, _ = fmt.Fprint(w, `{"name":"Platform"}`)
		case "/api/v1/applications":
			_, _ = fmt.Fprint(w, `[{
				"uuid":"app-1","name":"api","status":"running:unhealthy",
				"fqdn":"https://api.example.com,https://www.example.com",
				"git_repository":"resetnak/api","git_branch":"main",
				"git_commit_sha":"1234567890","build_pack":"nixpacks",
				"environment_id":3,"destination_id":7
			}]`)
		case "/api/v1/deployments":
			_, _ = fmt.Fprint(w, `[{
				"deployment_uuid":"dep-1","application_id":"app-1",
				"application_name":"api","status":"in_progress",
				"commit":"1234567890","is_api":true,
				"created_at":"2026-08-04T12:00:00Z"
			}]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	now := time.Date(2026, 8, 4, 12, 1, 0, 0, time.UTC)
	service := newTestService(t, server, now)
	connection, err := service.Connect(context.Background())
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if connection.Version != "v4.0.0-test" || connection.TeamName != "Platform" {
		t.Fatalf("connection = %#v", connection)
	}

	dashboard, err := service.Dashboard(context.Background())
	if err != nil {
		t.Fatalf("Dashboard() error = %v", err)
	}
	if len(dashboard.Applications) != 1 || len(dashboard.ActiveDeployments) != 1 {
		t.Fatalf("dashboard counts = %d apps, %d deployments", len(dashboard.Applications), len(dashboard.ActiveDeployments))
	}
	application := dashboard.Applications[0]
	if application.Status.State != domain.StatusDegraded {
		t.Fatalf("status = %s", application.Status.State)
	}
	if len(application.FQDNs) != 2 || application.LastDeployment == nil {
		t.Fatalf("application mapping = %#v", application)
	}
}

func TestReadRetriesButMutationDoesNot(t *testing.T) {
	var reads atomic.Int32
	var posts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications":
			if reads.Add(1) == 1 {
				http.Error(w, "temporary", http.StatusBadGateway)
				return
			}
			_, _ = fmt.Fprint(w, `[]`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/applications/app-1/restart":
			posts.Add(1)
			http.Error(w, "temporary", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := newTestService(t, server, time.Now())
	if _, err := service.listApplications(context.Background()); err != nil {
		t.Fatalf("listApplications() error = %v", err)
	}
	if reads.Load() != 2 {
		t.Fatalf("read attempts = %d, want 2", reads.Load())
	}
	if _, err := service.Restart(context.Background(), "app-1"); !domain.IsKind(err, domain.ErrorServer) {
		t.Fatalf("Restart() error = %v", err)
	}
	if posts.Load() != 1 {
		t.Fatalf("mutation attempts = %d, want 1", posts.Load())
	}
}

func TestHTTPErrorDoesNotExposeResponseBody(t *testing.T) {
	secret := "server accidentally echoed " + testToken
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, secret, http.StatusForbidden)
	}))
	defer server.Close()

	service := newTestService(t, server, time.Now())
	_, err := service.listApplications(context.Background())
	if !domain.IsKind(err, domain.ErrorForbidden) {
		t.Fatalf("error kind = %v", err)
	}
	if strings.Contains(err.Error(), testToken) || strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked response body: %v", err)
	}
}

func TestDeploymentLogsAreSanitized(t *testing.T) {
	raw := `[{"output":"\u001b]2;owned\u0007ERROR failed","type":"stderr","timestamp":"2026-08-04T12:00:00Z"}]`
	lines := parseDeploymentLogs(raw)
	if len(lines) != 1 {
		t.Fatalf("len(lines) = %d", len(lines))
	}
	if strings.Contains(lines[0].Text, "\x1b") || lines[0].Stream != "stderr" {
		t.Fatalf("line = %#v", lines[0])
	}
}

func newTestService(t *testing.T, server *httptest.Server, now time.Time) *Service {
	t.Helper()
	service, err := New(Options{
		Instance:   config.Instance{ID: "test", Name: "Test", URL: server.URL},
		Token:      credentials.NewToken(testToken),
		HTTPClient: server.Client(),
		Now:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

// Package mcpserver exposes cooldeck's use cases over the Model Context
// Protocol, so an agent can inspect a Coolify instance the same way the TUI
// does.
//
// Per docs/architecture.md the adapter implements tools against app.Service
// only: no duplicate HTTP client, no TUI imports, and the same domain errors.
// Mutating tools are registered only when the operator opts in, because an
// agent cannot answer the confirmation prompt the TUI puts in front of them.
package mcpserver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/logging"
)

// Options configures the server. Service is required.
type Options struct {
	Service      app.Service
	InstanceName string
	Version      string
	// AllowMutations registers deploy, restart, start and stop. Off by default:
	// read-only is the safe posture for something an agent drives unattended.
	AllowMutations bool
}

// Tool argument types. Field names are the wire names an agent sees, and the
// jsonschema tags are the only documentation it gets about them.
type (
	noArgs struct{}

	appArgs struct {
		UUID string `json:"application_uuid" jsonschema:"UUID of the application, as returned by list_applications"`
	}

	logArgs struct {
		UUID  string `json:"application_uuid" jsonschema:"UUID of the application, as returned by list_applications"`
		Lines int    `json:"lines,omitempty" jsonschema:"how many of the most recent lines to return (default 100)"`
	}

	deploymentsArgs struct {
		UUID  string `json:"application_uuid" jsonschema:"UUID of the application, as returned by list_applications"`
		Limit int    `json:"limit,omitempty" jsonschema:"how many deployments to return, newest first (default 20)"`
	}

	deploymentArgs struct {
		UUID string `json:"deployment_uuid" jsonschema:"UUID of the deployment, as returned by list_deployments"`
	}

	deployArgs struct {
		UUID  string `json:"application_uuid" jsonschema:"UUID of the application to deploy"`
		Force bool   `json:"force,omitempty" jsonschema:"rebuild without using the Docker layer cache"`
	}
)

const (
	// requestTimeout bounds every call, mirroring the bound the TUI puts on the
	// same operations. An agent blocked on a stdio read cannot time out a hung
	// request itself, so the server has to do it.
	requestTimeout = 20 * time.Second

	// Defaults are deliberately smaller than the TUI's: every line returned here
	// lands in an agent's context window, and it can always ask for more.
	// The ceilings stop a single call from spending that window all at once.
	defaultLogLines    = 100
	maxLogLines        = 2000
	defaultDeployments = 20
	maxDeployments     = 200
)

// call runs one service operation under the request timeout and maps a failure
// onto a tool error. Every handler goes through it, so the timeout cannot be
// forgotten on a tool added later.
func call[T any](ctx context.Context, run func(context.Context) (T, error)) (*mcp.CallToolResult, T, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	out, err := run(ctx)
	if err != nil {
		var zero T
		return nil, zero, toolError(err)
	}
	return nil, out, nil
}

// clamp resolves an optional count argument. Zero means "unset, use the
// default"; anything above the ceiling is capped rather than refused, because
// failing a call over a too-large number helps nobody.
func clamp(value, fallback, ceiling int) int {
	if value <= 0 {
		return fallback
	}
	return min(value, ceiling)
}

// New builds the MCP server. Read-only tools are always registered; mutating
// ones follow Options.AllowMutations.
func New(opts Options) *mcp.Server {
	name := "cooldeck"
	if opts.InstanceName != "" {
		name += " (" + opts.InstanceName + ")"
	}
	server := mcp.NewServer(&mcp.Implementation{Name: name, Version: opts.Version}, nil)
	svc := opts.Service

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_applications",
		Description: "List every application on the Coolify instance with its current status, " +
			"project, environment and last deployment, plus the deployments currently in flight. " +
			"Start here: the other tools take UUIDs this returns.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ noArgs) (*mcp.CallToolResult, app.DashboardSnapshot, error) {
		return call(ctx, svc.Dashboard)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_application",
		Description: "Full detail for one application: configuration, domains, health check, " +
			"and its recent deployment history.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args appArgs) (*mcp.CallToolResult, app.ApplicationDetail, error) {
		return call(ctx, func(ctx context.Context) (app.ApplicationDetail, error) {
			return svc.ApplicationDetail(ctx, args.UUID)
		})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_runtime_logs",
		Description: "Runtime logs of a running application - what the container is printing now. " +
			"Use this to diagnose an application that is up but misbehaving. For why a build " +
			"failed, use get_deployment_logs instead.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args logArgs) (*mcp.CallToolResult, app.LogSnapshot, error) {
		return call(ctx, func(ctx context.Context) (app.LogSnapshot, error) {
			return svc.RuntimeLogs(ctx, args.UUID, clamp(args.Lines, defaultLogLines, maxLogLines))
		})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_deployments",
		Description: "Deployment history for one application, newest first, with status, commit and duration.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args deploymentsArgs) (*mcp.CallToolResult, []domain.Deployment, error) {
		return call(ctx, func(ctx context.Context) ([]domain.Deployment, error) {
			return svc.Deployments(ctx, args.UUID, clamp(args.Limit, defaultDeployments, maxDeployments))
		})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_deployment_logs",
		Description: "Build and deployment log for one deployment - this is where a failed " +
			"build explains itself.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args deploymentArgs) (*mcp.CallToolResult, app.LogSnapshot, error) {
		return call(ctx, func(ctx context.Context) (app.LogSnapshot, error) {
			return svc.DeploymentLogs(ctx, args.UUID)
		})
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_instance_info",
		Description: "Which Coolify instance this server talks to: URL, version, team, measured " +
			"latency and which operations the token is allowed to perform. Never includes the token.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ noArgs) (*mcp.CallToolResult, app.Connection, error) {
		return call(ctx, svc.Connect)
	})

	if opts.AllowMutations {
		addMutationTools(server, svc)
	}
	return server
}

// addMutationTools registers the state-changing half of the surface. These are
// separated so the read-only default is one branch rather than eight.
func addMutationTools(server *mcp.Server, svc app.Service) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "deploy_application",
		Description: "Queue a deployment for an application. Returns immediately with the queued " +
			"deployment's UUID; follow it with list_deployments or get_deployment_logs.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args deployArgs) (*mcp.CallToolResult, app.OperationResult, error) {
		return call(ctx, func(ctx context.Context) (app.OperationResult, error) {
			return svc.Deploy(ctx, args.UUID, app.DeployOptions{Force: args.Force})
		})
	})

	for _, m := range []struct {
		name        string
		description string
		run         func(context.Context, string) (app.OperationResult, error)
	}{
		{"restart_application", "Restart a running application.", svc.Restart},
		{"start_application", "Start a stopped application.", svc.Start},
		{"stop_application", "Stop a running application. The application stays stopped until started again.", svc.Stop},
	} {
		mcp.AddTool(server, &mcp.Tool{
			Name:        m.name,
			Description: m.description,
		}, func(ctx context.Context, _ *mcp.CallToolRequest, args appArgs) (*mcp.CallToolResult, app.OperationResult, error) {
			return call(ctx, func(ctx context.Context) (app.OperationResult, error) {
				return m.run(ctx, args.UUID)
			})
		})
	}
}

// Run serves the MCP protocol over stdin/stdout until the client disconnects
// or ctx is cancelled. Nothing else may write to stdout while this runs.
//
// The write side is wrapped rather than using StdioTransport directly: every
// frame an agent receives carries production data - runtime logs, build logs,
// commit messages - straight into a context window belonging to somebody else's
// model provider. Redacting at the transport covers tools added later too,
// which redacting per result type would not.
func Run(ctx context.Context, opts Options) error {
	return New(opts).Run(ctx, &mcp.IOTransport{
		Reader: os.Stdin,
		Writer: &redactingWriter{w: os.Stdout},
	})
}

// redactingWriter applies logging.Redact to each newline-delimited JSON frame
// on its way out. Buffering to the newline matters: a partial write must not be
// scanned, or a credential split across two Write calls would slip through the
// pattern that would otherwise have matched it whole.
type redactingWriter struct {
	w   io.Writer
	buf []byte
}

func (r *redactingWriter) Write(p []byte) (int, error) {
	r.buf = append(r.buf, p...)
	for {
		i := bytes.IndexByte(r.buf, '\n')
		if i < 0 {
			return len(p), nil
		}
		frame := logging.Redact(string(r.buf[:i]))
		r.buf = r.buf[i+1:]
		if _, err := io.WriteString(r.w, frame+"\n"); err != nil {
			return 0, err
		}
	}
}

// Close flushes a trailing frame that never got its newline, so a client
// waiting on the last response is not left hanging on shutdown. The underlying
// writer is left open on purpose: it is stdout, and the SDK's own stdio
// transport never closes that either.
func (r *redactingWriter) Close() error {
	if len(r.buf) == 0 {
		return nil
	}
	_, err := io.WriteString(r.w, logging.Redact(string(r.buf)))
	r.buf = nil
	return err
}

// toolError turns a domain error into the message the agent sees. Error() alone
// drops the title and the suggestion, which is the actionable half - an agent
// that is told "the token lacks permission, check its scope in Coolify" can
// stop and report, where a bare "403" invites a retry loop.
func toolError(err error) error {
	var derr *domain.Error
	if !errors.As(err, &derr) {
		return err
	}

	var b strings.Builder
	if derr.Title != "" {
		b.WriteString(derr.Title)
		b.WriteString(": ")
	}
	b.WriteString(derr.Error())
	if derr.Suggestion != "" {
		b.WriteString(" - ")
		b.WriteString(derr.Suggestion)
	}
	return errors.New(b.String())
}

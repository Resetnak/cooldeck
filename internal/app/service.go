// Package app holds the use cases cooldeck offers. It sits between the
// Coolify client and the presentation layers and knows nothing about Bubble
// Tea, so the same services can back the TUI, a CLI subcommand or a future MCP
// adapter without duplicating behaviour.
package app

import (
	"context"
	"time"

	"github.com/resetnak/cooldeck/internal/domain"
)

// Capabilities records what a given instance and token can actually do.
// Everything except the read capabilities starts optimistic and is downgraded
// when Coolify answers 403, because Coolify exposes no endpoint to introspect
// a token's permissions.
type Capabilities struct {
	Applications    bool
	ApplicationLogs bool
	Deployments     bool
	DeploymentLogs  bool
	Deploy          bool
	Restart         bool
	StartStop       bool
	Projects        bool
	Servers         bool
	Services        bool
	Databases       bool
}

// FullCapabilities is the optimistic starting point used before probing.
func FullCapabilities() Capabilities {
	return Capabilities{
		Applications: true, ApplicationLogs: true,
		Deployments: true, DeploymentLogs: true,
		Deploy: true, Restart: true, StartStop: true,
		Projects: true, Servers: true, Services: true, Databases: true,
	}
}

// Connection is the result of connecting to an instance: everything the header,
// the instances screen and diagnostics need, and nothing sensitive.
type Connection struct {
	InstanceID   string
	InstanceName string
	// BaseURL is the normalised instance URL, never carrying a token.
	BaseURL      string
	Version      string
	TeamName     string
	Capabilities Capabilities
	Latency      time.Duration
	ConnectedAt  time.Time
}

// DashboardSnapshot is one consistent read of everything the applications
// screen shows. Taking it as a single value means a partially-failed refresh
// cannot leave the UI showing a mix of two different points in time.
type DashboardSnapshot struct {
	Applications []domain.Application
	// ActiveDeployments are the in-flight deployments across all applications,
	// used to badge rows without a per-application request.
	ActiveDeployments []domain.Deployment
	// RecentDeployments is a bounded history (active + finished) for the
	// top-level deployments section. May equal ActiveDeployments when the
	// backend only exposes the live queue.
	RecentDeployments []domain.Deployment
	LoadedAt          time.Time
	// Partial marks a snapshot where an optional enrichment (project names,
	// deployment badges) failed but the applications themselves loaded.
	Partial bool
	// Warnings explains what was degraded. Always safe to display.
	Warnings []string
}

// ApplicationDetail is the detail screen's payload.
type ApplicationDetail struct {
	Application domain.Application
	Deployments []domain.Deployment
	LoadedAt    time.Time
}

// LogSnapshot is a point-in-time read of a log endpoint. Coolify returns logs
// as a whole string rather than a stream, so the views diff snapshots instead
// of consuming events.
type LogSnapshot struct {
	Lines    []domain.LogLine
	LoadedAt time.Time
	// Truncated reports that older lines were dropped to respect the limit.
	Truncated bool
}

// DeployOptions carries the choices a deploy modal offers.
type DeployOptions struct {
	// Force rebuilds without using the Docker layer cache.
	Force bool
}

// OperationResult is the outcome of a mutating action, shaped so a future MCP
// adapter can return it verbatim.
type OperationResult struct {
	// Operation is the verb performed: "deploy", "restart", "start", "stop".
	Operation string
	// ResourceUUID identifies what it was performed on.
	ResourceUUID string
	// ResourceName is the human label, for the confirmation toast.
	ResourceName string
	// Message is Coolify's own response text.
	Message string
	// DeploymentUUID is set when the operation queued a deployment that can be
	// followed in the deployments view.
	DeploymentUUID string
	CompletedAt    time.Time
}

// Service is everything cooldeck can do with one Coolify instance. A Service
// is bound to a single instance; the Registry maps instance IDs onto services.
//
// Implementations must respect ctx for both timeout and cancellation, and must
// return *domain.Error so callers can react to the failure kind without
// parsing message text.
type Service interface {
	// InstanceID identifies which configured instance this service talks to.
	InstanceID() string

	// Connect verifies reachability and detects capabilities. It is called on
	// startup and when the user switches instances.
	Connect(ctx context.Context) (Connection, error)

	Dashboard(ctx context.Context) (DashboardSnapshot, error)
	ApplicationDetail(ctx context.Context, appUUID string) (ApplicationDetail, error)
	RuntimeLogs(ctx context.Context, appUUID string, lines int) (LogSnapshot, error)
	Deployments(ctx context.Context, appUUID string, limit int) ([]domain.Deployment, error)
	DeploymentLogs(ctx context.Context, deploymentUUID string) (LogSnapshot, error)

	Deploy(ctx context.Context, appUUID string, opts DeployOptions) (OperationResult, error)
	Restart(ctx context.Context, appUUID string) (OperationResult, error)
	Start(ctx context.Context, appUUID string) (OperationResult, error)
	Stop(ctx context.Context, appUUID string) (OperationResult, error)

	// Capabilities returns the currently known capability set. It is updated
	// as operations succeed or are refused.
	Capabilities() Capabilities
}

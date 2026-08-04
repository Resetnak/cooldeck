// Package domain holds the models and rules cooldeck works with. It knows
// nothing about HTTP, Coolify DTOs or the terminal UI.
package domain

import "strings"

// ResourceStatus is the normalised lifecycle state of an application, service
// or database. Coolify reports Docker-flavoured strings such as
// "running:healthy" or "exited:unhealthy"; everything is funnelled through
// ParseResourceStatus so an unexpected value degrades to StatusUnknown instead
// of leaking a raw string into the UI.
type ResourceStatus string

// Recognised resource states.
const (
	StatusRunning    ResourceStatus = "running"
	StatusStopped    ResourceStatus = "stopped"
	StatusStarting   ResourceStatus = "starting"
	StatusStopping   ResourceStatus = "stopping"
	StatusRestarting ResourceStatus = "restarting"
	StatusBuilding   ResourceStatus = "building"
	StatusDeploying  ResourceStatus = "deploying"
	StatusFailed     ResourceStatus = "failed"
	StatusDegraded   ResourceStatus = "degraded"
	StatusPaused     ResourceStatus = "paused"
	StatusUnknown    ResourceStatus = "unknown"
)

// Health is the container health-check verdict Coolify appends after the colon.
type Health string

// Recognised health values.
const (
	HealthHealthy   Health = "healthy"
	HealthUnhealthy Health = "unhealthy"
	HealthStarting  Health = "starting"
	HealthNone      Health = "none"
)

// Status pairs a lifecycle state with its health verdict. Both halves matter:
// a "running:unhealthy" application is running but failing its health check,
// which the dashboard must distinguish from a plain "running:healthy".
type Status struct {
	State  ResourceStatus
	Health Health
	// Raw preserves the original API value for the detail screen and
	// diagnostics. It is a status string only, never sensitive.
	Raw string
}

// ParseStatus normalises a Coolify status string.
//
// Observed shapes: "running:healthy", "running:unhealthy", "exited:unhealthy",
// "restarting:starting", "degraded:unhealthy", "running", "exited", "" .
func ParseStatus(raw string) Status {
	s := Status{Raw: strings.TrimSpace(raw), Health: HealthNone, State: StatusUnknown}
	if s.Raw == "" {
		return s
	}

	state, health, _ := strings.Cut(strings.ToLower(s.Raw), ":")
	s.State = parseState(strings.TrimSpace(state))
	s.Health = parseHealth(strings.TrimSpace(health))

	// A container that is up but failing its probe is degraded, not healthy.
	// Surfacing that as plain "running" would hide a real production problem.
	if s.State == StatusRunning && s.Health == HealthUnhealthy {
		s.State = StatusDegraded
	}
	return s
}

func parseState(s string) ResourceStatus {
	switch s {
	case "running", "healthy":
		return StatusRunning
	case "exited", "stopped", "dead", "removing", "created":
		return StatusStopped
	case "starting":
		return StatusStarting
	case "stopping":
		return StatusStopping
	case "restarting":
		return StatusRestarting
	case "building", "in_progress", "in progress":
		return StatusBuilding
	case "deploying", "queued":
		return StatusDeploying
	case "failed", "error":
		return StatusFailed
	case "degraded":
		return StatusDegraded
	case "paused":
		return StatusPaused
	case "", "unknown":
		return StatusUnknown
	default:
		return StatusUnknown
	}
}

func parseHealth(s string) Health {
	switch s {
	case "healthy":
		return HealthHealthy
	case "unhealthy":
		return HealthUnhealthy
	case "starting":
		return HealthStarting
	default:
		return HealthNone
	}
}

// Label is the human-readable state name shown next to the status symbol.
func (s Status) Label() string {
	switch s.State {
	case StatusRunning:
		return "Running"
	case StatusStopped:
		return "Stopped"
	case StatusStarting:
		return "Starting"
	case StatusStopping:
		return "Stopping"
	case StatusRestarting:
		return "Restarting"
	case StatusBuilding:
		return "Building"
	case StatusDeploying:
		return "Deploying"
	case StatusFailed:
		return "Failed"
	case StatusDegraded:
		return "Degraded"
	case StatusPaused:
		return "Paused"
	default:
		return "Unknown"
	}
}

// IsTransitional reports whether the resource is mid-flight. Transitional
// resources are polled more eagerly and their action buttons are disabled.
func (s Status) IsTransitional() bool {
	switch s.State {
	case StatusStarting, StatusStopping, StatusRestarting, StatusBuilding, StatusDeploying:
		return true
	default:
		return false
	}
}

// IsRunning reports whether the workload is up, healthy or not.
func (s Status) IsRunning() bool {
	return s.State == StatusRunning || s.State == StatusDegraded
}

// NeedsAttention reports whether the status should stand out in the list.
func (s Status) NeedsAttention() bool {
	return s.State == StatusFailed || s.State == StatusDegraded
}

// Severity orders statuses for sorting, worst first, so that "sort by status"
// surfaces the things a user has to act on at the top of the dashboard.
func (s Status) Severity() int {
	switch s.State {
	case StatusFailed:
		return 0
	case StatusDegraded:
		return 1
	case StatusBuilding, StatusDeploying, StatusRestarting, StatusStarting, StatusStopping:
		return 2
	case StatusStopped, StatusPaused:
		return 3
	case StatusRunning:
		return 4
	default:
		return 5
	}
}

// DeploymentStatus is the normalised state of a deployment queue entry.
type DeploymentStatus string

// Recognised deployment states.
const (
	DeploymentQueued     DeploymentStatus = "queued"
	DeploymentInProgress DeploymentStatus = "in_progress"
	DeploymentFinished   DeploymentStatus = "finished"
	DeploymentFailed     DeploymentStatus = "failed"
	DeploymentCancelled  DeploymentStatus = "cancelled"
	DeploymentUnknown    DeploymentStatus = "unknown"
)

// ParseDeploymentStatus normalises the values Coolify writes to the deployment
// queue: "queued", "in_progress", "finished", "failed", "cancelled-by-user".
func ParseDeploymentStatus(raw string) DeploymentStatus {
	switch s := strings.ToLower(strings.TrimSpace(raw)); s {
	case "queued", "pending":
		return DeploymentQueued
	case "in_progress", "in progress", "running", "building":
		return DeploymentInProgress
	case "finished", "success", "succeeded", "done":
		return DeploymentFinished
	case "failed", "error":
		return DeploymentFailed
	case "":
		return DeploymentUnknown
	default:
		if strings.HasPrefix(s, "cancelled") || strings.HasPrefix(s, "canceled") {
			return DeploymentCancelled
		}
		return DeploymentUnknown
	}
}

// Label is the human-readable deployment state.
func (d DeploymentStatus) Label() string {
	switch d {
	case DeploymentQueued:
		return "Queued"
	case DeploymentInProgress:
		return "Running"
	case DeploymentFinished:
		return "Finished"
	case DeploymentFailed:
		return "Failed"
	case DeploymentCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// IsActive reports whether the deployment is still in flight.
func (d DeploymentStatus) IsActive() bool {
	return d == DeploymentQueued || d == DeploymentInProgress
}

// AsResourceStatus projects a deployment state onto the shared status vocabulary
// so the dashboard can show one consistent set of symbols.
func (d DeploymentStatus) AsResourceStatus() ResourceStatus {
	switch d {
	case DeploymentQueued:
		return StatusDeploying
	case DeploymentInProgress:
		return StatusBuilding
	case DeploymentFinished:
		return StatusRunning
	case DeploymentFailed:
		return StatusFailed
	case DeploymentCancelled:
		return StatusStopped
	default:
		return StatusUnknown
	}
}

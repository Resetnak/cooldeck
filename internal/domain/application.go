package domain

import (
	"slices"
	"strings"
	"time"
)

// HealthCheck is the subset of Coolify's health-check configuration worth
// showing. Command and response-text fields are deliberately excluded: they can
// embed credentials.
type HealthCheck struct {
	Enabled  bool
	Method   string
	Scheme   string
	Host     string
	Port     string
	Path     string
	Interval time.Duration
	Timeout  time.Duration
	Retries  int
}

// Summary renders the health check as a single line, or "" when disabled.
func (h HealthCheck) Summary() string {
	if !h.Enabled {
		return ""
	}
	var b strings.Builder
	if h.Method != "" {
		b.WriteString(strings.ToUpper(h.Method))
		b.WriteByte(' ')
	}
	if h.Scheme != "" {
		b.WriteString(h.Scheme)
		b.WriteString("://")
	}
	if h.Host != "" {
		b.WriteString(h.Host)
	}
	if h.Port != "" {
		b.WriteByte(':')
		b.WriteString(h.Port)
	}
	if h.Path != "" {
		b.WriteString(h.Path)
	}
	if b.Len() == 0 {
		return "enabled"
	}
	return b.String()
}

// DeploymentSummary is the compact deployment information carried alongside an
// application in list views, avoiding a per-application detail request.
type DeploymentSummary struct {
	UUID      string
	Status    DeploymentStatus
	CommitSHA string
	Message   string
	StartedAt time.Time
	Duration  time.Duration
}

// Application is the normalised view of a Coolify application.
type Application struct {
	UUID          string
	Name          string
	Description   string
	Status        Status
	FQDNs         []string
	RepositoryURL string
	Branch        string
	CommitSHA     string
	BuildPack     string
	HealthCheck   HealthCheck
	// LastDeployment is nil when the deployment history is unknown, which is
	// different from "never deployed".
	LastDeployment *DeploymentSummary
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PrimaryDomain returns the first configured domain, or "" when there is none.
func (a Application) PrimaryDomain() string {
	if len(a.FQDNs) == 0 {
		return ""
	}
	return a.FQDNs[0]
}

// ShortCommit returns the abbreviated commit SHA used in tables.
func (a Application) ShortCommit() string { return ShortSHA(a.CommitSHA) }

// Deployment is the normalised view of a Coolify deployment queue entry.
type Deployment struct {
	UUID            string
	ApplicationUUID string
	ApplicationName string
	Status          DeploymentStatus
	CommitSHA       string
	CommitMessage   string
	// Trigger describes what started the deployment: "webhook", "api" or "manual".
	Trigger     string
	ForceBuild  bool
	Rollback    bool
	RestartOnly bool
	ServerName  string
	StartedAt   time.Time
	FinishedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// Logs is the raw build log payload, populated only by the detail request.
	Logs []LogLine
}

// Duration returns how long the deployment ran. For an in-flight deployment it
// is measured against now, so the UI can tick it upwards.
func (d Deployment) Duration(now time.Time) time.Duration {
	start := d.StartedAt
	if start.IsZero() {
		start = d.CreatedAt
	}
	if start.IsZero() {
		return 0
	}
	end := now
	if d.FinishedAt != nil {
		end = *d.FinishedAt
	} else if !d.Status.IsActive() && !d.UpdatedAt.IsZero() {
		end = d.UpdatedAt
	}
	if end.Before(start) {
		return 0
	}
	return end.Sub(start)
}

// ShortCommit returns the abbreviated commit SHA.
func (d Deployment) ShortCommit() string { return ShortSHA(d.CommitSHA) }

// Summary projects the deployment into the compact form used in list views.
func (d Deployment) Summary(now time.Time) DeploymentSummary {
	return DeploymentSummary{
		UUID:      d.UUID,
		Status:    d.Status,
		CommitSHA: d.CommitSHA,
		Message:   d.CommitMessage,
		StartedAt: d.StartedAt,
		Duration:  d.Duration(now),
	}
}

// ShortSHA abbreviates a git SHA to the customary seven characters.
func ShortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// BaselineDuration is the median run time of the most recent successful
// deployments of one application, used to judge whether an in-flight build is
// taking longer than that application usually does. It returns zero when there
// is no usable history, which callers must treat as "no expectation to show"
// rather than "instant".
//
// Only finished deployments count. A failed or cancelled build stops for
// reasons that say nothing about how long a successful one takes, and letting
// a build that died after ten seconds drag the median down would make every
// normal build look overdue.
//
// The median over a small sample is deliberate: one pathological run - a cold
// cache, a slow registry - should not move the number a user is shown.
func BaselineDuration(deployments []Deployment, appUUID string, sample int) time.Duration {
	if sample <= 0 {
		return 0
	}

	// Order is not guaranteed by the caller, and "most recent" has to mean
	// exactly that for the baseline to track a project as it grows.
	ordered := make([]Deployment, 0, len(deployments))
	for _, d := range deployments {
		if d.Status != DeploymentFinished {
			continue
		}
		if appUUID != "" && d.ApplicationUUID != appUUID {
			continue
		}
		ordered = append(ordered, d)
	}
	slices.SortFunc(ordered, func(a, b Deployment) int {
		return b.deploymentStart().Compare(a.deploymentStart())
	})

	durations := make([]time.Duration, 0, sample)
	for _, d := range ordered {
		if len(durations) == sample {
			break
		}
		// Duration falls back to now for an unfinished deployment, but these are
		// all finished, so the zero check only skips malformed history.
		if run := d.Duration(d.finishedOrStart()); run > 0 {
			durations = append(durations, run)
		}
	}
	if len(durations) == 0 {
		return 0
	}

	slices.Sort(durations)
	mid := len(durations) / 2
	if len(durations)%2 == 1 {
		return durations[mid]
	}
	return (durations[mid-1] + durations[mid]) / 2
}

// deploymentStart is the best available start time, matching what Duration uses.
func (d Deployment) deploymentStart() time.Time {
	if !d.StartedAt.IsZero() {
		return d.StartedAt
	}
	return d.CreatedAt
}

// finishedOrStart gives Duration a reference point that cannot drift with the
// clock, so a historical duration is stable across refreshes.
func (d Deployment) finishedOrStart() time.Time {
	if d.FinishedAt != nil {
		return *d.FinishedAt
	}
	if !d.UpdatedAt.IsZero() {
		return d.UpdatedAt
	}
	return d.deploymentStart()
}

package domain

import (
	"regexp"
	"strings"
	"time"
)

// ResourceRef is a light reference to a related Coolify object. Coolify's
// application payload only carries foreign keys, so a ref may be resolved
// (Name set) or unresolved (ID only).
type ResourceRef struct {
	ID   int
	UUID string
	Name string
}

// IsZero reports whether the reference is empty.
func (r ResourceRef) IsZero() bool { return r.ID == 0 && r.UUID == "" && r.Name == "" }

// String renders the reference for display, preferring the human name.
func (r ResourceRef) String() string {
	if r.Name != "" {
		return r.Name
	}
	if r.UUID != "" {
		return r.UUID
	}
	return ""
}

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
	Project       ResourceRef
	Environment   ResourceRef
	Server        ResourceRef
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

// productionEnvPattern matches the environment names that should be treated as
// production. Matching whole words avoids flagging "reproduction-test".
var productionEnvPattern = regexp.MustCompile(`(?i)(^|[^a-z])(prod|production|live|prd)([^a-z]|$)`)

// IsProduction reports whether the application runs in a production
// environment. Production resources get louder confirmation dialogs, so a
// false negative is worse than a false positive here.
func (a Application) IsProduction() bool {
	return IsProductionEnvironment(a.Environment.Name)
}

// IsProductionEnvironment applies the production heuristic to an environment name.
func IsProductionEnvironment(name string) bool {
	return productionEnvPattern.MatchString(strings.TrimSpace(name))
}

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

// Project groups environments in Coolify.
type Project struct {
	UUID         string
	Name         string
	Description  string
	Environments []Environment
}

// Environment is a named deployment target inside a project.
type Environment struct {
	ID   int
	UUID string
	Name string
}

// Server is a machine Coolify deploys onto.
type Server struct {
	UUID        string
	Name        string
	Description string
	IP          string
	Reachable   bool
	Usable      bool
}

// Service is a Coolify managed multi-container service.
type Service struct {
	UUID        string
	Name        string
	Description string
	Type        string
	Status      Status
	Environment ResourceRef
	Server      ResourceRef
}

// Database is a Coolify managed database.
type Database struct {
	UUID        string
	Name        string
	Description string
	Type        string
	Status      Status
	Environment ResourceRef
	Server      ResourceRef
}

// ShortSHA abbreviates a git SHA to the customary seven characters.
func ShortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

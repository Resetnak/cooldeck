package demo

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/resetnak/cooldeck/internal/domain"
)

// fixture describes one handcrafted demo application. The set is chosen to
// cover every visual state the dashboard has to render: healthy, degraded,
// building, stopped, failed and unknown, plus long names and missing domains.
type fixture struct {
	name    string
	project string
	env     string
	server  string
	branch  string
	status  string
	domains []string
	repo    string
	pack    string
	// lastDeploy is how long ago the most recent deployment finished.
	lastDeploy time.Duration
	// deployStatus is the state of that deployment.
	deployStatus domain.DeploymentStatus
	description  string
}

var fixtures = []fixture{
	{
		name: "shipyard-api", project: "Shipyard", env: "production", server: "hetzner-fsn1",
		branch: "main", status: "running:healthy", domains: []string{"api.shipyard.example"},
		repo: "git@github.com:acme/shipyard-api.git", pack: "nixpacks",
		lastDeploy: 3 * time.Minute, deployStatus: domain.DeploymentFinished,
		description: "Public REST API",
	},
	{
		name: "shipyard-web", project: "Shipyard", env: "production", server: "hetzner-fsn1",
		branch: "main", status: "running:healthy", domains: []string{"shipyard.example", "www.shipyard.example"},
		repo: "git@github.com:acme/shipyard-web.git", pack: "static",
		lastDeploy: 3 * time.Minute, deployStatus: domain.DeploymentFinished,
	},
	{
		name: "landing-web", project: "Personal", env: "production", server: "hetzner-fsn1",
		branch: "main", status: "in_progress", domains: []string{"landing.example"},
		repo: "git@github.com:acme/landing.git", pack: "static",
		lastDeploy: 21 * time.Second, deployStatus: domain.DeploymentInProgress,
		description: "Personal site",
	},
	{
		name: "billing-worker", project: "Billing", env: "production", server: "hetzner-nbg1",
		branch: "main", status: "exited:unhealthy",
		repo: "git@github.com:acme/billing.git", pack: "dockerfile",
		lastDeploy: 2 * time.Hour, deployStatus: domain.DeploymentFailed,
		description: "Background job runner",
	},
	{
		name: "billing-api", project: "Billing", env: "production", server: "hetzner-nbg1",
		branch: "main", status: "running:unhealthy", domains: []string{"api.billing.example"},
		repo: "git@github.com:acme/billing.git", pack: "nixpacks",
		lastDeploy: 47 * time.Minute, deployStatus: domain.DeploymentFinished,
		description: "Failing its health probe since the last deploy",
	},
	{
		name: "metrics-api", project: "Metrics", env: "staging", server: "hetzner-nbg1",
		branch: "develop", status: "running:healthy", domains: []string{"staging-api.metrics.example"},
		repo: "git@github.com:acme/metrics.git", pack: "dockercompose",
		lastDeploy: 26 * time.Hour, deployStatus: domain.DeploymentFinished,
	},
	{
		name: "metrics-web", project: "Metrics", env: "staging", server: "hetzner-nbg1",
		branch: "develop", status: "exited", domains: []string{"staging.metrics.example"},
		repo: "git@github.com:acme/metrics.git", pack: "nixpacks",
		lastDeploy: 26 * time.Hour, deployStatus: domain.DeploymentFinished,
		description: "Stopped to save resources",
	},
	{
		name: "ingest-service-with-a-very-long-name", project: "Ingest", env: "production",
		server: "contabo-eu", branch: "main", status: "running:healthy",
		domains: []string{"ingest.example-with-a-rather-long-domain-name.example"},
		repo:    "git@github.com:acme/ingest.git", pack: "dockerfile",
		lastDeploy: 5 * 24 * time.Hour, deployStatus: domain.DeploymentFinished,
		description: "Exercises column truncation",
	},
	{
		name: "ingest-dashboard", project: "Ingest", env: "preview", server: "contabo-eu",
		branch: "feat/charts", status: "restarting:starting",
		repo: "git@github.com:acme/ingest.git", pack: "railpack",
		lastDeploy: 90 * time.Second, deployStatus: domain.DeploymentFinished,
	},
	{
		name: "sandbox-api", project: "Sandbox", env: "development", server: "local-dev",
		branch: "main", status: "", // an unknown status must render safely
		repo: "git@github.com:acme/sandbox.git", pack: "dockercompose",
		deployStatus: domain.DeploymentUnknown,
	},
	{
		name: "notes-app", project: "Personal", env: "production", server: "hetzner-fsn1",
		branch: "main", status: "running:healthy", domains: []string{"notes.example"},
		repo: "git@github.com:acme/notes-app.git", pack: "nixpacks",
		lastDeploy: 11 * 24 * time.Hour, deployStatus: domain.DeploymentFinished,
	},
	{
		name: "vault-web", project: "Personal", env: "production", server: "hetzner-fsn1",
		branch: "main", status: "queued", domains: []string{"vault.example"},
		repo: "git@github.com:acme/vault-web.git", pack: "nixpacks",
		lastDeploy: 4 * time.Second, deployStatus: domain.DeploymentQueued,
	},
}

// generate builds the fleet, its deployment history and its logs. It is
// deterministic for a given seed and clock.
func (s *Service) generate() {
	r := rand.New(rand.NewSource(s.opts.Seed))
	now := s.now()

	set := fixtures
	if s.opts.AppCount > 0 {
		set = padFixtures(set, s.opts.AppCount)
	}

	s.apps = make([]domain.Application, 0, len(set))
	for i, f := range set {
		uuid := fmt.Sprintf("demo%03d%s", i, randomSuffix(r))
		a := domain.Application{
			UUID:          uuid,
			Name:          f.name,
			Description:   f.description,
			Status:        domain.ParseStatus(f.status),
			FQDNs:         f.domains,
			RepositoryURL: f.repo,
			Branch:        f.branch,
			CommitSHA:     randomSHA(r),
			BuildPack:     f.pack,
			Project:       domain.ResourceRef{ID: i + 1, Name: f.project},
			Environment:   domain.ResourceRef{ID: i + 1, Name: f.env},
			Server:        domain.ResourceRef{ID: i + 1, Name: f.server, UUID: "srv-" + f.server},
			CreatedAt:     now.Add(-time.Duration(60+r.Intn(300)) * 24 * time.Hour),
			UpdatedAt:     now.Add(-f.lastDeploy),
		}
		if len(f.domains) > 0 {
			a.HealthCheck = domain.HealthCheck{
				Enabled: true, Method: "GET", Scheme: "http", Host: "localhost",
				Port: "3000", Path: "/health",
				Interval: 30 * time.Second, Timeout: 5 * time.Second, Retries: 3,
			}
		}

		deps := s.generateDeployments(r, a, f, now)
		if len(deps) > 0 {
			summary := deps[0].Summary(now)
			a.LastDeployment = &summary
		}
		s.deployments[uuid] = deps
		s.logs[uuid] = runtimeLog(f, now)
		s.apps = append(s.apps, a)
	}
}

// generateDeployments builds a plausible history, newest first.
func (s *Service) generateDeployments(
	r *rand.Rand, a domain.Application, f fixture, now time.Time,
) []domain.Deployment {
	if f.deployStatus == domain.DeploymentUnknown {
		return nil
	}

	count := 4 + r.Intn(5)
	deps := make([]domain.Deployment, 0, count)
	triggers := []string{"webhook", "manual", "api"}

	age := f.lastDeploy
	for i := range count {
		status := domain.DeploymentFinished
		if i == 0 {
			status = f.deployStatus
		} else if r.Intn(6) == 0 {
			status = domain.DeploymentFailed
		}

		started := now.Add(-age)
		d := domain.Deployment{
			UUID:            fmt.Sprintf("dep-%s-%02d", a.UUID, i),
			ApplicationUUID: a.UUID,
			ApplicationName: a.Name,
			Status:          status,
			CommitSHA:       randomSHA(r),
			CommitMessage:   commitMessages[r.Intn(len(commitMessages))],
			Trigger:         triggers[r.Intn(len(triggers))],
			ServerName:      f.server,
			StartedAt:       started,
			CreatedAt:       started,
			UpdatedAt:       started,
		}
		if i == 0 {
			d.CommitSHA = a.CommitSHA
		}
		if !status.IsActive() {
			dur := time.Duration(25+r.Intn(140)) * time.Second
			finished := started.Add(dur)
			d.FinishedAt = &finished
			d.UpdatedAt = finished
		}
		d.Logs, _ = domain.ParseLogPayload(buildLog(a.Name, d.CommitSHA, status), 0)
		deps = append(deps, d)

		age += time.Duration(6+r.Intn(60)) * time.Hour
	}
	return deps
}

var commitMessages = []string{
	"feat: add weekly summary endpoint",
	"fix: handle empty response from upstream",
	"chore(deps): bump base image to alpine 3.21",
	"refactor: extract deployment mapper",
	"fix: correct timezone handling in reports",
	"perf: cache project lookups",
	"docs: document the new environment variables",
	"ci: run tests on arm64 too",
}

// buildLog renders a believable build transcript, including the escape
// sequences a real build emits, so the log sanitiser is exercised in the demo.
func buildLog(appName, sha string, status domain.DeploymentStatus) string {
	short := domain.ShortSHA(sha)
	lines := []string{
		"Starting deployment of " + appName,
		"Pulling latest changes from git (commit " + short + ")",
		"\x1b[32m✓\x1b[0m Repository cloned",
		"Building image with nixpacks",
		"#1 [internal] load build definition from Dockerfile",
		"#1 transferring dockerfile: 1.42kB done",
		"#2 [internal] load .dockerignore",
		"#3 [builder 1/6] FROM docker.io/library/node:22-alpine",
		"#4 [builder 3/6] RUN npm ci --omit=dev",
		"npm WARN deprecated inflight@1.0.6: This module is not supported",
		"added 412 packages in 18s",
		"#5 [builder 5/6] RUN npm run build",
		"INFO  build completed in 31.2s",
		"#6 exporting to image",
		"#6 writing image sha256:" + strings.Repeat("f0", 16),
	}
	switch status {
	case domain.DeploymentFinished:
		lines = append(lines,
			"Stopping old container",
			"Starting new container",
			"Waiting for health check to pass",
			"\x1b[32m✓\x1b[0m Health check passed",
			"New container is healthy, deployment successful",
		)
	case domain.DeploymentFailed:
		lines = append(lines,
			"ERROR  npm run build exited with code 1",
			"src/pages/report.tsx:42:18 - error TS2345: Argument of type 'string' is not assignable to parameter of type 'Date'.",
			"\x1b[31m×\x1b[0m Build failed",
			"Deployment failed, keeping the previous container running",
		)
	case domain.DeploymentQueued:
		lines = lines[:2]
		lines = append(lines, "Deployment queued, waiting for a free build slot")
	case domain.DeploymentInProgress:
		lines = lines[:10]
	case domain.DeploymentCancelled:
		lines = append(lines, "WARN  deployment cancelled by user")
	}
	return strings.Join(lines, "\n")
}

// runtimeLog renders believable application output with mixed levels.
func runtimeLog(f fixture, now time.Time) []string {
	start := now.Add(-4 * time.Minute)
	stamp := func(i int) string {
		return start.Add(time.Duration(i) * 7 * time.Second).UTC().Format(time.RFC3339)
	}

	body := []string{
		"INFO  starting " + f.name + " (build " + f.branch + ")",
		"INFO  connected to postgres at db:5432",
		"INFO  listening on 0.0.0.0:3000",
		"DEBUG cache warm-up finished in 42ms",
		"INFO  GET /healthz 200 in 2ms",
		"INFO  GET /api/v1/entries 200 in 31ms",
		"WARN  slow query detected: SELECT * FROM entries took 812ms",
		"INFO  POST /api/v1/entries 201 in 64ms",
		"INFO  GET /healthz 200 in 2ms",
		"DEBUG scheduled job 'digest' skipped, not due yet",
	}
	switch f.deployStatus {
	case domain.DeploymentFailed:
		body = append(body,
			"ERROR failed to connect to redis: dial tcp 10.0.1.4:6379: connect: connection refused",
			"ERROR job runner stopped after 5 consecutive failures",
			"FATAL shutting down",
		)
	default:
		if f.status == "running:unhealthy" {
			body = append(body,
				"WARN  health check /health returned 503",
				"ERROR upstream dependency timed out after 5s",
				"WARN  health check /health returned 503",
			)
		}
	}

	out := make([]string, 0, len(body))
	for i, l := range body {
		out = append(out, stamp(i)+" "+l)
	}
	return out
}

// padFixtures grows the fixture set to n entries so the dashboard can be
// exercised with a large fleet (`--demo-apps 200`).
func padFixtures(base []fixture, n int) []fixture {
	if n <= len(base) {
		return base[:n]
	}
	out := make([]fixture, 0, n)
	out = append(out, base...)
	for i := len(base); i < n; i++ {
		f := base[i%len(base)]
		f.name = fmt.Sprintf("%s-%02d", f.name, i/len(base))
		f.domains = nil
		out = append(out, f)
	}
	return out
}

func randomSuffix(r *rand.Rand) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = alphabet[r.Intn(len(alphabet))]
	}
	return string(b)
}

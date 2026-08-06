// Package demo provides an app.Service backed by generated data. It lets the
// whole UI be exercised, demonstrated and snapshot-tested without a Coolify
// instance, a token or a network.
package demo

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/domain"
)

// DefaultSeed keeps generated data identical between runs, which is what makes
// the demo usable for screenshots and golden tests.
const DefaultSeed int64 = 20260804

// Options configures the demo service.
type Options struct {
	// Seed drives every random choice. Equal seeds produce equal data.
	Seed int64
	// Latency simulates network round-trip time so loading states are visible.
	// Zero disables the delay, which is what tests want.
	Latency time.Duration
	// AppCount overrides the size of the generated fleet. Zero uses the
	// handcrafted fixture set; larger values pad it for performance testing.
	AppCount int
	// Now injects the clock. Nil uses time.Now.
	Now func() time.Time
}

// Service is a deterministic in-memory app.Service.
type Service struct {
	opts Options
	now  func() time.Time

	mu   sync.Mutex
	apps []domain.Application
	// deployments is keyed by application UUID, newest first.
	deployments map[string][]domain.Deployment
	logs        map[string][]string
	// offline makes every call fail, so the offline and error states can be
	// demonstrated and tested on demand.
	offline bool
	// startedAt anchors the simulated in-flight deployment.
	startedAt time.Time
}

var _ app.Service = (*Service)(nil)

// New builds a demo service.
func New(opts Options) *Service {
	if opts.Seed == 0 {
		opts.Seed = DefaultSeed
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	s := &Service{
		opts:        opts,
		now:         now,
		deployments: map[string][]domain.Deployment{},
		logs:        map[string][]string{},
		startedAt:   now(),
	}
	s.generate()
	return s
}

// InstanceID implements app.Service.
func (s *Service) InstanceID() string { return "demo" }

// SetOffline toggles the simulated outage used to demonstrate the offline and
// error states. It is safe to call from another goroutine.
func (s *Service) SetOffline(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.offline = v
}

// Offline reports whether the simulated outage is active.
func (s *Service) Offline() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.offline
}

// Capabilities implements app.Service. The demo advertises everything so the
// full command palette can be explored.
func (s *Service) Capabilities() app.Capabilities { return app.FullCapabilities() }

// Connect implements app.Service.
func (s *Service) Connect(ctx context.Context) (app.Connection, error) {
	if err := s.simulate(ctx); err != nil {
		return app.Connection{}, err
	}
	return app.Connection{
		InstanceID:   s.InstanceID(),
		InstanceName: "Demo",
		BaseURL:      "https://coolify.demo.invalid",
		Version:      "4.0.0-demo",
		TeamName:     "Demo Team",
		Capabilities: s.Capabilities(),
		Latency:      s.opts.Latency,
		ConnectedAt:  s.now(),
	}, nil
}

// Dashboard implements app.Service.
func (s *Service) Dashboard(ctx context.Context) (app.DashboardSnapshot, error) {
	if err := s.simulate(ctx); err != nil {
		return app.DashboardSnapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advanceLocked()

	snap := app.DashboardSnapshot{
		Applications: append([]domain.Application(nil), s.apps...),
		LoadedAt:     s.now(),
	}
	// Collect a short history per application (newest first already).
	for _, a := range s.apps {
		deps := s.deployments[a.UUID]
		for i, d := range deps {
			if d.Status.IsActive() {
				snap.ActiveDeployments = append(snap.ActiveDeployments, d)
			}
			// Cap recent history so the global list stays scannable.
			if i < 3 {
				snap.RecentDeployments = append(snap.RecentDeployments, d)
			}
		}
	}
	return snap, nil
}

// ApplicationDetail implements app.Service.
func (s *Service) ApplicationDetail(ctx context.Context, appUUID string) (app.ApplicationDetail, error) {
	if err := s.simulate(ctx); err != nil {
		return app.ApplicationDetail{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advanceLocked()

	for _, a := range s.apps {
		if a.UUID == appUUID {
			return app.ApplicationDetail{
				Application: a,
				Deployments: append([]domain.Deployment(nil), s.deployments[appUUID]...),
				LoadedAt:    s.now(),
			}, nil
		}
	}
	return app.ApplicationDetail{}, domain.NewError(domain.ErrorNotFound, nil).
		WithOperation("get application")
}

// RuntimeLogs implements app.Service.
func (s *Service) RuntimeLogs(ctx context.Context, appUUID string, lines int) (app.LogSnapshot, error) {
	if err := s.simulate(ctx); err != nil {
		return app.LogSnapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, ok := s.logs[appUUID]
	if !ok {
		return app.LogSnapshot{}, domain.NewError(domain.ErrorNotFound, nil).
			WithOperation("get application logs")
	}
	// Grow the tail over time so "follow" visibly does something.
	elapsed := int(s.now().Sub(s.startedAt) / (2 * time.Second))
	if elapsed > 0 {
		raw = append(append([]string(nil), raw...), s.tailLines(appUUID, elapsed)...)
	}

	parsed, truncated := domain.ParseLogPayload(strings.Join(raw, "\n"), lines)
	return app.LogSnapshot{
		Lines:     parsed,
		LoadedAt:  s.now(),
		Truncated: truncated,
	}, nil
}

// Deployments implements app.Service.
func (s *Service) Deployments(ctx context.Context, appUUID string, limit int) ([]domain.Deployment, error) {
	if err := s.simulate(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advanceLocked()

	deps := s.deployments[appUUID]
	if limit > 0 && len(deps) > limit {
		deps = deps[:limit]
	}
	return append([]domain.Deployment(nil), deps...), nil
}

// DeploymentLogs implements app.Service.
func (s *Service) DeploymentLogs(ctx context.Context, deploymentUUID string) (app.LogSnapshot, error) {
	if err := s.simulate(ctx); err != nil {
		return app.LogSnapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advanceLocked()

	for _, deps := range s.deployments {
		for _, d := range deps {
			if d.UUID != deploymentUUID {
				continue
			}
			return app.LogSnapshot{Lines: d.Logs, LoadedAt: s.now()}, nil
		}
	}
	return app.LogSnapshot{}, domain.NewError(domain.ErrorNotFound, nil).
		WithOperation("get deployment logs")
}

// Deploy implements app.Service. Nothing is really deployed: a queued
// deployment is inserted so the UI can follow it through to completion.
func (s *Service) Deploy(ctx context.Context, appUUID string, opts app.DeployOptions) (app.OperationResult, error) {
	return s.mutate(ctx, appUUID, "deploy", func(a *domain.Application) string {
		d := s.newDeployment(a, "manual", opts.Force)
		s.deployments[a.UUID] = append([]domain.Deployment{d}, s.deployments[a.UUID]...)
		a.Status = domain.ParseStatus("in_progress")
		return d.UUID
	})
}

// Restart implements app.Service.
func (s *Service) Restart(ctx context.Context, appUUID string) (app.OperationResult, error) {
	return s.mutate(ctx, appUUID, "restart", func(a *domain.Application) string {
		a.Status = domain.ParseStatus("restarting")
		return ""
	})
}

// Start implements app.Service.
func (s *Service) Start(ctx context.Context, appUUID string) (app.OperationResult, error) {
	return s.mutate(ctx, appUUID, "start", func(a *domain.Application) string {
		a.Status = domain.ParseStatus("running:starting")
		return ""
	})
}

// Stop implements app.Service.
func (s *Service) Stop(ctx context.Context, appUUID string) (app.OperationResult, error) {
	return s.mutate(ctx, appUUID, "stop", func(a *domain.Application) string {
		a.Status = domain.ParseStatus("exited")
		return ""
	})
}

func (s *Service) mutate(
	ctx context.Context,
	appUUID, operation string,
	apply func(*domain.Application) string,
) (app.OperationResult, error) {
	if err := s.simulate(ctx); err != nil {
		return app.OperationResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.apps {
		if s.apps[i].UUID != appUUID {
			continue
		}
		deploymentUUID := apply(&s.apps[i])
		return app.OperationResult{
			Operation:      operation,
			ResourceUUID:   appUUID,
			ResourceName:   s.apps[i].Name,
			Message:        strings.ToUpper(operation[:1]) + operation[1:] + " request queued.",
			DeploymentUUID: deploymentUUID,
			CompletedAt:    s.now(),
		}, nil
	}
	return app.OperationResult{}, domain.NewError(domain.ErrorNotFound, nil).WithOperation(operation)
}

// simulate applies the configured latency and the offline switch, honouring
// cancellation so the UI's request-superseding logic can be exercised.
func (s *Service) simulate(ctx context.Context) error {
	if s.opts.Latency > 0 {
		select {
		case <-ctx.Done():
			return domain.NewError(domain.ErrorCancelled, ctx.Err())
		case <-time.After(s.opts.Latency):
		}
	}
	if err := ctx.Err(); err != nil {
		return domain.NewError(domain.ErrorCancelled, err)
	}
	if s.Offline() {
		e := domain.NewError(domain.ErrorNetwork, nil)
		e.Detail = "demo mode is simulating an outage; press F2 to restore the connection"
		return e
	}
	return nil
}

// advanceLocked moves the simulated in-flight deployment forward so a demo
// left running shows a build progressing and finishing on its own.
func (s *Service) advanceLocked() {
	for uuid, deps := range s.deployments {
		for i := range deps {
			d := &deps[i]
			if !d.Status.IsActive() {
				continue
			}
			age := s.now().Sub(d.StartedAt)
			switch {
			case age > 90*time.Second:
				finished := d.StartedAt.Add(82 * time.Second)
				d.Status = domain.DeploymentFinished
				d.FinishedAt = &finished
				d.UpdatedAt = finished
				s.setAppStatusLocked(uuid, "running:healthy")
			case age > 20*time.Second && d.Status == domain.DeploymentQueued:
				d.Status = domain.DeploymentInProgress
				s.setAppStatusLocked(uuid, "in_progress")
			}
		}
	}
}

func (s *Service) setAppStatusLocked(appUUID, raw string) {
	for i := range s.apps {
		if s.apps[i].UUID == appUUID {
			s.apps[i].Status = domain.ParseStatus(raw)
			return
		}
	}
}

func (s *Service) newDeployment(a *domain.Application, trigger string, force bool) domain.Deployment {
	r := rand.New(rand.NewSource(s.opts.Seed + int64(len(s.deployments[a.UUID]))))
	now := s.now()
	d := domain.Deployment{
		UUID:            fmt.Sprintf("dep-%s-%d", a.UUID, now.UnixNano()),
		ApplicationUUID: a.UUID,
		ApplicationName: a.Name,
		Status:          domain.DeploymentQueued,
		CommitSHA:       randomSHA(r),
		CommitMessage:   "chore: trigger deployment from cooldeck demo",
		Trigger:         trigger,
		ForceBuild:      force,
		StartedAt:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	d.Logs, _ = domain.ParseLogPayload(buildLog(a.Name, d.CommitSHA, domain.DeploymentQueued), 0)
	return d
}

// tailLines serves the same synthetic lines for every application; the demo
// only needs plausible traffic, not per-app history.
func (s *Service) tailLines(_ string, n int) []string {
	if n > 40 {
		n = 40
	}
	out := make([]string, 0, n)
	base := s.startedAt
	for i := range n {
		ts := base.Add(time.Duration(i) * 2 * time.Second).UTC().Format(time.RFC3339)
		out = append(out, fmt.Sprintf("%s INFO  GET /healthz 200 in %dms", ts, 3+(i*7)%40))
	}
	return out
}

func randomSHA(r *rand.Rand) string {
	const hex = "0123456789abcdef"
	b := make([]byte, 40)
	for i := range b {
		b[i] = hex[r.Intn(len(hex))]
	}
	return string(b)
}

package coolify

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/credentials"
	"github.com/resetnak/cooldeck/internal/domain"
)

// Options binds a Service to one configured Coolify instance.
type Options struct {
	Instance   config.Instance
	Token      credentials.Token
	HTTPClient *http.Client
	Now        func() time.Time
}

// Service implements app.Service against the Coolify REST API.
type Service struct {
	instance config.Instance
	client   *client
	now      func() time.Time

	mu           sync.RWMutex
	capabilities app.Capabilities
}

var _ app.Service = (*Service)(nil)

// New creates a service without performing network I/O.
func New(opts Options) (*Service, error) {
	client, err := newClient(clientOptions{
		baseURL:    opts.Instance.URL,
		token:      opts.Token,
		insecure:   opts.Instance.Insecure,
		httpClient: opts.HTTPClient,
	})
	if err != nil {
		return nil, err
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		instance:     opts.Instance,
		client:       client,
		now:          now,
		capabilities: app.FullCapabilities(),
	}, nil
}

func (s *Service) InstanceID() string { return s.instance.ID }

func (s *Service) Capabilities() app.Capabilities {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.capabilities
}

func (s *Service) Connect(ctx context.Context) (app.Connection, error) {
	started := s.now()
	if _, err := s.client.getText(ctx, "health"); err != nil {
		return app.Connection{}, domain.AsError(err).WithOperation("check Coolify health")
	}
	version, err := s.client.getText(ctx, "version")
	if err != nil {
		return app.Connection{}, domain.AsError(err).WithOperation("read Coolify version")
	}
	team := teamDTO{}
	if err := s.client.getJSON(ctx, "teams/current", nil, &team); err != nil {
		if domain.IsKind(err, domain.ErrorUnauthorized) {
			return app.Connection{}, domain.AsError(err).WithOperation("authenticate")
		}
		team.Name = ""
	}
	return app.Connection{
		InstanceID:   s.instance.ID,
		InstanceName: s.instance.DisplayName(),
		BaseURL:      strings.TrimSuffix(s.client.baseURL.String(), "/api/v1/"),
		Version:      version,
		TeamName:     team.Name,
		Capabilities: s.Capabilities(),
		Latency:      s.now().Sub(started),
		ConnectedAt:  s.now(),
	}, nil
}

func (s *Service) Dashboard(ctx context.Context) (app.DashboardSnapshot, error) {
	applications, err := s.listApplications(ctx)
	if err != nil {
		s.downgradeForError(err, "applications")
		return app.DashboardSnapshot{}, domain.AsError(err).WithOperation("list applications")
	}
	deployments, deployErr := s.listDeployments(ctx)
	warnings := []string{}
	if deployErr != nil {
		s.downgradeForError(deployErr, "deployments")
		warnings = append(warnings, "Deployment status is temporarily unavailable.")
		deployments = []domain.Deployment{}
	}
	active, recent := splitDeployments(deployments, 100)
	attachDeployments(applications, deployments, s.now())
	return app.DashboardSnapshot{
		Applications:      applications,
		ActiveDeployments: active,
		RecentDeployments: recent,
		LoadedAt:          s.now(),
		Partial:           deployErr != nil,
		Warnings:          warnings,
	}, nil
}

func (s *Service) ApplicationDetail(ctx context.Context, appUUID string) (app.ApplicationDetail, error) {
	dto := applicationDTO{}
	if err := s.client.getJSON(ctx, "applications/"+url.PathEscape(appUUID), nil, &dto); err != nil {
		return app.ApplicationDetail{}, domain.AsError(err).WithOperation("get application")
	}
	deployments, deployErr := s.Deployments(ctx, appUUID, 10)
	if deployErr != nil {
		// The detail is still worth rendering without its history, but a 403
		// must downgrade the capability like everywhere else - silence here
		// would render "no deployments" for a permissions problem.
		s.downgradeForError(deployErr, "deployments")
		deployments = nil
	}
	application := mapApplication(dto)
	if len(deployments) > 0 {
		summary := deployments[0].Summary(s.now())
		application.LastDeployment = &summary
	}
	return app.ApplicationDetail{Application: application, Deployments: deployments, LoadedAt: s.now()}, nil
}

func (s *Service) RuntimeLogs(ctx context.Context, appUUID string, lines int) (app.LogSnapshot, error) {
	query := url.Values{"lines": {strconv.Itoa(lines)}, "show_timestamps": {"true"}}
	dto := logsDTO{}
	if err := s.client.getJSON(ctx, "applications/"+url.PathEscape(appUUID)+"/logs", query, &dto); err != nil {
		s.downgradeForError(err, "application_logs")
		return app.LogSnapshot{}, domain.AsError(err).WithOperation("get application logs")
	}
	parsed, truncated := domain.ParseLogPayload(dto.Logs, lines)
	return app.LogSnapshot{Lines: parsed, LoadedAt: s.now(), Truncated: truncated}, nil
}

func (s *Service) Deployments(ctx context.Context, appUUID string, limit int) ([]domain.Deployment, error) {
	query := url.Values{"skip": {"0"}, "take": {strconv.Itoa(limit)}}
	dtos := []deploymentDTO{}
	path := "deployments/applications/" + url.PathEscape(appUUID)
	if err := s.client.getJSON(ctx, path, query, &dtos); err != nil {
		return nil, domain.AsError(err).WithOperation("list deployments")
	}
	return mapDeployments(dtos), nil
}

func (s *Service) DeploymentLogs(ctx context.Context, deploymentUUID string) (app.LogSnapshot, error) {
	dto := deploymentDTO{}
	if err := s.client.getJSON(ctx, "deployments/"+url.PathEscape(deploymentUUID), nil, &dto); err != nil {
		s.downgradeForError(err, "deployment_logs")
		return app.LogSnapshot{}, domain.AsError(err).WithOperation("get deployment logs")
	}
	return app.LogSnapshot{Lines: parseDeploymentLogs(dto.Logs), LoadedAt: s.now()}, nil
}

func (s *Service) Deploy(ctx context.Context, appUUID string, opts app.DeployOptions) (app.OperationResult, error) {
	query := url.Values{"uuid": {appUUID}, "force": {strconv.FormatBool(opts.Force)}}
	dto := deployResponseDTO{}
	if err := s.client.postJSON(ctx, "deploy", query, &dto); err != nil {
		s.downgradeForError(err, "deploy")
		return app.OperationResult{}, domain.AsError(err).WithOperation("deploy application")
	}
	if len(dto.Deployments) == 0 {
		return app.OperationResult{}, domain.NewError(domain.ErrorDecode, nil).WithOperation("deploy application")
	}
	result := dto.Deployments[0]
	return app.OperationResult{
		Operation:      "deploy",
		ResourceUUID:   result.ResourceUUID,
		Message:        result.Message,
		DeploymentUUID: result.DeploymentUUID,
		CompletedAt:    s.now(),
	}, nil
}

func (s *Service) Restart(ctx context.Context, appUUID string) (app.OperationResult, error) {
	return s.mutate(ctx, appUUID, "restart")
}

func (s *Service) Start(ctx context.Context, appUUID string) (app.OperationResult, error) {
	return s.mutate(ctx, appUUID, "start")
}

func (s *Service) Stop(ctx context.Context, appUUID string) (app.OperationResult, error) {
	return s.mutate(ctx, appUUID, "stop")
}

func (s *Service) mutate(ctx context.Context, appUUID, operation string) (app.OperationResult, error) {
	dto := operationDTO{}
	path := fmt.Sprintf("applications/%s/%s", url.PathEscape(appUUID), operation)
	if err := s.client.postJSON(ctx, path, nil, &dto); err != nil {
		s.downgradeForError(err, operation)
		return app.OperationResult{}, domain.AsError(err).WithOperation(operation + " application")
	}
	return app.OperationResult{
		Operation:      operation,
		ResourceUUID:   appUUID,
		Message:        dto.Message,
		DeploymentUUID: dto.DeploymentUUID,
		CompletedAt:    s.now(),
	}, nil
}

func (s *Service) listApplications(ctx context.Context) ([]domain.Application, error) {
	dtos := []applicationDTO{}
	if err := s.client.getJSON(ctx, "applications", nil, &dtos); err != nil {
		return nil, err
	}
	applications := make([]domain.Application, 0, len(dtos))
	for _, dto := range dtos {
		applications = append(applications, mapApplication(dto))
	}
	return applications, nil
}

func (s *Service) listDeployments(ctx context.Context) ([]domain.Deployment, error) {
	dtos := []deploymentDTO{}
	if err := s.client.getJSON(ctx, "deployments", nil, &dtos); err != nil {
		return nil, err
	}
	return mapDeployments(dtos), nil
}

func mapDeployments(dtos []deploymentDTO) []domain.Deployment {
	deployments := make([]domain.Deployment, 0, len(dtos))
	for _, dto := range dtos {
		deployments = append(deployments, mapDeployment(dto))
	}
	// splitDeployments and attachDeployments assume newest first; enforce it
	// here instead of trusting the API's ordering.
	sort.SliceStable(deployments, func(i, j int) bool {
		return deployments[i].CreatedAt.After(deployments[j].CreatedAt)
	})
	return deployments
}

// splitDeployments separates the live queue from a bounded recent history
// (active first, then finished/failed, newest first overall within each group).
func splitDeployments(all []domain.Deployment, recentLimit int) (active, recent []domain.Deployment) {
	active = make([]domain.Deployment, 0, len(all))
	rest := make([]domain.Deployment, 0, len(all))
	for _, d := range all {
		if d.Status.IsActive() {
			active = append(active, d)
		} else {
			rest = append(rest, d)
		}
	}
	recent = append(append([]domain.Deployment{}, active...), rest...)
	if recentLimit > 0 && len(recent) > recentLimit {
		recent = recent[:recentLimit]
	}
	return active, recent
}

func attachDeployments(applications []domain.Application, deployments []domain.Deployment, now time.Time) {
	byApplication := make(map[string]domain.Deployment, len(deployments))
	for _, deployment := range deployments {
		if _, exists := byApplication[deployment.ApplicationUUID]; !exists {
			byApplication[deployment.ApplicationUUID] = deployment
		}
	}
	for i := range applications {
		if deployment, ok := byApplication[applications[i].UUID]; ok {
			summary := deployment.Summary(now)
			applications[i].LastDeployment = &summary
		}
	}
}

func (s *Service) downgradeForError(err error, capability string) {
	if !domain.IsKind(err, domain.ErrorForbidden) && !domain.IsKind(err, domain.ErrorNotFound) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch capability {
	case "applications":
		s.capabilities.Applications = false
	case "application_logs":
		s.capabilities.ApplicationLogs = false
	case "deployments":
		s.capabilities.Deployments = false
	case "deployment_logs":
		s.capabilities.DeploymentLogs = false
	case "deploy":
		s.capabilities.Deploy = false
	case "restart":
		s.capabilities.Restart = false
	case "start", "stop":
		s.capabilities.StartStop = false
	}
}

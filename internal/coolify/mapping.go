package coolify

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/resetnak/cooldeck/internal/domain"
)

func mapApplication(dto applicationDTO) domain.Application {
	return domain.Application{
		UUID:          dto.UUID,
		Name:          dto.Name,
		Description:   stringValue(dto.Description),
		Status:        domain.ParseStatus(dto.Status),
		FQDNs:         domain.SplitFQDNs(stringValue(dto.FQDN)),
		RepositoryURL: dto.GitRepository,
		Branch:        dto.GitBranch,
		CommitSHA:     dto.GitCommitSHA,
		BuildPack:     dto.BuildPack,
		Environment:   domain.ResourceRef{ID: dto.EnvironmentID},
		Server:        domain.ResourceRef{ID: dto.DestinationID},
		HealthCheck: domain.HealthCheck{
			Enabled:  dto.HealthCheckEnabled,
			Method:   dto.HealthCheckMethod,
			Scheme:   dto.HealthCheckScheme,
			Host:     dto.HealthCheckHost,
			Port:     dto.HealthCheckPort,
			Path:     dto.HealthCheckPath,
			Interval: time.Duration(dto.HealthCheckInterval) * time.Second,
			Timeout:  time.Duration(dto.HealthCheckTimeout) * time.Second,
			Retries:  dto.HealthCheckRetries,
		},
		CreatedAt: parseTime(dto.CreatedAt),
		UpdatedAt: parseTime(dto.UpdatedAt),
	}
}

func mapDeployment(dto deploymentDTO) domain.Deployment {
	created := parseTime(dto.CreatedAt)
	updated := parseTime(dto.UpdatedAt)
	deployment := domain.Deployment{
		UUID:            dto.DeploymentUUID,
		ApplicationUUID: dto.ApplicationID,
		ApplicationName: dto.ApplicationName,
		Status:          domain.ParseDeploymentStatus(dto.Status),
		CommitSHA:       dto.Commit,
		CommitMessage:   dto.CommitMessage,
		Trigger:         deploymentTrigger(dto),
		ForceBuild:      dto.ForceRebuild,
		Rollback:        dto.Rollback,
		RestartOnly:     dto.RestartOnly,
		ServerName:      dto.ServerName,
		StartedAt:       created,
		CreatedAt:       created,
		UpdatedAt:       updated,
		// Logs stay unparsed here on purpose: list endpoints carry the full
		// build log of every deployment, and nothing reads it from a list.
		// DeploymentLogs parses the one deployment actually opened.
	}
	if !deployment.Status.IsActive() && !updated.IsZero() {
		deployment.FinishedAt = &updated
	}
	return deployment
}

func deploymentTrigger(dto deploymentDTO) string {
	switch {
	case dto.IsWebhook:
		return "webhook"
	case dto.IsAPI:
		return "api"
	default:
		return "manual"
	}
}

func parseDeploymentLogs(raw string) []domain.LogLine {
	if strings.TrimSpace(raw) == "" {
		return []domain.LogLine{}
	}
	items := []deploymentLogDTO{}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		lines, _ := domain.ParseLogPayload(raw, 20_000)
		return lines
	}
	lines := make([]domain.LogLine, 0, len(items))
	for _, item := range items {
		line := domain.ParseLogLine(item.Output)
		line.Stream = item.Type
		if timestamp := parseTime(item.Timestamp); !timestamp.IsZero() {
			line.Timestamp = timestamp
		}
		lines = append(lines, line)
	}
	return lines
}

func parseTime(raw string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if value, err := time.Parse(layout, raw); err == nil {
			return value
		}
	}
	return time.Time{}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

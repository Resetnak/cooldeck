package coolify

import "encoding/json"

type applicationDTO struct {
	ID                  int     `json:"id"`
	UUID                string  `json:"uuid"`
	Name                string  `json:"name"`
	Description         *string `json:"description"`
	FQDN                *string `json:"fqdn"`
	GitRepository       string  `json:"git_repository"`
	GitBranch           string  `json:"git_branch"`
	GitCommitSHA        string  `json:"git_commit_sha"`
	BuildPack           string  `json:"build_pack"`
	Status              string  `json:"status"`
	HealthCheckEnabled  bool    `json:"health_check_enabled"`
	HealthCheckMethod   string  `json:"health_check_method"`
	HealthCheckScheme   string  `json:"health_check_scheme"`
	HealthCheckHost     string  `json:"health_check_host"`
	HealthCheckPort     string  `json:"health_check_port"`
	HealthCheckPath     string  `json:"health_check_path"`
	HealthCheckInterval int     `json:"health_check_interval"`
	HealthCheckTimeout  int     `json:"health_check_timeout"`
	HealthCheckRetries  int     `json:"health_check_retries"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type deploymentDTO struct {
	DeploymentUUID  string `json:"deployment_uuid"`
	ApplicationID   string `json:"application_id"`
	ApplicationName string `json:"application_name"`
	Status          string `json:"status"`
	Commit          string `json:"commit"`
	CommitMessage   string `json:"commit_message"`
	IsWebhook       bool   `json:"is_webhook"`
	IsAPI           bool   `json:"is_api"`
	ForceRebuild    bool   `json:"force_rebuild"`
	Rollback        bool   `json:"rollback"`
	RestartOnly     bool   `json:"restart_only"`
	ServerName      string `json:"server_name"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	Logs            string `json:"logs"`
}

// deploymentListDTO accepts every shape Coolify uses for a deployment list:
// a bare array, the {count, deployments} wrapper of the per-application
// history endpoint, and the keyed object json_encode produces when the
// running-queue endpoint serialises a key-preserving Laravel collection that
// sortBy has reordered.
type deploymentListDTO []deploymentDTO

func (l *deploymentListDTO) UnmarshalJSON(data []byte) error {
	var plain []deploymentDTO
	if err := json.Unmarshal(data, &plain); err == nil {
		*l = plain
		return nil
	}
	var wrapped struct {
		Deployments []deploymentDTO `json:"deployments"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && wrapped.Deployments != nil {
		*l = wrapped.Deployments
		return nil
	}
	var keyed map[string]deploymentDTO
	if err := json.Unmarshal(data, &keyed); err != nil {
		return err
	}
	// Map order is irrelevant: mapDeployments re-sorts newest first anyway.
	items := make([]deploymentDTO, 0, len(keyed))
	for _, item := range keyed {
		items = append(items, item)
	}
	*l = items
	return nil
}

type teamDTO struct {
	Name string `json:"name"`
}

type logsDTO struct {
	Logs string `json:"logs"`
}

type operationDTO struct {
	Message        string `json:"message"`
	DeploymentUUID string `json:"deployment_uuid"`
}

type deployResponseDTO struct {
	Deployments []struct {
		Message        string `json:"message"`
		ResourceUUID   string `json:"resource_uuid"`
		DeploymentUUID string `json:"deployment_uuid"`
	} `json:"deployments"`
}

type deploymentLogDTO struct {
	Output    string `json:"output"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
}

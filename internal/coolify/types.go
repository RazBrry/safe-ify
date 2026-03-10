package coolify

import "fmt"

// Application represents a Coolify application resource.
type Application struct {
	UUID          string `json:"uuid"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	FQDN          string `json:"fqdn"`
	Status        string `json:"status"`
	BuildPack     string `json:"build_pack"`
	GitRepository string `json:"git_repository"`
	GitBranch     string `json:"git_branch"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// DeploymentEntry represents a single deployment entry in a deploy response.
type DeploymentEntry struct {
	Message        string `json:"message"`
	ResourceUUID   string `json:"resource_uuid"`
	DeploymentUUID string `json:"deployment_uuid"`
}

// DeployResponse is returned by the Deploy API call.
type DeployResponse struct {
	Deployments []DeploymentEntry `json:"deployments"`
}

// LogsResponse holds log lines returned by the logs API.
type LogsResponse struct {
	Lines []string
}

// CoolifyError represents an error returned by the Coolify API.
type CoolifyError struct {
	StatusCode int
	Message    string
	Raw        string
}

// Error implements the error interface.
func (e *CoolifyError) Error() string {
	return fmt.Sprintf("coolify API error %d: %s", e.StatusCode, e.Message)
}

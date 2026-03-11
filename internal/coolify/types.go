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

// Deployment represents a single deployment with its status.
type Deployment struct {
	DeploymentUUID string `json:"deployment_uuid"`
	Status         string `json:"status"`
	CommitSHA      string `json:"commit_sha,omitempty"`
	CommitMessage  string `json:"commit_message,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// Resource represents resource usage metrics for an application.
type Resource struct {
	ContainerID string `json:"container_id,omitempty"`
	CPUPercent  string `json:"cpu_percent,omitempty"`
	MemUsage    string `json:"mem_usage,omitempty"`
	MemLimit    string `json:"mem_limit,omitempty"`
	MemPercent  string `json:"mem_percent,omitempty"`
	NetIO       string `json:"net_io,omitempty"`
	BlockIO     string `json:"block_io,omitempty"`
}

// EnvVar represents a Coolify environment variable.
type EnvVar struct {
	UUID        string `json:"uuid"`
	Key         string `json:"key"`
	Value       string `json:"value,omitempty"`
	RealValue   string `json:"real_value,omitempty"`
	Comment     string `json:"comment"`
	IsPreview   bool   `json:"is_preview"`
	IsMultiline bool   `json:"is_multiline"`
	IsLiteral   bool   `json:"is_literal"`
	IsRuntime   bool   `json:"is_runtime"`
	IsBuildtime bool   `json:"is_buildtime"`
	IsShownOnce bool   `json:"is_shown_once"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// CreateEnvRequest is the body for POST /applications/{uuid}/envs.
type CreateEnvRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	IsPreview   *bool  `json:"is_preview,omitempty"`
	IsRuntime   *bool  `json:"is_runtime,omitempty"`
	IsBuildtime *bool  `json:"is_buildtime,omitempty"`
}

// UpdateEnvRequest is the body for PATCH /applications/{uuid}/envs.
// Coolify looks up the env var by key (not by UUID in path).
type UpdateEnvRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	IsPreview   *bool  `json:"is_preview,omitempty"`
	IsRuntime   *bool  `json:"is_runtime,omitempty"`
	IsBuildtime *bool  `json:"is_buildtime,omitempty"`
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

// NetworkError represents a transport-level failure (connection refused, timeout, DNS, etc.)
// that occurred before a response was received from the Coolify API.
type NetworkError struct {
	Cause error
}

// Error implements the error interface.
func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error: %s", e.Cause)
}

// Unwrap returns the underlying cause for errors.Is/errors.As support.
func (e *NetworkError) Unwrap() error {
	return e.Cause
}

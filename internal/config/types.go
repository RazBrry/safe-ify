package config

// GlobalConfig represents the top-level global configuration stored at
// ~/.config/safe-ify/config.yaml with 0600 permissions.
type GlobalConfig struct {
	Instances map[string]Instance `yaml:"instances"`
	Defaults  DefaultSettings     `yaml:"defaults"`
}

// Instance holds the URL and API token for a single Coolify instance.
type Instance struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

// DefaultSettings holds global default settings applied to all projects.
type DefaultSettings struct {
	Permissions PermissionConfig `yaml:"permissions"`
}

// PermissionConfig holds the deny list for a permission scope (global or project).
type PermissionConfig struct {
	Deny []string `yaml:"deny"`
}

// AppConfig holds the configuration for a single named app within a project.
type AppConfig struct {
	UUID        string           `yaml:"uuid"`
	Permissions PermissionConfig `yaml:"permissions"`
}

// ProjectConfig represents a per-project config stored in .safe-ify.yaml.
// Supports two formats:
//   - Legacy: Instance + AppUUID + Permissions (single app)
//   - Multi:  Instance + Apps map + Permissions (project-level deny)
//
// This file contains no secrets and can be committed to version control.
type ProjectConfig struct {
	Instance    string               `yaml:"instance"`
	AppUUID     string               `yaml:"app_uuid,omitempty"`     // legacy single-app
	Apps        map[string]AppConfig `yaml:"apps,omitempty"`         // multi-app
	Permissions PermissionConfig     `yaml:"permissions"`
}

// RuntimeConfig is the resolved configuration for a single command invocation.
// It is computed from the global config and project config.
type RuntimeConfig struct {
	InstanceName string
	InstanceURL  string
	Token        string
	AppUUID      string
	AppName      string          // the selected app's name
	AllowedCmds  map[string]bool // computed from global + project + app deny lists
}

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

const projectConfigFilename = ".safe-ify.yaml"

// validAppName is the accepted pattern for app map keys.
var validAppName = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// knownAgentCommands is the authoritative list of agent-facing commands.
// Keep in sync with permissions.AllAgentCommands.
var knownAgentCommands = map[string]bool{
	"deploy":    true,
	"redeploy":  true,
	"logs":      true,
	"status":    true,
	"list":      true,
	"env-read":  true,
	"env-write": true,
}

// validateDenyList returns an error if any entry in deny is not a known agent command.
func validateDenyList(deny []string) error {
	for _, cmd := range deny {
		if !knownAgentCommands[cmd] {
			return fmt.Errorf("unknown command in deny list: %q", cmd)
		}
	}
	return nil
}

// LoadProject reads and parses the project config file at path.
// It auto-detects legacy (app_uuid) and multi-app (apps) formats,
// normalises legacy configs to the internal multi-app representation,
// and validates all fields.
func LoadProject(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ProjectConfigNotFoundError{SearchRoot: filepath.Dir(path)}
		}
		return nil, fmt.Errorf("cannot read project config: %w", err)
	}

	var cfg ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse project config: %w", err)
	}

	if cfg.Instance == "" {
		return nil, fmt.Errorf("project config %q: 'instance' field is required", path)
	}

	// Format detection (D2, D5).
	hasLegacy := cfg.AppUUID != ""
	hasMulti := len(cfg.Apps) > 0

	switch {
	case hasLegacy && hasMulti:
		return nil, fmt.Errorf("config has both 'app_uuid' and 'apps'; use only one format")

	case hasLegacy:
		// Legacy format: normalise to single-entry Apps map.
		cfg.Apps = map[string]AppConfig{
			"default": {
				UUID:        cfg.AppUUID,
				Permissions: PermissionConfig{Deny: []string{}},
			},
		}
		cfg.AppUUID = ""

	case hasMulti:
		// Multi-app format: validate each app entry.
		for name, app := range cfg.Apps {
			if !validAppName.MatchString(name) {
				return nil, fmt.Errorf(
					"project config %q: invalid app name %q (must match ^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$)",
					path, name,
				)
			}
			if app.UUID == "" {
				return nil, fmt.Errorf(
					"project config %q: app %q has an empty uuid",
					path, name,
				)
			}
			if err := validateDenyList(app.Permissions.Deny); err != nil {
				return nil, fmt.Errorf(
					"project config %q: app %q deny list: %w",
					path, name, err,
				)
			}
		}

	default:
		return nil, fmt.Errorf("project config %q: no apps configured", path)
	}

	// Validate project-level deny list.
	if err := validateDenyList(cfg.Permissions.Deny); err != nil {
		return nil, fmt.Errorf("project config %q: project deny list: %w", path, err)
	}

	return &cfg, nil
}

// FindProjectConfig traverses parent directories starting from startDir,
// looking for a .safe-ify.yaml file. It returns the full path to the first
// file found, or ProjectConfigNotFoundError if none is found.
func FindProjectConfig(startDir string) (string, error) {
	dir := startDir
	for {
		candidate := filepath.Join(dir, projectConfigFilename)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("cannot stat %q: %w", candidate, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding the config.
			break
		}
		dir = parent
	}

	return "", &ProjectConfigNotFoundError{SearchRoot: startDir}
}

// SaveProject marshals cfg to YAML and writes it to path with 0644 permissions.
// The file contains no secrets and can be committed to version control.
// It always writes the multi-app format; the legacy app_uuid field is omitted
// because it is cleared during LoadProject.
func SaveProject(path string, cfg *ProjectConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("cannot marshal project config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("cannot write project config: %w", err)
	}

	return nil
}

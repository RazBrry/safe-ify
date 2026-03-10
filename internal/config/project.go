package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const projectConfigFilename = ".safe-ify.yaml"

// LoadProject reads and parses the project config file at path.
// It validates that the instance name and app_uuid fields are not empty.
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
	if cfg.AppUUID == "" {
		return nil, fmt.Errorf("project config %q: 'app_uuid' field is required", path)
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

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultGlobalConfigPath returns the default path to the global config file:
// ~/.config/safe-ify/config.yaml
func DefaultGlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "safe-ify", "config.yaml"), nil
}

// CheckPermissions validates that:
//   - the file exists
//   - the file mode is no more permissive than 0600 (no group/other bits set)
//   - the parent directory mode is no more permissive than 0700
func CheckPermissions(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ConfigNotFoundError{Path: path}
		}
		return fmt.Errorf("cannot stat config file: %w", err)
	}

	// File must be no more permissive than 0600.
	if info.Mode().Perm()&0o077 != 0 {
		return &ConfigInsecureError{
			Path:     path,
			Current:  info.Mode().Perm(),
			Expected: os.FileMode(0o600),
		}
	}

	// Check parent directory.
	dir := filepath.Dir(path)
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("cannot stat config directory: %w", err)
	}
	if dirInfo.Mode().Perm()&0o077 != 0 {
		return &ConfigInsecureError{
			Path:     dir,
			Current:  dirInfo.Mode().Perm(),
			Expected: os.FileMode(0o700),
		}
	}

	return nil
}

// LoadGlobal reads and parses the global config file at path.
// It checks file permissions before reading; returns ConfigInsecureError if
// the file is more permissive than 0600.
func LoadGlobal(path string) (*GlobalConfig, error) {
	if err := CheckPermissions(path); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read global config: %w", err)
	}

	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse global config: %w", err)
	}

	if cfg.Instances == nil {
		cfg.Instances = make(map[string]Instance)
	}

	return &cfg, nil
}

// SaveGlobal marshals cfg to YAML and writes it to path with 0600 permissions.
// The parent directory is created with 0700 permissions if it does not exist.
func SaveGlobal(path string, cfg *GlobalConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("cannot marshal global config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("cannot write global config: %w", err)
	}

	return nil
}

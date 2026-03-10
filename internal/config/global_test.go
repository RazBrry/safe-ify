package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadGlobal_ValidConfig creates a temp file with valid YAML, sets
// permissions to 0600, loads the config and verifies the struct fields.
func TestLoadGlobal_ValidConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// TempDir creates directories with 0700 by default on most systems,
	// but explicitly set it to 0700 to satisfy CheckPermissions.
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}

	cfgPath := filepath.Join(dir, "config.yaml")
	content := []byte(`instances:
  my-coolify:
    url: https://coolify.example.com
    token: supersecrettoken123
defaults:
  permissions:
    deny: []
`)
	if err := os.WriteFile(cfgPath, content, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cfg, err := LoadGlobal(cfgPath)
	if err != nil {
		t.Fatalf("LoadGlobal returned unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadGlobal returned nil config")
	}

	inst, ok := cfg.Instances["my-coolify"]
	if !ok {
		t.Fatalf("expected instance %q in config, got: %v", "my-coolify", cfg.Instances)
	}
	if inst.URL != "https://coolify.example.com" {
		t.Errorf("URL: got %q, want %q", inst.URL, "https://coolify.example.com")
	}
	if inst.Token != "supersecrettoken123" {
		t.Errorf("Token: got %q, want %q", inst.Token, "supersecrettoken123")
	}
}

// TestLoadGlobal_MissingFile verifies that a non-existent path returns
// ConfigNotFoundError.
func TestLoadGlobal_MissingFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")

	_, err := LoadGlobal(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound *ConfigNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected *ConfigNotFoundError, got %T: %v", err, err)
	}
	if notFound.Path != path {
		t.Errorf("ConfigNotFoundError.Path: got %q, want %q", notFound.Path, path)
	}
}

// TestLoadGlobal_InsecurePermissions creates a temp file with 0644 permissions
// and verifies that LoadGlobal returns ConfigInsecureError.
func TestLoadGlobal_InsecurePermissions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}

	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("instances: {}\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := LoadGlobal(cfgPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var insecure *ConfigInsecureError
	if !errors.As(err, &insecure) {
		t.Errorf("expected *ConfigInsecureError, got %T: %v", err, err)
	}
	if insecure.Path != cfgPath {
		t.Errorf("ConfigInsecureError.Path: got %q, want %q", insecure.Path, cfgPath)
	}
	if insecure.Current != 0o644 {
		t.Errorf("ConfigInsecureError.Current: got %04o, want %04o", insecure.Current, 0o644)
	}
	if insecure.Expected != 0o600 {
		t.Errorf("ConfigInsecureError.Expected: got %04o, want %04o", insecure.Expected, 0o600)
	}
}

// TestSaveGlobal_CreatesWithCorrectPermissions saves a config to a temp dir
// and verifies the resulting file has 0600 permissions.
func TestSaveGlobal_CreatesWithCorrectPermissions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := &GlobalConfig{
		Instances: map[string]Instance{
			"test": {URL: "https://example.com", Token: "tok123"},
		},
	}

	if err := SaveGlobal(cfgPath, cfg); err != nil {
		t.Fatalf("SaveGlobal returned unexpected error: %v", err)
	}

	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}

	got := info.Mode().Perm()
	if got != 0o600 {
		t.Errorf("file permissions: got %04o, want %04o", got, 0o600)
	}
}

// TestSaveGlobal_CreatesDirectory saves a config to a non-existent directory
// and verifies that SaveGlobal creates the directory with 0700 permissions.
func TestSaveGlobal_CreatesDirectory(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	newDir := filepath.Join(base, "new-config-dir")
	cfgPath := filepath.Join(newDir, "config.yaml")

	cfg := &GlobalConfig{
		Instances: make(map[string]Instance),
	}

	if err := SaveGlobal(cfgPath, cfg); err != nil {
		t.Fatalf("SaveGlobal returned unexpected error: %v", err)
	}

	// Verify directory was created.
	dirInfo, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("stat directory: %v", err)
	}
	if !dirInfo.IsDir() {
		t.Errorf("expected %q to be a directory", newDir)
	}

	dirPerm := dirInfo.Mode().Perm()
	if dirPerm != 0o700 {
		t.Errorf("directory permissions: got %04o, want %04o", dirPerm, 0o700)
	}

	// Verify file was created.
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
}

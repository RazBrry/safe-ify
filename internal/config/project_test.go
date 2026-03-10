package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadProject_ValidConfig creates a temp file with valid project YAML,
// loads it, and verifies the struct fields.
func TestLoadProject_ValidConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".safe-ify.yaml")
	content := []byte(`instance: my-coolify
app_uuid: hgkks00abc123
permissions:
  deny:
    - deploy
`)
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cfg, err := LoadProject(cfgPath)
	if err != nil {
		t.Fatalf("LoadProject returned unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadProject returned nil config")
	}
	if cfg.Instance != "my-coolify" {
		t.Errorf("Instance: got %q, want %q", cfg.Instance, "my-coolify")
	}
	if cfg.AppUUID != "hgkks00abc123" {
		t.Errorf("AppUUID: got %q, want %q", cfg.AppUUID, "hgkks00abc123")
	}
	if len(cfg.Permissions.Deny) != 1 || cfg.Permissions.Deny[0] != "deploy" {
		t.Errorf("Permissions.Deny: got %v, want [deploy]", cfg.Permissions.Deny)
	}
}

// TestFindProjectConfig_CurrentDir creates a .safe-ify.yaml in a temp dir and
// verifies that FindProjectConfig finds it when starting from the same dir.
func TestFindProjectConfig_CurrentDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".safe-ify.yaml")
	if err := os.WriteFile(cfgPath, []byte("instance: x\napp_uuid: y\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	found, err := FindProjectConfig(dir)
	if err != nil {
		t.Fatalf("FindProjectConfig returned unexpected error: %v", err)
	}
	if found != cfgPath {
		t.Errorf("found path: got %q, want %q", found, cfgPath)
	}
}

// TestFindProjectConfig_ParentDir creates a .safe-ify.yaml in a parent temp
// dir and verifies that FindProjectConfig finds it when starting from a child.
func TestFindProjectConfig_ParentDir(t *testing.T) {
	t.Parallel()

	parentDir := t.TempDir()
	childDir := filepath.Join(parentDir, "subdir")
	if err := os.MkdirAll(childDir, 0o700); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}

	cfgPath := filepath.Join(parentDir, ".safe-ify.yaml")
	if err := os.WriteFile(cfgPath, []byte("instance: x\napp_uuid: y\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	found, err := FindProjectConfig(childDir)
	if err != nil {
		t.Fatalf("FindProjectConfig returned unexpected error: %v", err)
	}
	if found != cfgPath {
		t.Errorf("found path: got %q, want %q", found, cfgPath)
	}
}

// TestFindProjectConfig_NotFound verifies that FindProjectConfig returns
// ProjectConfigNotFoundError when no .safe-ify.yaml exists in the tree.
func TestFindProjectConfig_NotFound(t *testing.T) {
	t.Parallel()

	// Use an isolated temp dir tree with no .safe-ify.yaml file anywhere.
	rootDir := t.TempDir()
	leafDir := filepath.Join(rootDir, "a", "b", "c")
	if err := os.MkdirAll(leafDir, 0o700); err != nil {
		t.Fatalf("mkdir leaf: %v", err)
	}

	_, err := FindProjectConfig(leafDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var notFound *ProjectConfigNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected *ProjectConfigNotFoundError, got %T: %v", err, err)
	}
	if notFound.SearchRoot != leafDir {
		t.Errorf("SearchRoot: got %q, want %q", notFound.SearchRoot, leafDir)
	}
}

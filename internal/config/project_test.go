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
	// Legacy app_uuid is normalised to Apps["default"] after loading.
	if cfg.AppUUID != "" {
		t.Errorf("AppUUID should be cleared after normalisation, got %q", cfg.AppUUID)
	}
	defaultApp, ok := cfg.Apps["default"]
	if !ok {
		t.Fatal("expected Apps[\"default\"] after legacy normalisation")
	}
	if defaultApp.UUID != "hgkks00abc123" {
		t.Errorf("Apps[default].UUID: got %q, want %q", defaultApp.UUID, "hgkks00abc123")
	}
	if len(cfg.Permissions.Deny) != 1 || cfg.Permissions.Deny[0] != "deploy" {
		t.Errorf("Permissions.Deny: got %v, want [deploy]", cfg.Permissions.Deny)
	}
}

// TestLoadProject_MultiApp verifies that a multi-app YAML loads correctly,
// the Apps map is populated, and AppUUID is empty (not set in multi-app format).
func TestLoadProject_MultiApp(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".safe-ify.yaml")
	content := []byte(`instance: my-coolify
apps:
  api:
    uuid: api-uuid-001
    permissions:
      deny: []
  worker:
    uuid: worker-uuid-002
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
	if cfg.AppUUID != "" {
		t.Errorf("AppUUID should be empty in multi-app format, got %q", cfg.AppUUID)
	}
	if len(cfg.Apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(cfg.Apps))
	}
	api, ok := cfg.Apps["api"]
	if !ok {
		t.Fatal("expected Apps[\"api\"] to exist")
	}
	if api.UUID != "api-uuid-001" {
		t.Errorf("Apps[api].UUID: got %q, want %q", api.UUID, "api-uuid-001")
	}
	worker, ok := cfg.Apps["worker"]
	if !ok {
		t.Fatal("expected Apps[\"worker\"] to exist")
	}
	if worker.UUID != "worker-uuid-002" {
		t.Errorf("Apps[worker].UUID: got %q, want %q", worker.UUID, "worker-uuid-002")
	}
}

// TestLoadProject_Legacy verifies that a legacy single-app YAML (with app_uuid)
// is normalised to an Apps map with key "default".
func TestLoadProject_Legacy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".safe-ify.yaml")
	content := []byte(`instance: my-coolify
app_uuid: legacy-uuid-abc
`)
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	cfg, err := LoadProject(cfgPath)
	if err != nil {
		t.Fatalf("LoadProject returned unexpected error: %v", err)
	}
	if cfg.AppUUID != "" {
		t.Errorf("AppUUID should be cleared after normalisation, got %q", cfg.AppUUID)
	}
	defaultApp, ok := cfg.Apps["default"]
	if !ok {
		t.Fatal("expected Apps[\"default\"] after legacy normalisation")
	}
	if defaultApp.UUID != "legacy-uuid-abc" {
		t.Errorf("Apps[default].UUID: got %q, want %q", defaultApp.UUID, "legacy-uuid-abc")
	}
}

// TestLoadProject_BothFormats verifies that a YAML containing both app_uuid and
// apps fields returns an error.
func TestLoadProject_BothFormats(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".safe-ify.yaml")
	content := []byte(`instance: my-coolify
app_uuid: legacy-uuid-abc
apps:
  api:
    uuid: api-uuid-001
`)
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := LoadProject(cfgPath)
	if err == nil {
		t.Fatal("expected error for config with both app_uuid and apps, got nil")
	}
}

// TestLoadProject_EmptyApps verifies that a YAML with an empty apps: map returns
// an error (neither legacy nor valid multi-app format).
func TestLoadProject_EmptyApps(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".safe-ify.yaml")
	// YAML parses an empty map (apps: {}) as len==0, which hits the default case.
	content := []byte(`instance: my-coolify
apps: {}
`)
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := LoadProject(cfgPath)
	if err == nil {
		t.Fatal("expected error for config with empty apps map, got nil")
	}
}

// TestLoadProject_InvalidAppName verifies that an app name containing spaces or
// special characters is rejected.
func TestLoadProject_InvalidAppName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".safe-ify.yaml")
	// App name "my app" contains a space, which is invalid.
	content := []byte(`instance: my-coolify
apps:
  "my app":
    uuid: some-uuid-123
`)
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := LoadProject(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid app name containing space, got nil")
	}
}

// TestLoadProject_MissingAppUUID verifies that an app entry with an empty uuid
// field returns an error.
func TestLoadProject_MissingAppUUID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".safe-ify.yaml")
	content := []byte(`instance: my-coolify
apps:
  api:
    uuid: ""
`)
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := LoadProject(cfgPath)
	if err == nil {
		t.Fatal("expected error for app with empty uuid, got nil")
	}
}

// TestLoadProject_InvalidAppDeny verifies that a multi-app config with an unknown
// command in an app's deny list returns an error.
func TestLoadProject_InvalidAppDeny(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".safe-ify.yaml")
	content := []byte(`instance: my-coolify
apps:
  api:
    uuid: api-uuid-001
    permissions:
      deny:
        - delete
`)
	if err := os.WriteFile(cfgPath, content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := LoadProject(cfgPath)
	if err == nil {
		t.Fatal("expected error for unknown command \"delete\" in app deny list, got nil")
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

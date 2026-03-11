package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// testEnv holds temporary config files and an httptest server for CLI integration tests.
type testEnv struct {
	globalConfigPath  string
	projectConfigPath string
	server            *httptest.Server
	t                 *testing.T
}

// setupTestEnv creates temporary global and project config files pointing to
// the given httptest server URL. The caller must call env.Close() when done.
func setupTestEnv(t *testing.T, srv *httptest.Server) *testEnv {
	t.Helper()

	dir := t.TempDir()

	// Write global config (must be 0600).
	globalPath := filepath.Join(dir, "config.yaml")
	globalContent := fmt.Sprintf(`instances:
  test-instance:
    url: %s
    token: test-token-abc
defaults:
  permissions:
    deny: []
`, srv.URL)
	if err := os.WriteFile(globalPath, []byte(globalContent), 0o600); err != nil {
		t.Fatalf("writing global config: %v", err)
	}
	// Ensure the parent dir also has safe permissions.
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod temp dir: %v", err)
	}

	// Write project config (0644 is fine for project files).
	projectPath := filepath.Join(dir, ".safe-ify.yaml")
	projectContent := `instance: test-instance
app_uuid: a1b2c3d4-e5f6-7890-abcd-ef1234567890
permissions:
  deny: []
`
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o644); err != nil {
		t.Fatalf("writing project config: %v", err)
	}

	return &testEnv{
		globalConfigPath:  globalPath,
		projectConfigPath: projectPath,
		server:            srv,
		t:                 t,
	}
}

// setupTestEnvDenyCommand creates a test environment where the project config
// denies the given command. It writes the project config directly (bypassing
// the base helper) so there is no duplicate YAML key.
func setupTestEnvDenyCommand(t *testing.T, srv *httptest.Server, command string) *testEnv {
	t.Helper()

	dir := t.TempDir()

	// Write global config (must be 0600).
	globalPath := filepath.Join(dir, "config.yaml")
	globalContent := fmt.Sprintf(`instances:
  test-instance:
    url: %s
    token: test-token-abc
defaults:
  permissions:
    deny: []
`, srv.URL)
	if err := os.WriteFile(globalPath, []byte(globalContent), 0o600); err != nil {
		t.Fatalf("writing global config: %v", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod temp dir: %v", err)
	}

	// Write project config that explicitly denies the given command.
	projectPath := filepath.Join(dir, ".safe-ify.yaml")
	projectContent := fmt.Sprintf(`instance: test-instance
app_uuid: a1b2c3d4-e5f6-7890-abcd-ef1234567890
permissions:
  deny:
  - %s
`, command)
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o644); err != nil {
		t.Fatalf("writing project config: %v", err)
	}

	return &testEnv{
		globalConfigPath:  globalPath,
		projectConfigPath: projectPath,
		server:            srv,
		t:                 t,
	}
}

// Close shuts down the httptest server.
func (e *testEnv) Close() {
	e.server.Close()
}

// runCommand executes a subcommand against rootCmd with --json, --config, and --project
// flags set to the test environment paths. It returns the captured stdout and any error.
// stderr is captured separately so that Cobra usage output does not contaminate the JSON.
// SilenceUsage is temporarily enabled to prevent Cobra from appending help text to stdout
// when a command returns an error.
func runCommand(t *testing.T, env *testEnv, args ...string) (string, error) {
	t.Helper()

	// Build the full argument list: subcommand + flags that wire up test configs.
	fullArgs := append(
		args,
		"--json",
		"--config", env.globalConfigPath,
		"--project", env.projectConfigPath,
	)

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs(fullArgs)

	// Suppress Cobra's automatic usage printing on error so that only the JSON
	// envelope is written to stdout. Restore after the call.
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	defer func() {
		rootCmd.SilenceUsage = false
		rootCmd.SilenceErrors = false
	}()

	err := rootCmd.Execute()
	return stdout.String(), err
}

// parseResponse unmarshals the JSON output into a generic map so tests can
// inspect the envelope fields without depending on private structs.
func parseResponse(t *testing.T, output string) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output %q: %v", output, err)
	}
	return result
}

// assertOK checks that the response envelope has ok=true.
func assertOK(t *testing.T, resp map[string]interface{}) {
	t.Helper()
	ok, _ := resp["ok"].(bool)
	if !ok {
		t.Errorf("expected ok=true, got: %v", resp)
	}
}

// assertError checks that the response envelope has ok=false and the error code matches.
func assertError(t *testing.T, resp map[string]interface{}, wantCode string) {
	t.Helper()
	ok, _ := resp["ok"].(bool)
	if ok {
		t.Errorf("expected ok=false, got ok=true in response: %v", resp)
		return
	}
	errObj, _ := resp["error"].(map[string]interface{})
	if errObj == nil {
		t.Fatalf("expected error object, got nil; full response: %v", resp)
	}
	code, _ := errObj["code"].(string)
	if code != wantCode {
		t.Errorf("expected error code %q, got %q", wantCode, code)
	}
}

// ── Test: deploy success ─────────────────────────────────────────────────────

func TestDeploy_JSON_Success(t *testing.T) {
	deployResp := `{"deployments":[{"message":"Deployment request queued.","resource_uuid":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","deployment_uuid":"dl8k4s0"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/deploy" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, deployResp)
	}))
	env := setupTestEnv(t, srv)
	defer env.Close()

	out, _ := runCommand(t, env, "deploy")
	resp := parseResponse(t, out)

	assertOK(t, resp)

	data, _ := resp["data"].(map[string]interface{})
	if data == nil {
		t.Fatal("expected data object in response")
	}
	deploymentUUID, _ := data["deployment_uuid"].(string)
	if deploymentUUID != "dl8k4s0" {
		t.Errorf("expected deployment_uuid=dl8k4s0, got %q", deploymentUUID)
	}
	message, _ := data["message"].(string)
	if message == "" {
		t.Error("expected non-empty message in data")
	}
}

// ── Test: deploy permission denied ───────────────────────────────────────────

func TestDeploy_JSON_PermissionDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should never be called; permission check happens before the API call.
		t.Error("API server should not be called when permission is denied")
		w.WriteHeader(http.StatusOK)
	}))
	env := setupTestEnvDenyCommand(t, srv, "deploy")
	defer env.Close()

	out, _ := runCommand(t, env, "deploy")
	resp := parseResponse(t, out)

	assertError(t, resp, ErrCodePermissionDenied)
}

// ── Test: redeploy success ────────────────────────────────────────────────────

func TestRedeploy_JSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		// URL should be /api/v1/applications/<uuid>/restart
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"message":"Restart triggered."}`)
	}))
	env := setupTestEnv(t, srv)
	defer env.Close()

	out, _ := runCommand(t, env, "redeploy")
	resp := parseResponse(t, out)

	assertOK(t, resp)

	data, _ := resp["data"].(map[string]interface{})
	if data == nil {
		t.Fatal("expected data object in response")
	}
	message, _ := data["message"].(string)
	if message == "" {
		t.Error("expected non-empty message in data")
	}
}

// ── Test: logs success ────────────────────────────────────────────────────────

func TestLogs_JSON_Success(t *testing.T) {
	logLines := []string{
		"2026-03-11T10:00:00Z INFO starting",
		"2026-03-11T10:00:01Z INFO ready",
		"2026-03-11T10:00:02Z INFO request received",
	}
	var capturedTail string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTail = r.URL.Query().Get("tail")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		data, _ := json.Marshal(logLines)
		w.Write(data) //nolint:errcheck
	}))
	env := setupTestEnv(t, srv)
	defer env.Close()

	out, _ := runCommand(t, env, "logs", "--tail", "50")
	resp := parseResponse(t, out)

	assertOK(t, resp)

	data, _ := resp["data"].(map[string]interface{})
	if data == nil {
		t.Fatal("expected data object in response")
	}
	count, _ := data["count"].(float64)
	if int(count) != 3 {
		t.Errorf("expected count=3, got %v", count)
	}
	lines, _ := data["lines"].([]interface{})
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if capturedTail != "50" {
		t.Errorf("expected tail query parameter=50, got %q", capturedTail)
	}
}

// ── Test: status success ──────────────────────────────────────────────────────

func TestStatus_JSON_Success(t *testing.T) {
	appJSON := `{"uuid":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","name":"my-app","status":"running","fqdn":"https://app.example.com","updated_at":"2026-03-11T10:00:00Z"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "expected GET", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, appJSON)
	}))
	env := setupTestEnv(t, srv)
	defer env.Close()

	out, _ := runCommand(t, env, "status")
	resp := parseResponse(t, out)

	assertOK(t, resp)

	data, _ := resp["data"].(map[string]interface{})
	if data == nil {
		t.Fatal("expected data object in response")
	}
	uuid, _ := data["uuid"].(string)
	if uuid != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("expected uuid=a1b2c3d4-e5f6-7890-abcd-ef1234567890, got %q", uuid)
	}
	name, _ := data["name"].(string)
	if name != "my-app" {
		t.Errorf("expected name=my-app, got %q", name)
	}
	status, _ := data["status"].(string)
	if status != "running" {
		t.Errorf("expected status=running, got %q", status)
	}
	fqdn, _ := data["fqdn"].(string)
	if fqdn != "https://app.example.com" {
		t.Errorf("expected fqdn=https://app.example.com, got %q", fqdn)
	}
}

// ── Test: list success ────────────────────────────────────────────────────────

func TestList_JSON_Success(t *testing.T) {
	// list now reads from project config, not Coolify API.
	// setupTestEnv creates a legacy config with app_uuid a1b2c3d4-...,
	// which normalizes to Apps["default"].
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "list should not call API", http.StatusInternalServerError)
	}))
	env := setupTestEnv(t, srv)
	defer env.Close()

	out, _ := runCommand(t, env, "list")
	resp := parseResponse(t, out)

	assertOK(t, resp)

	data, _ := resp["data"].(map[string]interface{})
	if data == nil {
		t.Fatal("expected data object in response")
	}
	count, _ := data["count"].(float64)
	if int(count) != 1 {
		t.Errorf("expected count=1, got %v", count)
	}
	applications, _ := data["applications"].([]interface{})
	if len(applications) != 1 {
		t.Fatalf("expected 1 application, got %d", len(applications))
	}
	first, _ := applications[0].(map[string]interface{})
	if first["uuid"] != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("unexpected app UUID: %v", first["uuid"])
	}
	if first["name"] != "default" {
		t.Errorf("unexpected app name: %v (expected 'default' from legacy normalization)", first["name"])
	}
}

// ── Test: list with no project config ────────────────────────────────────────

// TestList_JSON_NoProjectConfig verifies that the list command returns
// CONFIG_NOT_FOUND when no project config file exists.
func TestList_JSON_NoProjectConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("API server should not be called when project config is missing")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod temp dir: %v", err)
	}

	// Write only a global config — no project config file.
	globalPath := filepath.Join(dir, "config.yaml")
	globalContent := fmt.Sprintf(`instances:
  test-instance:
    url: %s
    token: test-token-abc
defaults:
  permissions:
    deny: []
`, srv.URL)
	if err := os.WriteFile(globalPath, []byte(globalContent), 0o600); err != nil {
		t.Fatalf("writing global config: %v", err)
	}

	// Point --project at a path that does not exist.
	nonExistentProject := filepath.Join(dir, "does-not-exist.yaml")

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"list",
		"--json",
		"--config", globalPath,
		"--project", nonExistentProject,
	})
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	defer func() {
		rootCmd.SilenceUsage = false
		rootCmd.SilenceErrors = false
	}()
	_ = rootCmd.Execute()

	out := stdout.String()
	resp := parseResponse(t, out)

	assertError(t, resp, ErrCodeConfigNotFound)
}

// ── Test: network error ───────────────────────────────────────────────────────

// TestCommand_JSON_NetworkError verifies that a network-level failure (server
// closed before request) produces a JSON response with code NETWORK_ERROR.
func TestCommand_JSON_NetworkError(t *testing.T) {
	// Create a server then close it immediately so all connections fail.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srvURL := srv.URL
	srv.Close()

	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod temp dir: %v", err)
	}

	globalPath := filepath.Join(dir, "config.yaml")
	globalContent := fmt.Sprintf(`instances:
  test-instance:
    url: %s
    token: test-token-abc
defaults:
  permissions:
    deny: []
`, srvURL)
	if err := os.WriteFile(globalPath, []byte(globalContent), 0o600); err != nil {
		t.Fatalf("writing global config: %v", err)
	}

	projectPath := filepath.Join(dir, ".safe-ify.yaml")
	projectContent := `instance: test-instance
app_uuid: a1b2c3d4-e5f6-7890-abcd-ef1234567890
permissions:
  deny: []
`
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o644); err != nil {
		t.Fatalf("writing project config: %v", err)
	}

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"status",
		"--json",
		"--config", globalPath,
		"--project", projectPath,
	})
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	defer func() {
		rootCmd.SilenceUsage = false
		rootCmd.SilenceErrors = false
	}()
	_ = rootCmd.Execute()

	out := stdout.String()
	resp := parseResponse(t, out)

	assertError(t, resp, ErrCodeNetworkError)
}

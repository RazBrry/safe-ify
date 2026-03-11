package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RazBrry/safe-ify/internal/config"
)

// ── Multi-app test helpers ────────────────────────────────────────────────────

// UUIDs used in multi-app tests — must match the standard UUID format
// (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx) because the Coolify client validates them.
const (
	multiAppAPIUUID = "aa000000-0000-0000-0000-000000000001"
	multiAppWebUUID = "bb000000-0000-0000-0000-000000000002"
)

// setupMultiAppTestEnv creates a test environment with a multi-app project
// config (apps: api + web). The API app UUID is multiAppAPIUUID and the
// web app UUID is multiAppWebUUID.
func setupMultiAppTestEnv(t *testing.T, srv *httptest.Server) *testEnv {
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

	// Write multi-app project config.
	projectPath := filepath.Join(dir, ".safe-ify.yaml")
	projectContent := fmt.Sprintf(`instance: test-instance
apps:
  api:
    uuid: %s
    permissions:
      deny: []
  web:
    uuid: %s
    permissions:
      deny: []
permissions:
  deny: []
`, multiAppAPIUUID, multiAppWebUUID)
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

// runCommandWithApp runs a command with an explicit --app flag in addition to
// the standard test environment flags. It resets the package-level appName
// variable after execution so that subsequent tests are not affected.
func runCommandWithApp(t *testing.T, env *testEnv, selectedApp string, args ...string) (string, error) {
	t.Helper()
	// Reset appName after this call so Cobra's persistent flag doesn't bleed
	// into subsequent test cases. Since we're in the same package we can
	// access the package-level var directly.
	defer func() { appName = "" }()
	return runCommand(t, env, append([]string{"--app", selectedApp}, args...)...)
}

// ── mapConfigError tests ──────────────────────────────────────────────────────

// TestMapConfigError_AppNotFound verifies that mapConfigError maps
// *config.AppNotFoundError to ErrCodeAppNotFound ("APP_NOT_FOUND").
func TestMapConfigError_AppNotFound(t *testing.T) {
	err := &config.AppNotFoundError{
		Name:          "unknown",
		AvailableApps: []string{"api"},
	}
	got := mapConfigError(err)
	if got != ErrCodeAppNotFound {
		t.Errorf("mapConfigError(AppNotFoundError): got %q, want %q", got, ErrCodeAppNotFound)
	}
}

// TestMapConfigError_AppAmbiguous verifies that mapConfigError maps
// *config.AppAmbiguousError to ErrCodeAppAmbiguous ("APP_AMBIGUOUS").
func TestMapConfigError_AppAmbiguous(t *testing.T) {
	err := &config.AppAmbiguousError{
		AvailableApps: []string{"api", "web"},
	}
	got := mapConfigError(err)
	if got != ErrCodeAppAmbiguous {
		t.Errorf("mapConfigError(AppAmbiguousError): got %q, want %q", got, ErrCodeAppAmbiguous)
	}
}

// ── Deploy command multi-app tests ───────────────────────────────────────────

// TestDeployCmd_WithAppFlag verifies that when --app=api is passed against a
// multi-app config, the deploy request targets the correct UUID ("api-uuid-1111").
func TestDeployCmd_WithAppFlag(t *testing.T) {
	var capturedUUID string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/deploy" {
			// The UUID is sent as a query parameter "uuid".
			capturedUUID = r.URL.Query().Get("uuid")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"deployments":[{"message":"queued","resource_uuid":%q,"deployment_uuid":"dep-abc"}]}`, multiAppAPIUUID)
			return
		}
		http.Error(w, "unexpected request: "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
	}))
	env := setupMultiAppTestEnv(t, srv)
	defer env.Close()

	out, _ := runCommandWithApp(t, env, "api", "deploy")
	resp := parseResponse(t, out)

	assertOK(t, resp)

	// The UUID sent to the API must be the one configured for "api".
	if capturedUUID != multiAppAPIUUID {
		t.Errorf("expected deploy request UUID=%q, got %q", multiAppAPIUUID, capturedUUID)
	}
}

// TestDeployCmd_AmbiguousApp verifies that running deploy against a multi-app
// config WITHOUT --app returns a JSON error with code APP_AMBIGUOUS.
func TestDeployCmd_AmbiguousApp(t *testing.T) {
	// Ensure the package-level appName is cleared so no stale --app value
	// from a previous test is inherited.
	appName = ""
	defer func() { appName = "" }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should never be called; config resolution fails before the API call.
		t.Error("API server should not be called when app selection is ambiguous")
		w.WriteHeader(http.StatusOK)
	}))
	env := setupMultiAppTestEnv(t, srv)
	defer env.Close()

	// Run deploy WITHOUT --app flag.
	out, _ := runCommand(t, env, "deploy")
	resp := parseResponse(t, out)

	assertError(t, resp, ErrCodeAppAmbiguous)
}

// TestDeployCmd_AppNotFound verifies that running deploy with --app=unknown
// against a multi-app config returns a JSON error with code APP_NOT_FOUND.
func TestDeployCmd_AppNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("API server should not be called when app is not found")
		w.WriteHeader(http.StatusOK)
	}))
	env := setupMultiAppTestEnv(t, srv)
	defer env.Close()

	// Run deploy with an --app name that does not exist in the config.
	out, _ := runCommandWithApp(t, env, "unknown", "deploy")
	resp := parseResponse(t, out)

	assertError(t, resp, ErrCodeAppNotFound)
}

// ── List command multi-app test ───────────────────────────────────────────────

// TestListCmd_NoAppRequired verifies that the list command succeeds against a
// multi-app config even when --app is NOT provided. The list command bypasses
// the app-selection requirement (D6).
func TestListCmd_NoAppRequired(t *testing.T) {
	// Ensure no stale --app value is inherited from a previous test.
	appName = ""
	defer func() { appName = "" }()

	appsJSON := fmt.Sprintf(`[{"uuid":%q,"name":"api","status":"running","fqdn":""},{"uuid":%q,"name":"web","status":"running","fqdn":""}]`, multiAppAPIUUID, multiAppWebUUID)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, appsJSON)
			return
		}
		http.Error(w, "unexpected request: "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
	}))
	env := setupMultiAppTestEnv(t, srv)
	defer env.Close()

	// Run list WITHOUT --app flag on multi-app config — must NOT return APP_AMBIGUOUS.
	out, _ := runCommand(t, env, "list")
	resp := parseResponse(t, out)

	// Must succeed (ok=true), not fail with APP_AMBIGUOUS.
	assertOK(t, resp)

	// Sanity-check: response must not contain the APP_AMBIGUOUS error code.
	if strings.Contains(out, "APP_AMBIGUOUS") {
		t.Errorf("list command returned APP_AMBIGUOUS error on multi-app config without --app\n  output: %q", out)
	}

	data, _ := resp["data"].(map[string]interface{})
	if data == nil {
		t.Fatal("expected data object in response")
	}
	count, _ := data["count"].(float64)
	if int(count) != 2 {
		t.Errorf("expected count=2, got %v", count)
	}
}

// ── Doctor command multi-app test ────────────────────────────────────────────

// TestDoctorCmd_MultiApp verifies that the doctor command, when run against a
// multi-app project config, includes all configured app names in its output.
func TestDoctorCmd_MultiApp(t *testing.T) {
	// Mock server that handles all API calls made by doctor for multi-app config.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/healthcheck":
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `"v4.0.0-beta.360"`)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications/"+multiAppAPIUUID:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"uuid":%q,"name":"my-api","status":"running","fqdn":"https://api.example.com"}`, multiAppAPIUUID)

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications/"+multiAppWebUUID:
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"uuid":%q,"name":"my-web","status":"running","fqdn":"https://web.example.com"}`, multiAppWebUUID)

		default:
			http.Error(w, "unexpected request: "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	// Build multi-app config files.
	dir := t.TempDir()

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

	projectPath := filepath.Join(dir, ".safe-ify.yaml")
	projectContent := fmt.Sprintf(`instance: test-instance
apps:
  api:
    uuid: %s
    permissions:
      deny: []
  web:
    uuid: %s
    permissions:
      deny: []
permissions:
  deny: []
`, multiAppAPIUUID, multiAppWebUUID)
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o644); err != nil {
		t.Fatalf("writing project config: %v", err)
	}

	// Run doctor using the runDoctorCommand helper pattern.
	env := &doctorTestEnv{
		globalPath:  globalPath,
		projectPath: projectPath,
		server:      srv,
	}
	stdout, _ := runDoctorCommand(t, env)

	// Both app names ("api" and "web") must appear in the doctor stdout.
	if !strings.Contains(stdout, "api") {
		t.Errorf("doctor stdout does not contain app name %q\n  got: %q", "api", stdout)
	}
	if !strings.Contains(stdout, "web") {
		t.Errorf("doctor stdout does not contain app name %q\n  got: %q", "web", stdout)
	}
}

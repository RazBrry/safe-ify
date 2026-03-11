package cli

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// doctorTestEnv sets up config files and an httptest.Server for doctor tests.
// It returns the global config path, project config path, and the server.
// The caller is responsible for closing the server.
type doctorTestEnv struct {
	globalPath  string
	projectPath string
	server      *httptest.Server
}

// setupDoctorTestEnv creates a minimal but complete test environment for the doctor
// command: a valid global config (0600) with one instance pointing to the mock
// server, and a valid project config referencing that instance and a UUID.
func setupDoctorTestEnv(t *testing.T, srv *httptest.Server, projectDeny []string) *doctorTestEnv {
	t.Helper()

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

	var denyBlock string
	if len(projectDeny) > 0 {
		denyBlock = "  deny:\n"
		for _, cmd := range projectDeny {
			denyBlock += fmt.Sprintf("  - %s\n", cmd)
		}
	} else {
		denyBlock = "  deny: []\n"
	}

	projectPath := filepath.Join(dir, ".safe-ify.yaml")
	projectContent := fmt.Sprintf(`instance: test-instance
app_uuid: a1b2c3d4-e5f6-7890-abcd-ef1234567890
permissions:
%s`, denyBlock)
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o644); err != nil {
		t.Fatalf("writing project config: %v", err)
	}

	return &doctorTestEnv{
		globalPath:  globalPath,
		projectPath: projectPath,
		server:      srv,
	}
}

// runDoctorCommand executes the doctor command against rootCmd with the given
// test environment paths. It returns the captured stdout (the markdown snippet)
// and stderr (diagnostics).
func runDoctorCommand(t *testing.T, env *doctorTestEnv) (stdout, stderr string) {
	t.Helper()

	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs([]string{
		"doctor",
		"--config", env.globalPath,
		"--project", env.projectPath,
	})
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	defer func() {
		rootCmd.SilenceUsage = false
		rootCmd.SilenceErrors = false
	}()

	_ = rootCmd.Execute()

	return outBuf.String(), errBuf.String()
}

// newDoctorMockServer creates an httptest.Server that handles all API endpoints
// required by the doctor command. appName is used in the GetApplication response.
func newDoctorMockServer(t *testing.T, appName string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/healthcheck":
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `"v4.0.0-beta.360"`)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/applications/"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"uuid":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","name":%q,"status":"running","fqdn":"https://app.example.com"}`, appName)
		default:
			http.Error(w, "unexpected request: "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
		}
	}))
}

// TestDoctorOutput_ContainsMarkdownTable verifies that the doctor stdout contains
// the markdown table header "| Command | Status |".
func TestDoctorOutput_ContainsMarkdownTable(t *testing.T) {
	srv := newDoctorMockServer(t, "my-app")
	defer srv.Close()

	env := setupDoctorTestEnv(t, srv, nil)
	stdout, _ := runDoctorCommand(t, env)

	if !strings.Contains(stdout, "| Command | Status |") {
		t.Errorf("doctor stdout does not contain markdown table header\n  got: %q", stdout)
	}
	if !strings.Contains(stdout, "|---------|--------|") {
		t.Errorf("doctor stdout does not contain table separator row\n  got: %q", stdout)
	}
}

// TestDoctorOutput_ListsAllCommands verifies that all 5 agent command names
// appear in the doctor stdout.
func TestDoctorOutput_ListsAllCommands(t *testing.T) {
	srv := newDoctorMockServer(t, "my-app")
	defer srv.Close()

	env := setupDoctorTestEnv(t, srv, nil)
	stdout, _ := runDoctorCommand(t, env)

	wantCommands := []string{
		"safe-ify deploy --json",
		"safe-ify redeploy --json",
		"safe-ify logs --json --tail N",
		"safe-ify status --json",
		"safe-ify list --json",
	}

	for _, cmd := range wantCommands {
		if !strings.Contains(stdout, cmd) {
			t.Errorf("doctor stdout missing command %q\n  full output: %q", cmd, stdout)
		}
	}
}

// TestDoctorOutput_ShowsAllowedDenied verifies that when a project deny list is
// configured, denied commands show "Denied" and allowed commands show "Allowed"
// in the markdown table.
func TestDoctorOutput_ShowsAllowedDenied(t *testing.T) {
	srv := newDoctorMockServer(t, "my-app")
	defer srv.Close()

	// Deny "list" and "redeploy" at the project level.
	env := setupDoctorTestEnv(t, srv, []string{"list", "redeploy"})
	stdout, _ := runDoctorCommand(t, env)

	// "list" and "redeploy" should appear as Denied.
	// We check that their table rows contain "Denied".
	rows := strings.Split(stdout, "\n")
	for _, row := range rows {
		if strings.Contains(row, "safe-ify list --json") {
			if !strings.Contains(row, "Denied") {
				t.Errorf("expected 'list' row to show Denied, got: %q", row)
			}
		}
		if strings.Contains(row, "safe-ify redeploy --json") {
			if !strings.Contains(row, "Denied") {
				t.Errorf("expected 'redeploy' row to show Denied, got: %q", row)
			}
		}
		// "deploy", "logs", "status" should be Allowed.
		if strings.Contains(row, "safe-ify deploy --json") {
			if !strings.Contains(row, "Allowed") {
				t.Errorf("expected 'deploy' row to show Allowed, got: %q", row)
			}
		}
		if strings.Contains(row, "safe-ify logs --json") {
			if !strings.Contains(row, "Allowed") {
				t.Errorf("expected 'logs' row to show Allowed, got: %q", row)
			}
		}
		if strings.Contains(row, "safe-ify status --json") {
			if !strings.Contains(row, "Allowed") {
				t.Errorf("expected 'status' row to show Allowed, got: %q", row)
			}
		}
	}

	// Sanity-check: output must contain at least one "Denied" and one "Allowed".
	if !strings.Contains(stdout, "Denied") {
		t.Errorf("expected doctor output to contain 'Denied', full output: %q", stdout)
	}
	if !strings.Contains(stdout, "Allowed") {
		t.Errorf("expected doctor output to contain 'Allowed', full output: %q", stdout)
	}
}

// TestDoctorOutput_IncludesInstanceInfo verifies that the instance name and URL
// appear in the doctor stdout.
func TestDoctorOutput_IncludesInstanceInfo(t *testing.T) {
	srv := newDoctorMockServer(t, "my-test-app")
	defer srv.Close()

	env := setupDoctorTestEnv(t, srv, nil)
	stdout, _ := runDoctorCommand(t, env)

	// The instance name "test-instance" should appear.
	if !strings.Contains(stdout, "test-instance") {
		t.Errorf("doctor stdout does not contain instance name 'test-instance'\n  got: %q", stdout)
	}

	// The instance URL (mock server URL) should appear.
	if !strings.Contains(stdout, srv.URL) {
		t.Errorf("doctor stdout does not contain instance URL %q\n  got: %q", srv.URL, stdout)
	}
}

// TestDoctorOutput_ContainsHeader verifies that the doctor stdout contains the
// expected markdown heading.
func TestDoctorOutput_ContainsHeader(t *testing.T) {
	srv := newDoctorMockServer(t, "my-app")
	defer srv.Close()

	env := setupDoctorTestEnv(t, srv, nil)
	stdout, _ := runDoctorCommand(t, env)

	if !strings.Contains(stdout, "## safe-ify (Coolify Safety Layer)") {
		t.Errorf("doctor stdout missing markdown heading\n  got: %q", stdout)
	}
}

// runDoctorCommandFull is like runDoctorCommand but also returns the cobra error
// so callers can check the exit-code-equivalent behaviour (non-nil error → exit 1).
func runDoctorCommandFull(t *testing.T, args []string) (stdout, stderr string, err error) {
	t.Helper()

	var outBuf, errBuf bytes.Buffer
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	rootCmd.SetArgs(args)
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	defer func() {
		rootCmd.SilenceUsage = false
		rootCmd.SilenceErrors = false
	}()

	err = rootCmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

// TestDoctorExitCode_GlobalConfigMissing verifies that doctor exits with a
// non-nil error (equivalent to exit code 1) when the global config file does
// not exist, and that the stderr output contains [FAIL] diagnostic markers.
func TestDoctorExitCode_GlobalConfigMissing(t *testing.T) {
	dir := t.TempDir()
	nonExistentGlobal := filepath.Join(dir, "does-not-exist.yaml")

	_, stderr, err := runDoctorCommandFull(t, []string{
		"doctor",
		"--config", nonExistentGlobal,
	})

	// The command must signal failure (non-nil error → exit code 1).
	if err == nil {
		t.Errorf("expected doctor to return a non-nil error when global config is missing, got nil")
	}

	// stderr must contain at least one [FAIL] marker.
	if !strings.Contains(stderr, "[FAIL]") {
		t.Errorf("expected stderr to contain [FAIL] diagnostic marker\n  got: %q", stderr)
	}
}

// TestDoctorExitCode_ConnectivityFailure verifies that doctor exits with a
// non-nil error (equivalent to exit code 1) when an instance configured in the
// global config is unreachable, and that stderr contains [FAIL] markers.
func TestDoctorExitCode_ConnectivityFailure(t *testing.T) {
	// Create a server then close it immediately so all connections fail.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	unreachableURL := srv.URL
	srv.Close()

	dir := t.TempDir()

	globalPath := filepath.Join(dir, "config.yaml")
	globalContent := fmt.Sprintf(`instances:
  dead-instance:
    url: %s
    token: test-token-abc
defaults:
  permissions:
    deny: []
`, unreachableURL)
	if err := os.WriteFile(globalPath, []byte(globalContent), 0o600); err != nil {
		t.Fatalf("writing global config: %v", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod temp dir: %v", err)
	}

	// Supply a valid project config so doctor proceeds past project checks.
	projectPath := filepath.Join(dir, ".safe-ify.yaml")
	projectContent := `instance: dead-instance
app_uuid: a1b2c3d4-e5f6-7890-abcd-ef1234567890
permissions:
  deny: []
`
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o644); err != nil {
		t.Fatalf("writing project config: %v", err)
	}

	_, stderr, err := runDoctorCommandFull(t, []string{
		"doctor",
		"--config", globalPath,
		"--project", projectPath,
	})

	// The command must signal failure.
	if err == nil {
		t.Errorf("expected doctor to return a non-nil error when instance is unreachable, got nil")
	}

	// stderr must contain [FAIL] for the connectivity / version checks.
	if !strings.Contains(stderr, "[FAIL]") {
		t.Errorf("expected stderr to contain [FAIL] diagnostic marker\n  got: %q", stderr)
	}
}

// TestDoctorExitCode_OnlyProjectMissing verifies that doctor exits 0 (nil error)
// when the global config is valid but no project config exists in the working
// directory. Missing project config is not a critical failure — doctor should
// skip the project-level checks gracefully.
func TestDoctorExitCode_OnlyProjectMissing(t *testing.T) {
	srv := newDoctorMockServer(t, "my-app")
	defer srv.Close()

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

	// Change to an isolated temp directory so FindProjectConfig cannot discover
	// any .safe-ify.yaml from the real working directory or its parents.
	isolatedDir := t.TempDir()
	t.Chdir(isolatedDir)

	// Do NOT pass --project so doctor auto-discovers (and fails to find) the
	// project config. This exercises the non-critical skip path.
	_, stderr, err := runDoctorCommandFull(t, []string{
		"doctor",
		"--config", globalPath,
	})

	// Missing project config must NOT cause a non-nil error return.
	if err != nil {
		t.Errorf("expected doctor to succeed (exit 0) when only project config is missing, got error: %v\nstderr: %s", err, stderr)
	}

	// Diagnostics should mention that project checks were skipped, not failed.
	if !strings.Contains(stderr, "SKIP") {
		t.Errorf("expected stderr to mention SKIP for project checks\n  got: %q", stderr)
	}
}

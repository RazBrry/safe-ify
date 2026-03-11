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

// TestDeploy_WritesAuditLogEntry is an integration test that verifies the deploy
// command writes an audit log entry after a successful API call.
//
// Strategy:
//   - Spin up an httptest.Server that handles the /api/v1/deploy endpoint.
//   - Write temporary global and project configs pointing to that server.
//   - Redirect HOME to a temp directory so that audit.DefaultAuditLogPath()
//     resolves to a controlled location (not the real ~/.config/safe-ify/audit.log).
//   - Run the deploy command programmatically via rootCmd.
//   - Read the audit log file and verify that exactly one entry was written,
//     containing the "deploy" command token.
func TestDeploy_WritesAuditLogEntry(t *testing.T) {
	// Set up a mock Coolify server that accepts the deploy POST.
	deployResp := `{"deployments":[{"message":"Deployment request queued.","resource_uuid":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","deployment_uuid":"audit-test-uuid"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/deploy" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, deployResp)
			return
		}
		// Unexpected request — fail the handler but keep the test running.
		http.Error(w, "unexpected request: "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
	}))
	defer srv.Close()

	// Create an isolated temp directory tree.
	dir := t.TempDir()

	// Write global config (must be 0600; parent dir must be 0700).
	globalPath := filepath.Join(dir, "config.yaml")
	globalContent := fmt.Sprintf(`instances:
  audit-test-instance:
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

	// Write project config.
	projectPath := filepath.Join(dir, ".safe-ify.yaml")
	projectContent := `instance: audit-test-instance
app_uuid: a1b2c3d4-e5f6-7890-abcd-ef1234567890
permissions:
  deny: []
`
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o644); err != nil {
		t.Fatalf("writing project config: %v", err)
	}

	// Redirect HOME so that audit.DefaultAuditLogPath() writes to a temp location.
	// This prevents pollution of the developer's real audit log during tests.
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	// Compute where the audit log will land.
	expectedAuditLog := filepath.Join(fakeHome, ".config", "safe-ify", "audit.log")

	// Run the deploy command with --json.
	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"deploy",
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

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deploy command returned error: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Verify the audit log was created.
	data, err := os.ReadFile(expectedAuditLog)
	if err != nil {
		t.Fatalf("audit log not created at %q: %v\nstdout: %s\nstderr: %s",
			expectedAuditLog, err, stdout.String(), stderr.String())
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		t.Fatalf("audit log is empty; expected at least one entry")
	}

	// The entry should contain the command name "deploy".
	if !strings.Contains(content, "deploy") {
		t.Errorf("audit log entry does not contain 'deploy'\n  entry: %q", content)
	}

	// The entry should contain the app UUID from the project config.
	if !strings.Contains(content, "a1b2c3d4-e5f6-7890-abcd-ef1234567890") {
		t.Errorf("audit log entry does not contain app UUID\n  entry: %q", content)
	}

	// The entry should contain the instance name.
	if !strings.Contains(content, "audit-test-instance") {
		t.Errorf("audit log entry does not contain instance name\n  entry: %q", content)
	}

	// The entry should contain a result field ("ok" for success).
	if !strings.Contains(content, "ok") {
		t.Errorf("audit log entry does not contain result 'ok'\n  entry: %q", content)
	}

	// The token must NOT appear in the audit log (security contract).
	if strings.Contains(content, "test-token-abc") {
		t.Errorf("SECURITY: audit log must not contain the token, but it does\n  entry: %q", content)
	}

	// There should be exactly one non-empty line (one deploy invocation).
	lines := strings.Split(content, "\n")
	nonEmpty := 0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty++
		}
	}
	if nonEmpty != 1 {
		t.Errorf("expected exactly 1 audit log entry, got %d\n  content: %q", nonEmpty, content)
	}
}

// TestRedeploy_WritesAuditLogEntry verifies that the redeploy command also
// writes an audit log entry, ensuring the audit middleware covers all agent
// commands and not just deploy.
func TestRedeploy_WritesAuditLogEntry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/restart") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"message":"Restart triggered."}`)
			return
		}
		http.Error(w, "unexpected request: "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
	}))
	defer srv.Close()

	dir := t.TempDir()

	globalPath := filepath.Join(dir, "config.yaml")
	globalContent := fmt.Sprintf(`instances:
  audit-redeploy-instance:
    url: %s
    token: test-token-xyz
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
	projectContent := `instance: audit-redeploy-instance
app_uuid: b2c3d4e5-f6a7-8901-bcde-f12345678901
permissions:
  deny: []
`
	if err := os.WriteFile(projectPath, []byte(projectContent), 0o644); err != nil {
		t.Fatalf("writing project config: %v", err)
	}

	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	expectedAuditLog := filepath.Join(fakeHome, ".config", "safe-ify", "audit.log")

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{
		"redeploy",
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

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("redeploy command returned error: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	data, err := os.ReadFile(expectedAuditLog)
	if err != nil {
		t.Fatalf("audit log not created at %q: %v", expectedAuditLog, err)
	}

	content := strings.TrimSpace(string(data))
	if !strings.Contains(content, "redeploy") {
		t.Errorf("audit log entry does not contain 'redeploy'\n  entry: %q", content)
	}
	if strings.Contains(content, "test-token-xyz") {
		t.Errorf("SECURITY: audit log must not contain the token, but it does\n  entry: %q", content)
	}
}

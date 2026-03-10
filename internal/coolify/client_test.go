package coolify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer creates an httptest.Server with the given handler and returns
// a Client configured to use it. The caller is responsible for closing the server.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	client := NewClient(srv.URL, "test-token-abc")
	return srv, client
}

// TestClient_Healthcheck_Success verifies that a 200 response returns no error.
func TestClient_Healthcheck_Success(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})
	defer srv.Close()

	if err := client.Healthcheck(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// TestClient_Healthcheck_AuthFailure verifies that a 401 response returns an error.
func TestClient_Healthcheck_AuthFailure(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Unauthorized")
	})
	defer srv.Close()

	err := client.Healthcheck(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	coolifyErr, ok := err.(*CoolifyError)
	if !ok {
		t.Fatalf("expected *CoolifyError, got %T: %v", err, err)
	}
	if coolifyErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", coolifyErr.StatusCode)
	}
}

// TestClient_ListApplications_Success verifies that a JSON array is parsed correctly.
func TestClient_ListApplications_Success(t *testing.T) {
	apps := []Application{
		{UUID: "uuid-1", Name: "app-one", Status: "running"},
		{UUID: "uuid-2", Name: "app-two", Status: "stopped"},
	}
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(apps)
	})
	defer srv.Close()

	result, err := client.ListApplications(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 applications, got %d", len(result))
	}
	if result[0].UUID != "uuid-1" || result[0].Name != "app-one" {
		t.Errorf("unexpected first app: %+v", result[0])
	}
	if result[1].UUID != "uuid-2" || result[1].Status != "stopped" {
		t.Errorf("unexpected second app: %+v", result[1])
	}
}

// TestClient_GetApplication_Success verifies that a single application is returned.
func TestClient_GetApplication_Success(t *testing.T) {
	app := Application{
		UUID:        "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		Name:        "my-app",
		Status:      "running",
		FQDN:        "https://app.example.com",
		BuildPack:   "nixpacks",
		GitBranch:   "main",
	}
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(app)
	})
	defer srv.Close()

	result, err := client.GetApplication(context.Background(), "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UUID != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("expected UUID a1b2c3d4-e5f6-7890-abcd-ef1234567890, got %s", result.UUID)
	}
	if result.Name != "my-app" {
		t.Errorf("expected name my-app, got %s", result.Name)
	}
	if result.Status != "running" {
		t.Errorf("expected status running, got %s", result.Status)
	}
	if result.FQDN != "https://app.example.com" {
		t.Errorf("expected FQDN https://app.example.com, got %s", result.FQDN)
	}
}

// TestClient_GetApplication_NotFound verifies that a 404 returns an error mentioning the UUID.
func TestClient_GetApplication_NotFound(t *testing.T) {
	uuid := "00000000-0000-0000-0000-000000000000"
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "not found")
	})
	defer srv.Close()

	_, err := client.GetApplication(context.Background(), uuid)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	coolifyErr, ok := err.(*CoolifyError)
	if !ok {
		t.Fatalf("expected *CoolifyError, got %T: %v", err, err)
	}
	if coolifyErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", coolifyErr.StatusCode)
	}
	// The error message should mention the UUID concept (app UUID check).
	if !strings.Contains(coolifyErr.Message, "UUID") && !strings.Contains(coolifyErr.Message, "uuid") {
		t.Errorf("expected error message to mention UUID, got: %s", coolifyErr.Message)
	}
}

// TestClient_Deploy_Success verifies that the deployment UUID is extracted from the response.
func TestClient_Deploy_Success(t *testing.T) {
	deployResp := DeployResponse{
		Deployments: []DeploymentEntry{
			{
				Message:        "Deployment request queued.",
				ResourceUUID:   "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				DeploymentUUID: "dl8k4s0",
			},
		},
	}
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(deployResp)
	})
	defer srv.Close()

	result, err := client.Deploy(context.Background(), "a1b2c3d4-e5f6-7890-abcd-ef1234567890", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Deployments) == 0 {
		t.Fatal("expected at least one deployment entry")
	}
	if result.Deployments[0].DeploymentUUID != "dl8k4s0" {
		t.Errorf("expected deployment UUID dl8k4s0, got %s", result.Deployments[0].DeploymentUUID)
	}
}

// TestClient_Deploy_RateLimit verifies that a 429 with Retry-After header returns an error
// including the retry-after information.
func TestClient_Deploy_RateLimit(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, "rate limited")
	})
	defer srv.Close()

	_, err := client.Deploy(context.Background(), "a1b2c3d4-e5f6-7890-abcd-ef1234567890", false)
	if err == nil {
		t.Fatal("expected error for 429, got nil")
	}
	coolifyErr, ok := err.(*CoolifyError)
	if !ok {
		t.Fatalf("expected *CoolifyError, got %T: %v", err, err)
	}
	if coolifyErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", coolifyErr.StatusCode)
	}
	// Message should mention retry-after value.
	if !strings.Contains(coolifyErr.Message, "60") {
		t.Errorf("expected retry-after value 60 in message, got: %s", coolifyErr.Message)
	}
	if !strings.Contains(strings.ToLower(coolifyErr.Message), "retry") {
		t.Errorf("expected 'retry' in error message, got: %s", coolifyErr.Message)
	}
}

// TestClient_Restart_Success verifies that a successful restart returns no error.
func TestClient_Restart_Success(t *testing.T) {
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"message":"Restart triggered."}`)
	})
	defer srv.Close()

	if err := client.Restart(context.Background(), "a1b2c3d4-e5f6-7890-abcd-ef1234567890"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestClient_GetLogs_Success verifies that log lines are returned and their count and
// content are correct.
func TestClient_GetLogs_Success(t *testing.T) {
	logLines := []string{
		"2026-03-11T10:00:00Z INFO starting",
		"2026-03-11T10:00:01Z INFO ready",
		"2026-03-11T10:00:02Z INFO request received",
	}
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(logLines)
	})
	defer srv.Close()

	lines, err := client.GetLogs(context.Background(), "a1b2c3d4-e5f6-7890-abcd-ef1234567890", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 log lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "starting") {
		t.Errorf("unexpected first line: %s", lines[0])
	}
	if !strings.Contains(lines[2], "request received") {
		t.Errorf("unexpected third line: %s", lines[2])
	}
}

// TestClient_GetLogs_TailParameter verifies that the tail query parameter is sent to the server.
func TestClient_GetLogs_TailParameter(t *testing.T) {
	var capturedTail string
	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedTail = r.URL.Query().Get("tail")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `["line1"]`)
	})
	defer srv.Close()

	_, err := client.GetLogs(context.Background(), "a1b2c3d4-e5f6-7890-abcd-ef1234567890", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTail != "50" {
		t.Errorf("expected tail=50 in query, got %q", capturedTail)
	}
}

// TestClient_NetworkError verifies that a network error is returned when the server is down.
func TestClient_NetworkError(t *testing.T) {
	// Create and immediately close the server so the port is unavailable.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	client := NewClient(srv.URL, "test-token")
	srv.Close() // Close server before making any requests.

	err := client.Healthcheck(context.Background())
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
	// Should be classified as a NetworkError, not a CoolifyError (HTTP-layer).
	if _, ok := err.(*CoolifyError); ok {
		t.Errorf("expected a transport/network error, got CoolifyError: %v", err)
	}
	if _, ok := err.(*NetworkError); !ok {
		t.Errorf("expected *NetworkError, got %T: %v", err, err)
	}
}

// TestClient_BearerToken verifies that the Authorization header is set on all requests.
func TestClient_BearerToken(t *testing.T) {
	const token = "my-super-secret-token"
	var capturedAuthHeaders []string

	srv, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeaders = append(capturedAuthHeaders, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})
	// Override the client token for this test.
	client.token = token
	defer srv.Close()

	// Test Healthcheck
	_ = client.Healthcheck(context.Background())

	// Test ListApplications
	srv2, client2 := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeaders = append(capturedAuthHeaders, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[]`)
	})
	client2.token = token
	defer srv2.Close()
	_, _ = client2.ListApplications(context.Background())

	for i, h := range capturedAuthHeaders {
		expected := "Bearer " + token
		if h != expected {
			t.Errorf("request %d: expected Authorization %q, got %q", i, expected, h)
		}
	}
}

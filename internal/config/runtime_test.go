package config

import (
	"errors"
	"testing"
)

// makeGlobalWithInstance returns a GlobalConfig with a single named instance.
func makeGlobalWithInstance(name, url, token string) *GlobalConfig {
	return &GlobalConfig{
		Instances: map[string]Instance{
			name: {URL: url, Token: token},
		},
	}
}

// makeProjectSingleApp returns a ProjectConfig with one app in the Apps map.
func makeProjectSingleApp(instanceName, appName, uuid string) *ProjectConfig {
	return &ProjectConfig{
		Instance: instanceName,
		Apps: map[string]AppConfig{
			appName: {UUID: uuid, Permissions: PermissionConfig{Deny: []string{}}},
		},
	}
}

// makeProjectMultiApp returns a ProjectConfig with two apps in the Apps map.
func makeProjectMultiApp(instanceName string) *ProjectConfig {
	return &ProjectConfig{
		Instance: instanceName,
		Apps: map[string]AppConfig{
			"api": {UUID: "api-uuid-001", Permissions: PermissionConfig{Deny: []string{}}},
			"web": {UUID: "web-uuid-002", Permissions: PermissionConfig{Deny: []string{}}},
		},
	}
}

// TestResolveRuntime_SingleApp_NoFlag verifies that a single-app project config
// with empty appName auto-selects the only available app.
func TestResolveRuntime_SingleApp_NoFlag(t *testing.T) {
	t.Parallel()

	global := makeGlobalWithInstance("prod", "https://coolify.example.com", "tok123")
	project := makeProjectSingleApp("prod", "myapp", "myapp-uuid-xyz")

	rt, err := ResolveRuntime(global, project, "")
	if err != nil {
		t.Fatalf("ResolveRuntime returned unexpected error: %v", err)
	}
	if rt.AppName != "myapp" {
		t.Errorf("AppName: got %q, want %q", rt.AppName, "myapp")
	}
	if rt.AppUUID != "myapp-uuid-xyz" {
		t.Errorf("AppUUID: got %q, want %q", rt.AppUUID, "myapp-uuid-xyz")
	}
	if rt.InstanceURL != "https://coolify.example.com" {
		t.Errorf("InstanceURL: got %q, want %q", rt.InstanceURL, "https://coolify.example.com")
	}
	if rt.Token != "tok123" {
		t.Errorf("Token: got %q, want %q", rt.Token, "tok123")
	}
}

// TestResolveRuntime_MultiApp_WithFlag verifies that specifying appName="api"
// on a multi-app project config resolves to the correct UUID.
func TestResolveRuntime_MultiApp_WithFlag(t *testing.T) {
	t.Parallel()

	global := makeGlobalWithInstance("prod", "https://coolify.example.com", "tok123")
	project := makeProjectMultiApp("prod")

	rt, err := ResolveRuntime(global, project, "api")
	if err != nil {
		t.Fatalf("ResolveRuntime returned unexpected error: %v", err)
	}
	if rt.AppName != "api" {
		t.Errorf("AppName: got %q, want %q", rt.AppName, "api")
	}
	if rt.AppUUID != "api-uuid-001" {
		t.Errorf("AppUUID: got %q, want %q", rt.AppUUID, "api-uuid-001")
	}
}

// TestResolveRuntime_MultiApp_NoFlag verifies that an empty appName on a
// multi-app config returns AppAmbiguousError.
func TestResolveRuntime_MultiApp_NoFlag(t *testing.T) {
	t.Parallel()

	global := makeGlobalWithInstance("prod", "https://coolify.example.com", "tok123")
	project := makeProjectMultiApp("prod")

	_, err := ResolveRuntime(global, project, "")
	if err == nil {
		t.Fatal("expected AppAmbiguousError, got nil")
	}

	var ambig *AppAmbiguousError
	if !errors.As(err, &ambig) {
		t.Errorf("expected *AppAmbiguousError, got %T: %v", err, err)
	}
	if len(ambig.AvailableApps) != 2 {
		t.Errorf("AvailableApps: got %d entries, want 2", len(ambig.AvailableApps))
	}
}

// TestResolveRuntime_AppNotFound verifies that specifying an unknown appName
// returns AppNotFoundError.
func TestResolveRuntime_AppNotFound(t *testing.T) {
	t.Parallel()

	global := makeGlobalWithInstance("prod", "https://coolify.example.com", "tok123")
	project := makeProjectMultiApp("prod")

	_, err := ResolveRuntime(global, project, "unknown")
	if err == nil {
		t.Fatal("expected AppNotFoundError, got nil")
	}

	var notFound *AppNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("expected *AppNotFoundError, got %T: %v", err, err)
	}
	if notFound.Name != "unknown" {
		t.Errorf("AppNotFoundError.Name: got %q, want %q", notFound.Name, "unknown")
	}
	if len(notFound.AvailableApps) != 2 {
		t.Errorf("AvailableApps: got %d entries, want 2", len(notFound.AvailableApps))
	}
}

// TestResolveRuntime_ThreeLayerDeny verifies that the three-layer deny merge
// (global + project + app) correctly denies commands from each layer while
// leaving the rest allowed.
func TestResolveRuntime_ThreeLayerDeny(t *testing.T) {
	t.Parallel()

	global := &GlobalConfig{
		Instances: map[string]Instance{
			"prod": {URL: "https://coolify.example.com", Token: "tok123"},
		},
		Defaults: DefaultSettings{
			Permissions: PermissionConfig{Deny: []string{"deploy"}},
		},
	}

	project := &ProjectConfig{
		Instance: "prod",
		Apps: map[string]AppConfig{
			"api": {
				UUID:        "api-uuid-001",
				Permissions: PermissionConfig{Deny: []string{"logs"}},
			},
		},
		Permissions: PermissionConfig{Deny: []string{"redeploy"}},
	}

	rt, err := ResolveRuntime(global, project, "api")
	if err != nil {
		t.Fatalf("ResolveRuntime returned unexpected error: %v", err)
	}

	// Commands that should be denied by each layer.
	for _, denied := range []string{"deploy", "redeploy", "logs"} {
		if rt.AllowedCmds[denied] {
			t.Errorf("expected %q to be denied, but AllowedCmds[%q]=true", denied, denied)
		}
	}

	// Commands that should remain allowed.
	for _, allowed := range []string{"status", "list"} {
		if !rt.AllowedCmds[allowed] {
			t.Errorf("expected %q to be allowed, but AllowedCmds[%q]=false", allowed, allowed)
		}
	}
}

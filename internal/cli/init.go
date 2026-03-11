package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/erwinmaasbach/safe-ify/internal/config"
	"github.com/erwinmaasbach/safe-ify/internal/permissions"
	"github.com/erwinmaasbach/safe-ify/internal/tui"
)

func init() {
	rootCmd.AddCommand(newInitCmd())
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Link the current directory to a Coolify application",
		Long: `Interactively link the current project directory to a Coolify instance and
application. Creates a .safe-ify.yaml file in the current directory.

The generated file contains no secrets and can be committed to version control.`,
		RunE: runInit,
	}
}

// coolifyApp is the minimal shape of an application object returned by
// GET /api/v1/applications.
type coolifyApp struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// fetchApplications calls GET {url}/api/v1/applications with the given bearer
// token and returns the list of applications.
func fetchApplications(instanceURL, token string) ([]coolifyApp, error) {
	endpoint := strings.TrimRight(instanceURL, "/") + "/api/v1/applications"

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot build applications request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "safe-ify/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach Coolify at %s: %w", instanceURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// OK — fall through to parse.
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("authentication failed (401): check your API token")
	case http.StatusForbidden:
		return nil, fmt.Errorf("permission denied (403): token may lack 'read' ability")
	default:
		return nil, fmt.Errorf("unexpected status %d from Coolify API", resp.StatusCode)
	}

	var apps []coolifyApp
	if err := json.Unmarshal(body, &apps); err != nil {
		return nil, fmt.Errorf("cannot parse applications response: %w", err)
	}

	return apps, nil
}

// sanitizeAppName converts a Coolify app name to a valid safe-ify app config key.
// It lowercases the string, replaces spaces with hyphens, and strips any characters
// that are not alphanumeric or hyphens. If the result is empty or starts with a
// hyphen, "app" is used as fallback.
func sanitizeAppName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	// Strip any character that is not alphanumeric or a hyphen.
	var buf strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			buf.WriteRune(r)
		}
	}
	result := strings.TrimLeft(buf.String(), "-")
	if result == "" {
		return "app"
	}
	return result
}

func runInit(cmd *cobra.Command, args []string) error {
	// Step a: load global config.
	cfgPath, err := resolveConfigPath()
	if err != nil {
		return err
	}

	globalCfg, err := config.LoadGlobal(cfgPath)
	if err != nil {
		var notFound *config.ConfigNotFoundError
		if errors.As(err, &notFound) {
			return fmt.Errorf("no instances configured — run `safe-ify auth add` first")
		}
		return err
	}

	if len(globalCfg.Instances) == 0 {
		return fmt.Errorf("no instances configured — run `safe-ify auth add` first")
	}

	// Step b: determine current directory and check for existing project config.
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine current directory: %w", err)
	}
	outputPath := filepath.Join(cwd, ".safe-ify.yaml")

	existingConfig := false
	if _, statErr := os.Stat(outputPath); statErr == nil {
		existingConfig = true
	}

	if existingConfig {
		return runInitAddApp(globalCfg, outputPath)
	}
	return runInitNew(globalCfg, outputPath)
}

// runInitNew handles the flow when no project config exists (Case A).
func runInitNew(globalCfg *config.GlobalConfig, outputPath string) error {
	// Step c: collect instance names.
	instanceNames := make([]string, 0, len(globalCfg.Instances))
	for name := range globalCfg.Instances {
		instanceNames = append(instanceNames, name)
	}

	// Step d: instance picker TUI.
	var selectedInstance string
	if err := tui.InitSelectInstanceForm(instanceNames, &selectedInstance).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("instance picker error: %w", err)
	}

	inst := globalCfg.Instances[selectedInstance]

	// Step e: fetch application list from the selected instance.
	fmt.Println(tui.InfoStyle.Render("Fetching applications from Coolify..."))

	coolifyApps, err := fetchApplications(inst.URL, inst.Token)
	if err != nil {
		return fmt.Errorf("cannot fetch applications: %w", err)
	}

	if len(coolifyApps) == 0 {
		return fmt.Errorf("no applications found on instance %q", selectedInstance)
	}

	// Convert to AppOption slice for the TUI.
	appOptions := make([]tui.AppOption, len(coolifyApps))
	for i, a := range coolifyApps {
		appOptions[i] = tui.AppOption{Name: a.Name, UUID: a.UUID}
	}

	// Step f: application picker TUI.
	var selectedAppUUID string
	if err := tui.InitSelectAppForm(appOptions, &selectedAppUUID).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("application picker error: %w", err)
	}

	// Resolve the Coolify app name for the selected UUID.
	coolifyAppName := selectedAppUUID
	for _, a := range coolifyApps {
		if a.UUID == selectedAppUUID {
			coolifyAppName = a.Name
			break
		}
	}

	// Step g: app name prompt.
	defaultName := sanitizeAppName(coolifyAppName)
	var appNameValue string
	if err := tui.InitAppNameForm(defaultName, nil, &appNameValue).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("app name form error: %w", err)
	}

	// Step h: permission deny-list TUI.
	var denyList []string
	if err := tui.InitPermissionsForm(permissions.AllAgentCommands, &denyList).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("permissions form error: %w", err)
	}

	// Step i: validate deny list.
	if err := permissions.ValidateDenyList(denyList); err != nil {
		return fmt.Errorf("invalid deny list: %w", err)
	}
	if denyList == nil {
		denyList = []string{}
	}

	// Step j: build ProjectConfig in multi-app format.
	projectCfg := &config.ProjectConfig{
		Instance: selectedInstance,
		Apps: map[string]config.AppConfig{
			appNameValue: {
				UUID:        selectedAppUUID,
				Permissions: config.PermissionConfig{Deny: denyList},
			},
		},
		Permissions: config.PermissionConfig{Deny: []string{}},
	}

	// Step k: save to .safe-ify.yaml.
	if err := config.SaveProject(outputPath, projectCfg); err != nil {
		return fmt.Errorf("cannot save project config: %w", err)
	}

	// Step l: print success summary.
	fmt.Println(tui.SuccessStyle.Render("Project initialised successfully."))
	fmt.Printf("  Instance:    %s\n", selectedInstance)
	fmt.Printf("  App name:    %s\n", appNameValue)
	fmt.Printf("  Application: %s (%s)\n", coolifyAppName, selectedAppUUID)
	if len(denyList) == 0 {
		fmt.Printf("  Deny list:   (none — all commands allowed)\n")
	} else {
		fmt.Printf("  Deny list:   %s\n", strings.Join(denyList, ", "))
	}
	fmt.Printf("  Config:      %s\n", outputPath)

	return nil
}

// runInitAddApp handles the flow when a project config already exists (Case B).
func runInitAddApp(globalCfg *config.GlobalConfig, outputPath string) error {
	// Load and auto-normalize existing config (handles legacy format per D2).
	projectCfg, err := config.LoadProject(outputPath)
	if err != nil {
		return fmt.Errorf("cannot load existing project config: %w", err)
	}

	// Prompt: "Add another app?"
	var addAnother bool
	addForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Existing project config found. Add another app to this project?").
				Value(&addAnother),
		),
	)
	if err := addForm.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("confirmation form error: %w", err)
	}
	if !addAnother {
		fmt.Println(tui.InfoStyle.Render("No changes made."))
		return nil
	}

	// Use the existing instance.
	selectedInstance := projectCfg.Instance
	inst, ok := globalCfg.Instances[selectedInstance]
	if !ok {
		return fmt.Errorf("instance %q not found in global config", selectedInstance)
	}

	// Fetch application list.
	fmt.Println(tui.InfoStyle.Render("Fetching applications from Coolify..."))

	coolifyApps, err := fetchApplications(inst.URL, inst.Token)
	if err != nil {
		return fmt.Errorf("cannot fetch applications: %w", err)
	}

	if len(coolifyApps) == 0 {
		return fmt.Errorf("no applications found on instance %q", selectedInstance)
	}

	// Convert to AppOption slice for the TUI.
	appOptions := make([]tui.AppOption, len(coolifyApps))
	for i, a := range coolifyApps {
		appOptions[i] = tui.AppOption{Name: a.Name, UUID: a.UUID}
	}

	// Application picker TUI.
	var selectedAppUUID string
	if err := tui.InitSelectAppForm(appOptions, &selectedAppUUID).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("application picker error: %w", err)
	}

	// Resolve Coolify app name.
	coolifyAppName := selectedAppUUID
	for _, a := range coolifyApps {
		if a.UUID == selectedAppUUID {
			coolifyAppName = a.Name
			break
		}
	}

	// Collect existing app names to prevent duplicates.
	existingNames := make([]string, 0, len(projectCfg.Apps))
	for name := range projectCfg.Apps {
		existingNames = append(existingNames, name)
	}

	// App name prompt.
	defaultName := sanitizeAppName(coolifyAppName)
	var appNameValue string
	if err := tui.InitAppNameForm(defaultName, existingNames, &appNameValue).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("app name form error: %w", err)
	}

	// Permission deny-list TUI.
	var denyList []string
	if err := tui.InitPermissionsForm(permissions.AllAgentCommands, &denyList).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("permissions form error: %w", err)
	}

	// Validate deny list.
	if err := permissions.ValidateDenyList(denyList); err != nil {
		return fmt.Errorf("invalid deny list: %w", err)
	}
	if denyList == nil {
		denyList = []string{}
	}

	// Add new app to the Apps map.
	if projectCfg.Apps == nil {
		projectCfg.Apps = make(map[string]config.AppConfig)
	}
	projectCfg.Apps[appNameValue] = config.AppConfig{
		UUID:        selectedAppUUID,
		Permissions: config.PermissionConfig{Deny: denyList},
	}

	// Save updated config.
	if err := config.SaveProject(outputPath, projectCfg); err != nil {
		return fmt.Errorf("cannot save project config: %w", err)
	}

	// Print success summary.
	fmt.Println(tui.SuccessStyle.Render("App added successfully."))
	fmt.Printf("  Instance:    %s\n", selectedInstance)
	fmt.Printf("  App name:    %s\n", appNameValue)
	fmt.Printf("  Application: %s (%s)\n", coolifyAppName, selectedAppUUID)
	if len(denyList) == 0 {
		fmt.Printf("  Deny list:   (none — all commands allowed)\n")
	} else {
		fmt.Printf("  Deny list:   %s\n", strings.Join(denyList, ", "))
	}
	fmt.Printf("  Config:      %s\n", outputPath)

	return nil
}

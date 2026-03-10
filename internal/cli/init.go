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

	// Step b: collect instance names.
	instanceNames := make([]string, 0, len(globalCfg.Instances))
	for name := range globalCfg.Instances {
		instanceNames = append(instanceNames, name)
	}

	// Step c: instance picker TUI.
	var selectedInstance string
	if err := tui.InitSelectInstanceForm(instanceNames, &selectedInstance).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("instance picker error: %w", err)
	}

	inst := globalCfg.Instances[selectedInstance]

	// Step d: fetch application list from the selected instance.
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

	// Step e: application picker TUI.
	var selectedAppUUID string
	if err := tui.InitSelectAppForm(appOptions, &selectedAppUUID).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("application picker error: %w", err)
	}

	// Step f: permission deny-list TUI.
	var denyList []string
	if err := tui.InitPermissionsForm(permissions.AllAgentCommands, &denyList).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("permissions form error: %w", err)
	}

	// Step g: validate deny list.
	if err := permissions.ValidateDenyList(denyList); err != nil {
		return fmt.Errorf("invalid deny list: %w", err)
	}

	// Step h: build ProjectConfig.
	projectCfg := &config.ProjectConfig{
		Instance: selectedInstance,
		AppUUID:  selectedAppUUID,
		Permissions: config.PermissionConfig{
			Deny: denyList,
		},
	}

	// Ensure Deny is never nil in the YAML output.
	if projectCfg.Permissions.Deny == nil {
		projectCfg.Permissions.Deny = []string{}
	}

	// Step i: save to .safe-ify.yaml in the current directory.
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine current directory: %w", err)
	}

	outputPath := filepath.Join(cwd, ".safe-ify.yaml")
	if err := config.SaveProject(outputPath, projectCfg); err != nil {
		return fmt.Errorf("cannot save project config: %w", err)
	}

	// Step j: print success summary.
	// Resolve the selected app name for display.
	appName := selectedAppUUID
	for _, a := range coolifyApps {
		if a.UUID == selectedAppUUID {
			appName = a.Name
			break
		}
	}

	fmt.Println(tui.SuccessStyle.Render("Project initialised successfully."))
	fmt.Printf("  Instance:    %s\n", selectedInstance)
	fmt.Printf("  Application: %s (%s)\n", appName, selectedAppUUID)
	if len(denyList) == 0 {
		fmt.Printf("  Deny list:   (none — all commands allowed)\n")
	} else {
		fmt.Printf("  Deny list:   %s\n", strings.Join(denyList, ", "))
	}
	fmt.Printf("  Config:      %s\n", outputPath)

	return nil
}

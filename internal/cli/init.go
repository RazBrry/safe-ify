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
	if err := requireTTY(); err != nil {
		return err
	}

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

	// Step b: determine current directory and load existing project config (if any).
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine current directory: %w", err)
	}
	outputPath := filepath.Join(cwd, ".safe-ify.yaml")

	var projectCfg *config.ProjectConfig
	if _, statErr := os.Stat(outputPath); statErr == nil {
		projectCfg, err = config.LoadProject(outputPath)
		if err != nil {
			return fmt.Errorf("cannot load existing project config: %w", err)
		}
	}

	// Step c: select instance.
	var selectedInstance string
	if projectCfg != nil {
		// Reuse existing instance.
		selectedInstance = projectCfg.Instance
		fmt.Printf("  Instance: %s (from existing config)\n", selectedInstance)
	} else {
		instanceNames := make([]string, 0, len(globalCfg.Instances))
		for name := range globalCfg.Instances {
			instanceNames = append(instanceNames, name)
		}
		if err := tui.InitSelectInstanceForm(instanceNames, &selectedInstance).Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Println(tui.InfoStyle.Render("Aborted."))
				return nil
			}
			return fmt.Errorf("instance picker error: %w", err)
		}
	}

	inst, ok := globalCfg.Instances[selectedInstance]
	if !ok {
		return fmt.Errorf("instance %q not found in global config", selectedInstance)
	}

	// Step d: fetch all applications from Coolify.
	fmt.Println(tui.InfoStyle.Render("Fetching applications from Coolify..."))

	coolifyApps, err := fetchApplications(inst.URL, inst.Token)
	if err != nil {
		return fmt.Errorf("cannot fetch applications: %w", err)
	}
	if len(coolifyApps) == 0 {
		return fmt.Errorf("no applications found on instance %q", selectedInstance)
	}

	// Build lookup: UUID → Coolify app name.
	coolifyNameByUUID := make(map[string]string, len(coolifyApps))
	appOptions := make([]tui.AppOption, len(coolifyApps))
	for i, a := range coolifyApps {
		coolifyNameByUUID[a.UUID] = a.Name
		appOptions[i] = tui.AppOption{Name: a.Name, UUID: a.UUID}
	}

	// Determine which UUIDs are already configured (for pre-selection).
	var alreadySelected []string
	existingByUUID := make(map[string]string) // UUID → config key name
	if projectCfg != nil {
		for name, appCfg := range projectCfg.Apps {
			alreadySelected = append(alreadySelected, appCfg.UUID)
			existingByUUID[appCfg.UUID] = name
		}
	}

	// Step e: multi-select applications.
	var selectedUUIDs []string
	if err := tui.InitMultiSelectAppForm(appOptions, alreadySelected, &selectedUUIDs).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("application picker error: %w", err)
	}

	if len(selectedUUIDs) == 0 {
		return fmt.Errorf("no applications selected")
	}

	// Step f: determine which apps are new (need name + permissions prompts).
	newUUIDs := make([]string, 0)
	selectedSet := make(map[string]bool, len(selectedUUIDs))
	for _, uuid := range selectedUUIDs {
		selectedSet[uuid] = true
		if _, exists := existingByUUID[uuid]; !exists {
			newUUIDs = append(newUUIDs, uuid)
		}
	}

	// Build the new Apps map, starting with existing apps that are still selected.
	apps := make(map[string]config.AppConfig)
	usedNames := make(map[string]bool)
	var removedApps []string

	if projectCfg != nil {
		for name, appCfg := range projectCfg.Apps {
			if selectedSet[appCfg.UUID] {
				// Keep existing app (preserve its config name and permissions).
				apps[name] = appCfg
				usedNames[name] = true
			} else {
				removedApps = append(removedApps, name)
			}
		}
	}

	// Step g: for each new app, prompt for config name and permissions.
	existingNamesList := make([]string, 0, len(usedNames))
	for name := range usedNames {
		existingNamesList = append(existingNamesList, name)
	}

	for _, uuid := range newUUIDs {
		coolifyName := coolifyNameByUUID[uuid]
		defaultName := sanitizeAppName(coolifyName)

		// Ensure default name doesn't conflict.
		if usedNames[defaultName] {
			for i := 2; ; i++ {
				candidate := fmt.Sprintf("%s-%d", defaultName, i)
				if !usedNames[candidate] {
					defaultName = candidate
					break
				}
			}
		}

		fmt.Printf("\n  Configuring: %s (%s)\n", coolifyName, uuid)

		var appNameValue string
		if err := tui.InitAppNameForm(defaultName, existingNamesList, &appNameValue).Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Println(tui.InfoStyle.Render("Aborted."))
				return nil
			}
			return fmt.Errorf("app name form error: %w", err)
		}

		var denyList []string
		if err := tui.InitPermissionsForm(permissions.AllAgentCommands, &denyList).Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Println(tui.InfoStyle.Render("Aborted."))
				return nil
			}
			return fmt.Errorf("permissions form error: %w", err)
		}
		if denyList == nil {
			denyList = []string{}
		}

		apps[appNameValue] = config.AppConfig{
			UUID:        uuid,
			Permissions: config.PermissionConfig{Deny: denyList},
		}
		usedNames[appNameValue] = true
		existingNamesList = append(existingNamesList, appNameValue)
	}

	// Step h: build and save config.
	newProjectCfg := &config.ProjectConfig{
		Instance:    selectedInstance,
		Apps:        apps,
		Permissions: config.PermissionConfig{Deny: []string{}},
	}
	if projectCfg != nil {
		newProjectCfg.Permissions = projectCfg.Permissions
	}

	if err := config.SaveProject(outputPath, newProjectCfg); err != nil {
		return fmt.Errorf("cannot save project config: %w", err)
	}

	// Step i: print summary.
	fmt.Println()
	fmt.Println(tui.SuccessStyle.Render("Project configured successfully."))
	fmt.Printf("  Instance: %s\n", selectedInstance)
	fmt.Printf("  Config:   %s\n", outputPath)
	fmt.Println()

	for name, appCfg := range apps {
		coolifyName := coolifyNameByUUID[appCfg.UUID]
		if coolifyName == "" {
			coolifyName = appCfg.UUID
		}
		marker := ""
		if _, wasExisting := existingByUUID[appCfg.UUID]; !wasExisting {
			marker = " (new)"
		}
		fmt.Printf("  [%s]%s %s (%s)\n", name, marker, coolifyName, appCfg.UUID)
		if len(appCfg.Permissions.Deny) > 0 {
			fmt.Printf("    deny: %s\n", strings.Join(appCfg.Permissions.Deny, ", "))
		}
	}

	if len(removedApps) > 0 {
		fmt.Println()
		for _, name := range removedApps {
			fmt.Printf("  [%s] removed\n", name)
		}
	}

	return nil
}

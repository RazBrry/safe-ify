package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/erwinmaasbach/safe-ify/internal/config"
	"github.com/erwinmaasbach/safe-ify/internal/coolify"
	"github.com/erwinmaasbach/safe-ify/internal/permissions"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate setup and output CLAUDE.md snippet",
	Long: `Run diagnostic checks on your safe-ify configuration and output a
markdown snippet suitable for appending to CLAUDE.md.

Diagnostics are printed to stderr; the markdown snippet is written to stdout.
Example: safe-ify doctor >> CLAUDE.md`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// commandDisplay maps each agent command name to its display form for the markdown table.
var commandDisplay = []struct {
	name    string
	display string
}{
	{"deploy", "`safe-ify deploy --json`"},
	{"redeploy", "`safe-ify redeploy --json`"},
	{"logs", "`safe-ify logs --json --tail N`"},
	{"status", "`safe-ify status --json`"},
	{"list", "`safe-ify list --json`"},
}

func runDoctor(cmd *cobra.Command, args []string) error {
	stderr := cmd.ErrOrStderr()
	stdout := cmd.OutOrStdout()
	anyFail := false

	fmt.Fprintln(stderr, "=== safe-ify doctor ===")
	fmt.Fprintln(stderr, "")

	// --- Resolve global config path ---
	globalOverride, _ := cmd.Root().PersistentFlags().GetString("config")
	var (
		globalPath string
		err        error
	)
	if globalOverride != "" {
		globalPath = globalOverride
	} else {
		globalPath, err = config.DefaultGlobalConfigPath()
		if err != nil {
			fmt.Fprintf(stderr, "[FAIL] Global config path: %s\n", err)
			return errExitCode1
		}
	}

	// Check (a): global config file exists and has correct permissions.
	fmt.Fprintln(stderr, "[1/8] Global config check")
	permErr := config.CheckPermissions(globalPath)
	if permErr != nil {
		fmt.Fprintf(stderr, "  [FAIL] %s\n", permErr)
		anyFail = true
	} else {
		fmt.Fprintf(stderr, "  [PASS] %s — permissions OK (0600)\n", globalPath)
	}

	// Load global config for subsequent checks (proceed even if perm check failed for better diagnostics).
	globalCfg, loadErr := config.LoadGlobal(globalPath)
	if loadErr != nil {
		fmt.Fprintf(stderr, "  [FAIL] Cannot load global config: %s\n", loadErr)
		// Cannot continue without global config.
		return errExitCode1
	}

	// Check (b): at least one instance configured.
	fmt.Fprintln(stderr, "[2/8] Instances check")
	if len(globalCfg.Instances) == 0 {
		fmt.Fprintln(stderr, "  [FAIL] No instances configured — run `safe-ify auth add`")
		anyFail = true
	} else {
		fmt.Fprintf(stderr, "  [PASS] %d instance(s) configured\n", len(globalCfg.Instances))
	}

	// Check (c): connectivity — healthcheck each instance.
	fmt.Fprintln(stderr, "[3/8] Connectivity check")
	if len(globalCfg.Instances) == 0 {
		fmt.Fprintln(stderr, "  [SKIP] No instances to check")
	} else {
		for name, inst := range globalCfg.Instances {
			client := coolify.NewClient(inst.URL, inst.Token)
			if hErr := client.Healthcheck(context.Background()); hErr != nil {
				fmt.Fprintf(stderr, "  [FAIL] %s (%s): %s\n", name, inst.URL, hErr)
				anyFail = true
			} else {
				fmt.Fprintf(stderr, "  [PASS] %s (%s): reachable\n", name, inst.URL)
			}
		}
	}

	// Check (d): version — call Version() for each instance.
	fmt.Fprintln(stderr, "[4/8] Version check")
	if len(globalCfg.Instances) == 0 {
		fmt.Fprintln(stderr, "  [SKIP] No instances to check")
	} else {
		for name, inst := range globalCfg.Instances {
			client := coolify.NewClient(inst.URL, inst.Token)
			ver, vErr := client.Version(context.Background())
			if vErr != nil {
				fmt.Fprintf(stderr, "  [FAIL] %s: cannot get version: %s\n", name, vErr)
				anyFail = true
			} else {
				fmt.Fprintf(stderr, "  [PASS] %s: Coolify version %s\n", name, ver)
			}
		}
	}

	// --- Resolve project config path ---
	projectOverride, _ := cmd.Root().PersistentFlags().GetString("project")
	var projectPath string
	var projectFound bool

	// Check (e): project config found.
	fmt.Fprintln(stderr, "[5/8] Project config check")
	if projectOverride != "" {
		projectPath = projectOverride
		if _, statErr := os.Stat(projectPath); statErr == nil {
			projectFound = true
			fmt.Fprintf(stderr, "  [PASS] %s found\n", projectPath)
		} else {
			fmt.Fprintf(stderr, "  [FAIL] %s not found\n", projectPath)
			anyFail = true
		}
	} else {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			fmt.Fprintf(stderr, "  [FAIL] Cannot determine working directory: %s\n", cwdErr)
			anyFail = true
		} else {
			found, findErr := config.FindProjectConfig(cwd)
			if findErr != nil {
				fmt.Fprintf(stderr, "  [FAIL] .safe-ify.yaml not found (searched from %s)\n", cwd)
				// Not critical — skip remaining project checks.
			} else {
				projectPath = found
				projectFound = true
				fmt.Fprintf(stderr, "  [PASS] %s found\n", projectPath)
			}
		}
	}

	// The remaining checks require a project config. If not found, skip but don't fail entirely.
	if !projectFound {
		fmt.Fprintln(stderr, "[6/8] Instance reference check — SKIP (no project config)")
		fmt.Fprintln(stderr, "[7/8] App UUID check — SKIP (no project config)")
		fmt.Fprintln(stderr, "[8/8] Permissions check — SKIP (no project config)")
		fmt.Fprintln(stderr, "")

		if anyFail {
			return errExitCode1
		}
		return nil
	}

	projectCfg, projLoadErr := config.LoadProject(projectPath)
	if projLoadErr != nil {
		fmt.Fprintf(stderr, "  [FAIL] Cannot load project config: %s\n", projLoadErr)
		anyFail = true
		fmt.Fprintln(stderr, "[6/8] Instance reference check — SKIP (project config invalid)")
		fmt.Fprintln(stderr, "[7/8] App UUID check — SKIP (project config invalid)")
		fmt.Fprintln(stderr, "[8/8] Permissions check — SKIP (project config invalid)")
		fmt.Fprintln(stderr, "")
		if anyFail {
			return errExitCode1
		}
		return nil
	}

	// Check (f): referenced instance exists in global config.
	fmt.Fprintln(stderr, "[6/8] Instance reference check")
	inst, instOK := globalCfg.Instances[projectCfg.Instance]
	if !instOK {
		fmt.Fprintf(stderr, "  [FAIL] Instance %q not found in global config\n", projectCfg.Instance)
		anyFail = true
	} else {
		fmt.Fprintf(stderr, "  [PASS] Instance %q found (%s)\n", projectCfg.Instance, inst.URL)
	}

	// Check (g): app UUID valid — call GetApplication.
	fmt.Fprintln(stderr, "[7/8] App UUID check")
	var appName string
	if !instOK {
		fmt.Fprintln(stderr, "  [SKIP] No valid instance to use for API call")
	} else {
		client := coolify.NewClient(inst.URL, inst.Token)
		app, appErr := client.GetApplication(context.Background(), projectCfg.AppUUID)
		if appErr != nil {
			fmt.Fprintf(stderr, "  [FAIL] App UUID %q: %s\n", projectCfg.AppUUID, appErr)
			anyFail = true
		} else {
			appName = app.Name
			fmt.Fprintf(stderr, "  [PASS] App UUID %q — name: %q\n", projectCfg.AppUUID, app.Name)
		}
	}

	// Check (h): resolve permissions, list allowed and denied.
	fmt.Fprintln(stderr, "[8/8] Permissions check")
	enforcer := permissions.NewEnforcer(*globalCfg, *projectCfg)
	allowedCmds := enforcer.AllowedCommands()
	deniedCmds := enforcer.DeniedCommands()
	fmt.Fprintf(stderr, "  Allowed: %v\n", allowedCmds)
	fmt.Fprintf(stderr, "  Denied:  %v\n", deniedCmds)
	fmt.Fprintln(stderr, "  [PASS]")

	fmt.Fprintln(stderr, "")

	// --- Output CLAUDE.md markdown snippet to stdout ---
	var instanceURL string
	if instOK {
		instanceURL = inst.URL
	}

	fmt.Fprintln(stdout, "## safe-ify (Coolify Safety Layer)")
	fmt.Fprintln(stdout, "")
	fmt.Fprintf(stdout, "Instance: %s (%s)\n", projectCfg.Instance, instanceURL)
	fmt.Fprintf(stdout, "Application: %s (%s)\n", appName, projectCfg.AppUUID)
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "### Available commands")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "| Command | Status |")
	fmt.Fprintln(stdout, "|---------|--------|")

	for _, cd := range commandDisplay {
		status := "Allowed"
		if enforcer.Check(cd.name) != nil {
			status = "Denied"
		}
		fmt.Fprintf(stdout, "| %s | %s |\n", cd.display, status)
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "### Usage")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "All commands support `--json` for structured output.")

	if anyFail {
		return errExitCode1
	}
	return nil
}

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
			fmt.Fprintf(stderr, "  [SKIP] %s not found — skipping project checks\n", projectPath)
			// Missing project config is not a critical failure; skip remaining project checks.
		}
	} else {
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			fmt.Fprintf(stderr, "  [FAIL] Cannot determine working directory: %s\n", cwdErr)
			anyFail = true
		} else {
			found, findErr := config.FindProjectConfig(cwd)
			if findErr != nil {
				fmt.Fprintf(stderr, "  [SKIP] .safe-ify.yaml not found (searched from %s) — skipping project checks\n", cwd)
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

		// Output a partial CLAUDE.md snippet (instance info only, no app/permission sections).
		fmt.Fprintln(stdout, "## safe-ify (Coolify Safety Layer)")
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "<!-- No project config found. Run `safe-ify init` to link this project. -->")
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "### Usage")
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "All commands support `--json` for structured output.")

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

	// Check (g): app UUID valid — call GetApplication for each app in the Apps map.
	fmt.Fprintln(stderr, "[7/8] App UUID check")
	// appNames maps app key -> resolved name (for CLAUDE.md snippet).
	appNames := make(map[string]string, len(projectCfg.Apps))
	if !instOK {
		fmt.Fprintln(stderr, "  [SKIP] No valid instance to use for API call")
	} else {
		client := coolify.NewClient(inst.URL, inst.Token)
		for appKey, appCfg := range projectCfg.Apps {
			app, appErr := client.GetApplication(context.Background(), appCfg.UUID)
			if appErr != nil {
				fmt.Fprintf(stderr, "  [FAIL] App %q (UUID %q): %s\n", appKey, appCfg.UUID, appErr)
				anyFail = true
			} else {
				appNames[appKey] = app.Name
				fmt.Fprintf(stderr, "  [PASS] App %q (UUID %q) — name: %q\n", appKey, appCfg.UUID, app.Name)
			}
		}
	}

	// Check (h): resolve permissions per app, list allowed and denied.
	fmt.Fprintln(stderr, "[8/8] Permissions check")
	for appKey, appCfg := range projectCfg.Apps {
		enforcer := permissions.NewEnforcer(*globalCfg, *projectCfg, appCfg.Permissions.Deny)
		allowedCmds := enforcer.AllowedCommands()
		deniedCmds := enforcer.DeniedCommands()
		fmt.Fprintf(stderr, "  App %q — Allowed: %v\n", appKey, allowedCmds)
		fmt.Fprintf(stderr, "  App %q — Denied:  %v\n", appKey, deniedCmds)
	}
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
	fmt.Fprintln(stdout, "")

	multiApp := len(projectCfg.Apps) > 1

	for appKey, appCfg := range projectCfg.Apps {
		resolvedName := appNames[appKey]
		fmt.Fprintf(stdout, "Application: %s — %s (%s)\n", appKey, resolvedName, appCfg.UUID)

		enforcer := permissions.NewEnforcer(*globalCfg, *projectCfg, appCfg.Permissions.Deny)
		fmt.Fprintln(stdout, "")
		fmt.Fprintf(stdout, "### Available commands (%s)\n", appKey)
		fmt.Fprintln(stdout, "")
		fmt.Fprintln(stdout, "| Command | Status |")
		fmt.Fprintln(stdout, "|---------|--------|")

		for _, cd := range commandDisplay {
			status := "Allowed"
			if enforcer.Check(cd.name) != nil {
				status = "Denied"
			}
			display := cd.display
			if multiApp {
				// Insert --app <name> flag into the command display string.
				// Replace e.g. "`safe-ify deploy --json`" with "`safe-ify deploy --app <name> --json`"
				// by inserting before " --json" or at end before the closing backtick.
				appFlag := " --app " + appKey
				if idx := len(display) - 1; display[idx] == '`' {
					display = display[:idx] + appFlag + "`"
				}
			}
			fmt.Fprintf(stdout, "| %s | %s |\n", display, status)
		}
		fmt.Fprintln(stdout, "")
	}

	fmt.Fprintln(stdout, "### Usage")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "All commands support `--json` for structured output.")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "**Note:** App UUIDs are Coolify-assigned identifiers. They may not follow RFC 4122 format — this is normal. Do not attempt to change or reformat them.")

	if anyFail {
		return errExitCode1
	}
	return nil
}

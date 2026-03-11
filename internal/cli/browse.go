package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/RazBrry/safe-ify/internal/config"
	"github.com/RazBrry/safe-ify/internal/coolify"
	"github.com/RazBrry/safe-ify/internal/tui"
)

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse all applications on the Coolify instance",
	Long: `Browse all applications available on the connected Coolify instance.
This is an interactive command that requires a terminal.

To see only the apps configured for this project, use: safe-ify list`,
	RunE: runBrowse,
}

func init() {
	rootCmd.AddCommand(browseCmd)
}

func runBrowse(cmd *cobra.Command, args []string) error {
	if err := requireTTY(); err != nil {
		return err
	}

	// Load global config.
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

	// Try to use instance from project config, otherwise prompt.
	var selectedInstance string

	cwd, _ := os.Getwd()
	if cwd != "" {
		if found, findErr := config.FindProjectConfig(cwd); findErr == nil {
			if projectCfg, loadErr := config.LoadProject(found); loadErr == nil {
				selectedInstance = projectCfg.Instance
			}
		}
	}

	if selectedInstance == "" {
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

	fmt.Printf("Fetching applications from %s (%s)...\n\n", selectedInstance, inst.URL)

	client := coolify.NewClient(inst.URL, inst.Token)
	apps, err := client.ListApplications(context.Background())
	if err != nil {
		return fmt.Errorf("cannot fetch applications: %w", err)
	}

	if len(apps) == 0 {
		fmt.Println("No applications found on this instance.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "All applications on %s:\n\n", selectedInstance)
	fmt.Fprintf(cmd.OutOrStdout(), "  %-40s %-30s %s\n", "UUID", "Name", "Status")
	fmt.Fprintf(cmd.OutOrStdout(), "  %-40s %-30s %s\n", "----", "----", "------")
	for _, app := range apps {
		fmt.Fprintf(cmd.OutOrStdout(), "  %-40s %-30s %s\n", app.UUID, app.Name, app.Status)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n  Total: %d applications\n", len(apps))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "To add apps to this project, run: safe-ify init")

	return nil
}

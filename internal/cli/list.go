package cli

import (
	"fmt"
	"os"

	"github.com/erwinmaasbach/safe-ify/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured applications for this project",
	Long:  "List the applications configured in .safe-ify.yaml for this project.",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")

	// Load project config directly — list doesn't need a specific app resolved.
	projectOverride, _ := cmd.Root().PersistentFlags().GetString("project")
	var projectPath string

	if projectOverride != "" {
		projectPath = projectOverride
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			msg := fmt.Sprintf("cannot determine working directory: %s", err)
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodeAPIError, msg)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", msg)
			}
			return errExitCode1
		}
		found, err := config.FindProjectConfig(cwd)
		if err != nil {
			msg := "No project config found. Run `safe-ify init` first."
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodeConfigNotFound, msg)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", msg)
			}
			return errExitCode1
		}
		projectPath = found
	}

	projectCfg, err := config.LoadProject(projectPath)
	if err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), mapConfigError(err), err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		}
		return errExitCode1
	}

	// Build the app list from the project config.
	type appEntry struct {
		Name     string   `json:"name"`
		UUID     string   `json:"uuid"`
		Instance string   `json:"instance"`
		Deny     []string `json:"deny,omitempty"`
	}
	type listData struct {
		Applications []appEntry `json:"applications"`
		Count        int        `json:"count"`
	}

	entries := make([]appEntry, 0, len(projectCfg.Apps))
	for name, appCfg := range projectCfg.Apps {
		entries = append(entries, appEntry{
			Name:     name,
			UUID:     appCfg.UUID,
			Instance: projectCfg.Instance,
			Deny:     appCfg.Permissions.Deny,
		})
	}

	data := listData{
		Applications: entries,
		Count:        len(entries),
	}

	if useJSON {
		OutputJSON(cmd.OutOrStdout(), Response{
			OK:   true,
			Data: data,
		})
	} else {
		if len(entries) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No applications configured. Run `safe-ify init` to add apps.")
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Instance: %s\n\n", projectCfg.Instance)
		fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-40s %s\n", "Name", "UUID", "Deny")
		fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-40s %s\n", "----", "----", "----")
		for _, e := range entries {
			deny := "(none)"
			if len(e.Deny) > 0 {
				deny = fmt.Sprintf("%v", e.Deny)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%-20s %-40s %s\n", e.Name, e.UUID, deny)
		}
	}

	return nil
}

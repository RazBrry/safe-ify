package cli

import (
	"context"
	"fmt"

	"github.com/erwinmaasbach/safe-ify/internal/config"
	"github.com/erwinmaasbach/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available applications",
	Long:  "List all applications available on the Coolify instance configured in .safe-ify.yaml.",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")

	err := runAgentCommand(cmd, "list", func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["list"] {
			err := fmt.Errorf("command %q is not permitted for this project", "list")
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
			}
			return nil, errExitCode1
		}

		apps, err := client.ListApplications(context.Background())
		if err != nil {
			if useJSON {
				OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
			}
			return nil, errExitCode1
		}

		type appSummary struct {
			UUID   string `json:"uuid"`
			Name   string `json:"name"`
			Status string `json:"status"`
			FQDN   string `json:"fqdn,omitempty"`
		}
		type listData struct {
			Applications []appSummary `json:"applications"`
			Count        int          `json:"count"`
		}

		summaries := make([]appSummary, len(apps))
		for i, app := range apps {
			summaries[i] = appSummary{
				UUID:   app.UUID,
				Name:   app.Name,
				Status: app.Status,
				FQDN:   app.FQDN,
			}
		}
		data := listData{
			Applications: summaries,
			Count:        len(summaries),
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{
				OK:   true,
				Data: data,
			})
		} else {
			if len(apps) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No applications found.")
				return data, nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%-40s %-30s %s\n", "UUID", "Name", "Status")
			fmt.Fprintf(cmd.OutOrStdout(), "%-40s %-30s %s\n", "----", "----", "------")
			for _, app := range apps {
				fmt.Fprintf(cmd.OutOrStdout(), "%-40s %-30s %s\n", app.UUID, app.Name, app.Status)
			}
		}
		return data, nil
	})

	if err == errExitCode1 {
		return errExitCode1
	}
	if err != nil {
		// Provide a specific user-friendly message for missing project config.
		if _, ok := err.(*config.ProjectConfigNotFoundError); ok {
			msg := "No project config found. Run `safe-ify init` first."
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodeConfigNotFound, msg)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", msg)
			}
		} else {
			if useJSON {
				OutputError(cmd.OutOrStdout(), mapConfigError(err), err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
			}
		}
		return errExitCode1
	}
	return nil
}

package cli

import (
	"context"
	"fmt"

	"github.com/RazBrry/safe-ify/internal/config"
	"github.com/RazBrry/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check deployment status",
	Long:  "Retrieve the current status of the application configured in .safe-ify.yaml.",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")

	err := runAgentCommand(cmd, "status", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["status"] {
			err := fmt.Errorf("command %q is not permitted for this project", "status")
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
			}
			return nil, errExitCode1
		}

		app, err := client.GetApplication(context.Background(), cfg.AppUUID)
		if err != nil {
			if useJSON {
				OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
			}
			return nil, errExitCode1
		}

		type statusData struct {
			UUID           string `json:"uuid"`
			Name           string `json:"name"`
			Status         string `json:"status"`
			FQDN           string `json:"fqdn,omitempty"`
			LastDeployment string `json:"last_deployment,omitempty"`
		}
		data := statusData{
			UUID:           app.UUID,
			Name:           app.Name,
			Status:         app.Status,
			FQDN:           app.FQDN,
			LastDeployment: app.UpdatedAt,
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{
				OK:   true,
				Data: data,
			})
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Application: %s (%s)\n", app.Name, app.UUID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status:      %s\n", app.Status)
			if app.FQDN != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "FQDN:        %s\n", app.FQDN)
			}
		}
		return data, nil
	})

	if err == errExitCode1 {
		return errExitCode1
	}
	if err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), mapConfigError(err), err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		}
		return errExitCode1
	}
	return nil
}

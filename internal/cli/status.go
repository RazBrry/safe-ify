package cli

import (
	"context"
	"fmt"

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

	runtime, client, err := resolveAgentConfig(cmd)
	if err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), mapConfigError(err), err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		}
		return errExitCode1
	}

	// Check permission before making any API call.
	if err := checkPermission(runtime, "status"); err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
		}
		return errExitCode1
	}

	app, err := client.GetApplication(context.Background(), runtime.AppUUID)
	if err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), ErrCodeAPIError, err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		}
		return errExitCode1
	}

	if useJSON {
		type statusData struct {
			UUID           string `json:"uuid"`
			Name           string `json:"name"`
			Status         string `json:"status"`
			FQDN           string `json:"fqdn,omitempty"`
			LastDeployment string `json:"last_deployment,omitempty"`
		}
		OutputJSON(cmd.OutOrStdout(), Response{
			OK: true,
			Data: statusData{
				UUID:           app.UUID,
				Name:           app.Name,
				Status:         app.Status,
				FQDN:           app.FQDN,
				LastDeployment: app.UpdatedAt,
			},
		})
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Application: %s (%s)\n", app.Name, app.UUID)
		fmt.Fprintf(cmd.OutOrStdout(), "Status:      %s\n", app.Status)
		if app.FQDN != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "FQDN:        %s\n", app.FQDN)
		}
	}

	return nil
}

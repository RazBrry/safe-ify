package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var redeployCmd = &cobra.Command{
	Use:   "redeploy",
	Short: "Redeploy the current version",
	Long:  "Restart/redeploy the currently deployed version of the application.",
	RunE:  runRedeploy,
}

func init() {
	rootCmd.AddCommand(redeployCmd)
}

func runRedeploy(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")

	runtime, client, enforcer, err := resolveAgentConfig(cmd)
	if err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), mapConfigError(err), err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		}
		return errExitCode1
	}

	// Check permission before making any API call.
	if err := enforcer.Check("redeploy"); err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
		}
		return errExitCode1
	}

	if err := client.Restart(context.Background(), runtime.AppUUID); err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		}
		return errExitCode1
	}

	if useJSON {
		type redeployData struct {
			Message string `json:"message"`
		}
		OutputJSON(cmd.OutOrStdout(), Response{
			OK:   true,
			Data: redeployData{Message: "Restart triggered."},
		})
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Restart triggered.")
	}

	return nil
}

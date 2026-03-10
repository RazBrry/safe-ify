package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Trigger a deployment",
	Long:  "Trigger a new deployment for the application configured in .safe-ify.yaml.",
	RunE:  runDeploy,
}

func init() {
	deployCmd.Flags().Bool("force", false, "Force deployment even if no changes detected")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
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
	if err := enforcer.Check("deploy"); err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
		}
		return errExitCode1
	}

	force, _ := cmd.Flags().GetBool("force")
	resp, err := client.Deploy(context.Background(), runtime.AppUUID, force)
	if err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		}
		return errExitCode1
	}

	if useJSON {
		type deployData struct {
			Message        string `json:"message"`
			DeploymentUUID string `json:"deployment_uuid,omitempty"`
		}
		var deploymentUUID string
		if len(resp.Deployments) > 0 {
			deploymentUUID = resp.Deployments[0].DeploymentUUID
		}
		OutputJSON(cmd.OutOrStdout(), Response{
			OK: true,
			Data: deployData{
				Message:        "Deployment queued.",
				DeploymentUUID: deploymentUUID,
			},
		})
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Deployment queued.")
		if len(resp.Deployments) > 0 && resp.Deployments[0].DeploymentUUID != "" {
			fmt.Fprintf(cmd.OutOrStdout(), " (deployment UUID: %s)", resp.Deployments[0].DeploymentUUID)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	return nil
}

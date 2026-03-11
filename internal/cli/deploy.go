package cli

import (
	"context"
	"fmt"

	"github.com/erwinmaasbach/safe-ify/internal/config"
	"github.com/erwinmaasbach/safe-ify/internal/coolify"
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
	force, _ := cmd.Flags().GetBool("force")

	err := runAgentCommand(cmd, "deploy", func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["deploy"] {
			err := fmt.Errorf("command %q is not permitted for this project", "deploy")
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
			}
			return nil, errExitCode1
		}

		resp, err := client.Deploy(context.Background(), cfg.AppUUID, force)
		if err != nil {
			if useJSON {
				OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
			}
			return nil, errExitCode1
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
			data := deployData{
				Message:        "Deployment queued.",
				DeploymentUUID: deploymentUUID,
			}
			OutputJSON(cmd.OutOrStdout(), Response{
				OK:   true,
				Data: data,
			})
			return data, nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Deployment queued.")
		if len(resp.Deployments) > 0 && resp.Deployments[0].DeploymentUUID != "" {
			fmt.Fprintf(cmd.OutOrStdout(), " (deployment UUID: %s)", resp.Deployments[0].DeploymentUUID)
		}
		fmt.Fprintln(cmd.OutOrStdout())
		return resp, nil
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

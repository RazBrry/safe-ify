package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/RazBrry/safe-ify/internal/config"
	"github.com/RazBrry/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Trigger a deployment",
	Long: `Trigger a new deployment for the application configured in .safe-ify.yaml.

Use --wait to poll for completion (checks deployment status every --poll-interval
seconds, times out after --timeout seconds). Tracks the specific deployment, not
the app's general status.`,
	RunE: runDeploy,
}

func init() {
	deployCmd.Flags().Bool("force", false, "Force deployment even if no changes detected")
	deployCmd.Flags().Bool("wait", false, "Wait for deployment to complete (poll status)")
	deployCmd.Flags().Int("timeout", 300, "Max seconds to wait for deployment (with --wait)")
	deployCmd.Flags().Int("poll-interval", 15, "Seconds between status polls (with --wait)")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")
	force, _ := cmd.Flags().GetBool("force")
	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetInt("timeout")
	pollInterval, _ := cmd.Flags().GetInt("poll-interval")

	err := runAgentCommand(cmd, "deploy", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
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

		var deploymentUUID string
		if len(resp.Deployments) > 0 {
			deploymentUUID = resp.Deployments[0].DeploymentUUID
		}

		if !wait {
			type deployData struct {
				Message        string `json:"message"`
				DeploymentUUID string `json:"deployment_uuid,omitempty"`
			}
			data := deployData{
				Message:        "Deployment queued.",
				DeploymentUUID: deploymentUUID,
			}
			if useJSON {
				OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Deployment queued.")
				if deploymentUUID != "" {
					fmt.Fprintf(cmd.OutOrStdout(), " (deployment UUID: %s)", deploymentUUID)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return data, nil
		}

		// --wait: poll the specific deployment until it completes or times out.
		if deploymentUUID == "" {
			msg := "cannot use --wait: no deployment UUID returned by Coolify"
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodeAPIError, msg)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", msg)
			}
			return nil, errExitCode1
		}

		if !useJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "Deployment queued (%s). Waiting for completion (timeout: %ds)...\n", deploymentUUID, timeout)
		}

		finalStatus, err := pollDeployment(cmd, client, deploymentUUID, useJSON, timeout, pollInterval)
		if err != nil {
			return nil, err
		}

		type deployWaitData struct {
			Message        string `json:"message"`
			DeploymentUUID string `json:"deployment_uuid"`
			Status         string `json:"status"`
		}
		data := deployWaitData{
			DeploymentUUID: deploymentUUID,
			Status:         finalStatus,
		}

		if isDeploymentFinished(finalStatus) {
			data.Message = "Deployment completed successfully."
		} else {
			data.Message = fmt.Sprintf("Deployment finished with status: %s", finalStatus)
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Deployment complete — status: %s\n", finalStatus)
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

// pollDeployment polls GET /api/v1/deployments/{uuid} for the specific deployment
// until its status is no longer in-progress, or the timeout is reached.
func pollDeployment(cmd *cobra.Command, client *coolify.Client, deploymentUUID string, useJSON bool, timeoutSec, intervalSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	interval := time.Duration(intervalSec) * time.Second

	for {
		time.Sleep(interval)

		d, err := client.GetDeployment(context.Background(), deploymentUUID)
		if err != nil {
			if useJSON {
				OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error polling deployment: %s\n", err)
			}
			return "", errExitCode1
		}

		if !useJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "  deployment status: %s\n", d.Status)
		}

		if !isDeploymentInProgress(d.Status) {
			return d.Status, nil
		}

		if time.Now().After(deadline) {
			msg := fmt.Sprintf("timed out after %ds — deployment %s still %s", timeoutSec, deploymentUUID, d.Status)
			if useJSON {
				OutputError(cmd.OutOrStdout(), "DEPLOY_TIMEOUT", msg)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", msg)
			}
			return d.Status, errExitCode1
		}
	}
}

// isDeploymentInProgress returns true if the deployment status indicates it's still running.
// Coolify deployment statuses: queued, in_progress, finished, failed, cancelled.
func isDeploymentInProgress(status string) bool {
	switch status {
	case "queued", "in_progress", "deploying", "building":
		return true
	}
	return false
}

// isDeploymentFinished returns true if the deployment completed successfully.
func isDeploymentFinished(status string) bool {
	return status == "finished"
}

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

Use --wait to poll for completion (checks status every --poll-interval seconds,
times out after --timeout seconds).`,
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
			// No wait — return immediately.
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

		// --wait: poll status until deployment completes or timeout.
		if !useJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "Deployment queued")
			if deploymentUUID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), " (%s)", deploymentUUID)
			}
			fmt.Fprintf(cmd.OutOrStdout(), ". Waiting for completion (timeout: %ds)...\n", timeout)
		}

		finalStatus, err := pollDeployment(cmd, client, cfg.AppUUID, useJSON, timeout, pollInterval)
		if err != nil {
			return nil, err
		}

		type deployWaitData struct {
			Message        string `json:"message"`
			DeploymentUUID string `json:"deployment_uuid,omitempty"`
			Status         string `json:"status"`
		}
		data := deployWaitData{
			DeploymentUUID: deploymentUUID,
			Status:         finalStatus,
		}

		if isHealthyStatus(finalStatus) {
			data.Message = "Deployment completed successfully."
		} else {
			data.Message = fmt.Sprintf("Deployment finished with status: %s", finalStatus)
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Final status: %s\n", finalStatus)
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

// pollDeployment polls GetApplication in two phases:
//  1. Wait for the status to transition to a deploying state (building/deploying/etc).
//     This handles the delay between triggering a deploy and Coolify starting the build.
//  2. Wait for the status to settle (no longer deploying).
//
// Returns the final status or times out.
func pollDeployment(cmd *cobra.Command, client *coolify.Client, uuid string, useJSON bool, timeoutSec, intervalSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	interval := time.Duration(intervalSec) * time.Second
	sawDeploying := false

	for {
		time.Sleep(interval)

		app, err := client.GetApplication(context.Background(), uuid)
		if err != nil {
			if useJSON {
				OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error polling status: %s\n", err)
			}
			return "", errExitCode1
		}

		if !useJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "  status: %s\n", app.Status)
		}

		if isDeployingStatus(app.Status) {
			sawDeploying = true
		}

		// Only return when we've seen it enter deploying and then settle.
		if sawDeploying && !isDeployingStatus(app.Status) {
			return app.Status, nil
		}

		if time.Now().After(deadline) {
			msg := fmt.Sprintf("timed out after %ds — last status: %s", timeoutSec, app.Status)
			if !sawDeploying {
				msg = fmt.Sprintf("timed out after %ds — deployment never started (status stayed %s)", timeoutSec, app.Status)
			}
			if useJSON {
				OutputError(cmd.OutOrStdout(), "DEPLOY_TIMEOUT", msg)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", msg)
			}
			return app.Status, errExitCode1
		}
	}
}

// isDeployingStatus returns true if the status indicates deployment is still in progress.
func isDeployingStatus(status string) bool {
	switch status {
	case "deploying", "building", "restarting", "starting":
		return true
	}
	return false
}

// isHealthyStatus returns true if the status indicates a successful deployment.
func isHealthyStatus(status string) bool {
	switch status {
	case "running", "running:healthy":
		return true
	}
	return false
}

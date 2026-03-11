package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/RazBrry/safe-ify/internal/config"
	"github.com/RazBrry/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

var previewCmd = &cobra.Command{
	Use:   "preview-deploy",
	Short: "Trigger a preview deployment for a branch or PR",
	Long: `Deploy a specific branch or tag as a preview deployment.

This is useful for PR-based workflows where agents deploy branches for review.
Use --wait to poll until the deployment completes.`,
	RunE: runPreviewDeploy,
}

func init() {
	previewCmd.Flags().String("branch", "", "Branch or tag to deploy (required)")
	previewCmd.Flags().Bool("wait", false, "Wait for deployment to complete")
	previewCmd.Flags().Int("timeout", 300, "Max seconds to wait (with --wait)")
	previewCmd.Flags().Int("poll-interval", 15, "Seconds between status polls (with --wait)")
	_ = previewCmd.MarkFlagRequired("branch")
	rootCmd.AddCommand(previewCmd)
}

func runPreviewDeploy(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")
	branch, _ := cmd.Flags().GetString("branch")
	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetInt("timeout")
	pollInterval, _ := cmd.Flags().GetInt("poll-interval")

	err := runAgentCommand(cmd, "preview-deploy", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["preview-deploy"] {
			err := fmt.Errorf("command %q is not permitted for this project", "preview-deploy")
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
			}
			return nil, errExitCode1
		}

		resp, err := client.DeployByTag(context.Background(), cfg.AppUUID, branch)
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
			type previewData struct {
				Message        string `json:"message"`
				Branch         string `json:"branch"`
				DeploymentUUID string `json:"deployment_uuid,omitempty"`
			}
			data := previewData{
				Message:        fmt.Sprintf("Preview deployment of %s queued.", branch),
				Branch:         branch,
				DeploymentUUID: deploymentUUID,
			}
			if useJSON {
				OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Preview deployment of %s queued.", branch)
				if deploymentUUID != "" {
					fmt.Fprintf(cmd.OutOrStdout(), " (deployment UUID: %s)", deploymentUUID)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return data, nil
		}

		// --wait: poll the deployment.
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
			fmt.Fprintf(cmd.OutOrStdout(), "Preview deployment of %s queued (%s). Waiting for completion (timeout: %ds)...\n", branch, deploymentUUID, timeout)
		}

		finalStatus, err := pollPreview(cmd, client, deploymentUUID, useJSON, timeout, pollInterval)
		if err != nil {
			return nil, err
		}

		type previewWaitData struct {
			Message        string `json:"message"`
			Branch         string `json:"branch"`
			DeploymentUUID string `json:"deployment_uuid"`
			Status         string `json:"status"`
		}
		data := previewWaitData{
			Branch:         branch,
			DeploymentUUID: deploymentUUID,
			Status:         finalStatus,
		}

		if isDeploymentFinished(finalStatus) {
			data.Message = fmt.Sprintf("Preview deployment of %s completed successfully.", branch)
		} else {
			data.Message = fmt.Sprintf("Preview deployment of %s finished with status: %s", branch, finalStatus)
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Preview deployment complete — status: %s\n", finalStatus)
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

func pollPreview(cmd *cobra.Command, client *coolify.Client, deploymentUUID string, useJSON bool, timeoutSec, intervalSec int) (string, error) {
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
				OutputError(cmd.OutOrStdout(), "PREVIEW_TIMEOUT", msg)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", msg)
			}
			return d.Status, errExitCode1
		}
	}
}

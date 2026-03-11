package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/RazBrry/safe-ify/internal/config"
	"github.com/RazBrry/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback to a previous deployment",
	Long: `Rollback to a previous deployment by specifying a commit SHA or tag.

Use "safe-ify deployments" to view past deployments and their commit SHAs.
Use --wait to poll until the rollback deployment completes.`,
	RunE: runRollback,
}

func init() {
	rollbackCmd.Flags().String("to", "", "Commit SHA or tag to roll back to (required)")
	rollbackCmd.Flags().Bool("wait", false, "Wait for rollback deployment to complete")
	rollbackCmd.Flags().Int("timeout", 300, "Max seconds to wait (with --wait)")
	rollbackCmd.Flags().Int("poll-interval", 15, "Seconds between status polls (with --wait)")
	_ = rollbackCmd.MarkFlagRequired("to")
	rootCmd.AddCommand(rollbackCmd)
}

func runRollback(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")
	target, _ := cmd.Flags().GetString("to")
	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetInt("timeout")
	pollInterval, _ := cmd.Flags().GetInt("poll-interval")

	err := runAgentCommand(cmd, "rollback", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["rollback"] {
			err := fmt.Errorf("command %q is not permitted for this project", "rollback")
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
			}
			return nil, errExitCode1
		}

		resp, err := client.DeployByTag(context.Background(), cfg.AppUUID, target)
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
			type rollbackData struct {
				Message        string `json:"message"`
				Target         string `json:"target"`
				DeploymentUUID string `json:"deployment_uuid,omitempty"`
			}
			data := rollbackData{
				Message:        fmt.Sprintf("Rollback to %s queued.", target),
				Target:         target,
				DeploymentUUID: deploymentUUID,
			}
			if useJSON {
				OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Rollback to %s queued.", target)
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
			fmt.Fprintf(cmd.OutOrStdout(), "Rollback to %s queued (%s). Waiting for completion (timeout: %ds)...\n", target, deploymentUUID, timeout)
		}

		finalStatus, err := pollRollback(cmd, client, deploymentUUID, useJSON, timeout, pollInterval)
		if err != nil {
			return nil, err
		}

		type rollbackWaitData struct {
			Message        string `json:"message"`
			Target         string `json:"target"`
			DeploymentUUID string `json:"deployment_uuid"`
			Status         string `json:"status"`
		}
		data := rollbackWaitData{
			Target:         target,
			DeploymentUUID: deploymentUUID,
			Status:         finalStatus,
		}

		if isDeploymentFinished(finalStatus) {
			data.Message = fmt.Sprintf("Rollback to %s completed successfully.", target)
		} else {
			data.Message = fmt.Sprintf("Rollback to %s finished with status: %s", target, finalStatus)
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Rollback complete — status: %s\n", finalStatus)
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

func pollRollback(cmd *cobra.Command, client *coolify.Client, deploymentUUID string, useJSON bool, timeoutSec, intervalSec int) (string, error) {
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
				OutputError(cmd.OutOrStdout(), "ROLLBACK_TIMEOUT", msg)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", msg)
			}
			return d.Status, errExitCode1
		}
	}
}

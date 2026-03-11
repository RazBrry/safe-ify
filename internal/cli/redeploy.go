package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/RazBrry/safe-ify/internal/config"
	"github.com/RazBrry/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

var redeployCmd = &cobra.Command{
	Use:   "redeploy",
	Short: "Redeploy the current version",
	Long: `Restart/redeploy the currently deployed version of the application.

Use --wait to poll until the app is healthy again (checks every --poll-interval
seconds, times out after --timeout seconds).`,
	RunE: runRedeploy,
}

func init() {
	redeployCmd.Flags().Bool("wait", false, "Wait for restart to complete (poll app status)")
	redeployCmd.Flags().Int("timeout", 120, "Max seconds to wait for restart (with --wait)")
	redeployCmd.Flags().Int("poll-interval", 10, "Seconds between status polls (with --wait)")
	rootCmd.AddCommand(redeployCmd)
}

func runRedeploy(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")
	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetInt("timeout")
	pollInterval, _ := cmd.Flags().GetInt("poll-interval")

	err := runAgentCommand(cmd, "redeploy", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["redeploy"] {
			err := fmt.Errorf("command %q is not permitted for this project", "redeploy")
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
			}
			return nil, errExitCode1
		}

		if err := client.Restart(context.Background(), cfg.AppUUID); err != nil {
			if useJSON {
				OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
			}
			return nil, errExitCode1
		}

		if !wait {
			type redeployData struct {
				Message string `json:"message"`
			}
			data := redeployData{Message: "Restart triggered."}
			if useJSON {
				OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Restart triggered.")
			}
			return data, nil
		}

		// --wait: poll app status until healthy or timeout.
		if !useJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "Restart triggered. Waiting for healthy status (timeout: %ds)...\n", timeout)
		}

		finalStatus, err := pollRestart(cmd, client, cfg.AppUUID, useJSON, timeout, pollInterval)
		if err != nil {
			return nil, err
		}

		type redeployWaitData struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		}
		data := redeployWaitData{Status: finalStatus}

		if finalStatus == "running" || finalStatus == "running:healthy" {
			data.Message = "Restart completed successfully."
		} else {
			data.Message = fmt.Sprintf("Restart finished with status: %s", finalStatus)
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Restart complete — status: %s\n", finalStatus)
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

// pollRestart polls GetApplication until the app transitions through a restart
// (status leaves healthy, then returns to healthy) or times out.
func pollRestart(cmd *cobra.Command, client *coolify.Client, uuid string, useJSON bool, timeoutSec, intervalSec int) (string, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	interval := time.Duration(intervalSec) * time.Second
	sawRestarting := false

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

		// Track when app enters a non-healthy state (restarting).
		if app.Status != "running" && app.Status != "running:healthy" {
			sawRestarting = true
		}

		// Done when we've seen the restart happen and it's healthy again.
		if sawRestarting && (app.Status == "running" || app.Status == "running:healthy") {
			return app.Status, nil
		}

		if time.Now().After(deadline) {
			msg := fmt.Sprintf("timed out after %ds — last status: %s", timeoutSec, app.Status)
			if !sawRestarting {
				msg = fmt.Sprintf("timed out after %ds — app never left healthy state (status: %s, restart may have been instant)", timeoutSec, app.Status)
			}
			if useJSON {
				OutputError(cmd.OutOrStdout(), "RESTART_TIMEOUT", msg)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", msg)
			}
			return app.Status, errExitCode1
		}
	}
}

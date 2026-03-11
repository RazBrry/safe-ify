package cli

import (
	"context"
	"fmt"

	"github.com/erwinmaasbach/safe-ify/internal/config"
	"github.com/erwinmaasbach/safe-ify/internal/coolify"
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

		type redeployData struct {
			Message string `json:"message"`
		}
		data := redeployData{Message: "Restart triggered."}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{
				OK:   true,
				Data: data,
			})
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Restart triggered.")
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

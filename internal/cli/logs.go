package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/erwinmaasbach/safe-ify/internal/config"
	"github.com/erwinmaasbach/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Fetch recent application logs",
	Long:  "Fetch recent log lines for the application configured in .safe-ify.yaml.",
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().Int("tail", 100, "Number of log lines to fetch")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")
	tail, _ := cmd.Flags().GetInt("tail")

	err := runAgentCommand(cmd, "logs", func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["logs"] {
			err := fmt.Errorf("command %q is not permitted for this project", "logs")
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
			}
			return nil, errExitCode1
		}

		lines, err := client.GetLogs(context.Background(), cfg.AppUUID, tail)
		if err != nil {
			if useJSON {
				OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
			}
			return nil, errExitCode1
		}

		type logsData struct {
			Lines []string `json:"lines"`
			Count int      `json:"count"`
		}
		data := logsData{
			Lines: lines,
			Count: len(lines),
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{
				OK:   true,
				Data: data,
			})
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), strings.Join(lines, "\n"))
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

package cli

import (
	"context"
	"fmt"
	"strings"

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

	runtime, client, err := resolveAgentConfig(cmd)
	if err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), mapConfigError(err), err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		}
		return errExitCode1
	}

	// Check permission before making any API call.
	if err := checkPermission(runtime, "logs"); err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
		}
		return errExitCode1
	}

	tail, _ := cmd.Flags().GetInt("tail")
	lines, err := client.GetLogs(context.Background(), runtime.AppUUID, tail)
	if err != nil {
		if useJSON {
			OutputError(cmd.OutOrStdout(), ErrCodeAPIError, err.Error())
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
		}
		return errExitCode1
	}

	if useJSON {
		type logsData struct {
			Lines []string `json:"lines"`
			Count int      `json:"count"`
		}
		OutputJSON(cmd.OutOrStdout(), Response{
			OK: true,
			Data: logsData{
				Lines: lines,
				Count: len(lines),
			},
		})
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), strings.Join(lines, "\n"))
	}

	return nil
}

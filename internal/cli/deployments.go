package cli

import (
	"context"
	"fmt"

	"github.com/RazBrry/safe-ify/internal/config"
	"github.com/RazBrry/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

var deploymentsCmd = &cobra.Command{
	Use:   "deployments",
	Short: "List deployment history",
	Long:  "Show the deployment history for the application configured in .safe-ify.yaml.",
	RunE:  runDeployments,
}

func init() {
	deploymentsCmd.Flags().Int("limit", 10, "Maximum number of deployments to show")
	rootCmd.AddCommand(deploymentsCmd)
}

func runDeployments(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")
	limit, _ := cmd.Flags().GetInt("limit")

	err := runAgentCommand(cmd, "deployments", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["deployments"] {
			err := fmt.Errorf("command %q is not permitted for this project", "deployments")
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
			}
			return nil, errExitCode1
		}

		deployments, err := client.ListDeployments(context.Background(), cfg.AppUUID)
		if err != nil {
			if useJSON {
				OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
			}
			return nil, errExitCode1
		}

		// Apply limit.
		if limit > 0 && len(deployments) > limit {
			deployments = deployments[:limit]
		}

		type deploymentItem struct {
			DeploymentUUID string `json:"deployment_uuid"`
			Status         string `json:"status"`
			CommitSHA      string `json:"commit_sha,omitempty"`
			CommitMessage  string `json:"commit_message,omitempty"`
			CreatedAt      string `json:"created_at"`
		}
		items := make([]deploymentItem, len(deployments))
		for i, d := range deployments {
			items[i] = deploymentItem{
				DeploymentUUID: d.DeploymentUUID,
				Status:         d.Status,
				CommitSHA:      d.CommitSHA,
				CommitMessage:  d.CommitMessage,
				CreatedAt:      d.CreatedAt,
			}
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: items})
		} else {
			if len(items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No deployments found.")
				return items, nil
			}
			for _, item := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  %s", item.CreatedAt, item.Status, item.DeploymentUUID)
				if item.CommitSHA != "" {
					sha := item.CommitSHA
					if len(sha) > 7 {
						sha = sha[:7]
					}
					fmt.Fprintf(cmd.OutOrStdout(), "  %s", sha)
				}
				if item.CommitMessage != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s", item.CommitMessage)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
		}
		return items, nil
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

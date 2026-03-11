package cli

import (
	"context"
	"fmt"

	"github.com/RazBrry/safe-ify/internal/config"
	"github.com/RazBrry/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

var resourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Show application resource usage",
	Long:  "Display CPU, memory, network, and disk I/O metrics for the application's containers.",
	RunE:  runResources,
}

func init() {
	rootCmd.AddCommand(resourcesCmd)
}

func runResources(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")

	err := runAgentCommand(cmd, "resources", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["resources"] {
			err := fmt.Errorf("command %q is not permitted for this project", "resources")
			if useJSON {
				OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
			}
			return nil, errExitCode1
		}

		resources, err := client.GetResources(context.Background(), cfg.AppUUID)
		if err != nil {
			if useJSON {
				OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
			}
			return nil, errExitCode1
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: resources})
		} else {
			if len(resources) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No resource data available.")
				return resources, nil
			}
			for _, r := range resources {
				if r.ContainerID != "" {
					id := r.ContainerID
					if len(id) > 12 {
						id = id[:12]
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Container: %s\n", id)
				}
				if r.CPUPercent != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  CPU:    %s\n", r.CPUPercent)
				}
				if r.MemUsage != "" && r.MemLimit != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  Memory: %s / %s", r.MemUsage, r.MemLimit)
					if r.MemPercent != "" {
						fmt.Fprintf(cmd.OutOrStdout(), " (%s)", r.MemPercent)
					}
					fmt.Fprintln(cmd.OutOrStdout())
				}
				if r.NetIO != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  Net:    %s\n", r.NetIO)
				}
				if r.BlockIO != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  Disk:   %s\n", r.BlockIO)
				}
			}
		}
		return resources, nil
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

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
	Short: "Show application resource limits",
	Long:  "Display the configured resource limits (CPU, memory) for the application.",
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

		app, err := client.GetApplication(context.Background(), cfg.AppUUID)
		if err != nil {
			if useJSON {
				OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
			}
			return nil, errExitCode1
		}

		type resourcesData struct {
			UUID              string `json:"uuid"`
			Name              string `json:"name"`
			Status            string `json:"status"`
			LimitsMemory      string `json:"limits_memory,omitempty"`
			LimitsCPUs        string `json:"limits_cpus,omitempty"`
			LimitsCPUShares   int    `json:"limits_cpu_shares,omitempty"`
			LimitsMemorySwap  string `json:"limits_memory_swap,omitempty"`
			LimitsMemoryReserv string `json:"limits_memory_reservation,omitempty"`
		}
		data := resourcesData{
			UUID:              app.UUID,
			Name:              app.Name,
			Status:            app.Status,
			LimitsMemory:      app.LimitsMemory,
			LimitsCPUs:        app.LimitsCPUs,
			LimitsCPUShares:   app.LimitsCPUShares,
			LimitsMemorySwap:  app.LimitsMemorySwap,
			LimitsMemoryReserv: app.LimitsMemoryReserv,
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Application: %s (%s)\n", app.Name, app.UUID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status:      %s\n", app.Status)
			if app.LimitsCPUs != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "CPU limit:   %s\n", app.LimitsCPUs)
			}
			if app.LimitsCPUShares > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "CPU shares:  %d\n", app.LimitsCPUShares)
			}
			if app.LimitsMemory != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Memory:      %s\n", app.LimitsMemory)
			}
			if app.LimitsMemoryReserv != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Memory resv: %s\n", app.LimitsMemoryReserv)
			}
			if app.LimitsMemorySwap != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Memory swap: %s\n", app.LimitsMemorySwap)
			}
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

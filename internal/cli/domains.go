package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/RazBrry/safe-ify/internal/config"
	"github.com/RazBrry/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

var domainsCmd = &cobra.Command{
	Use:   "domains",
	Short: "Show application domains/URLs",
	Long:  "Display the configured domains and URLs for the application.",
	RunE:  runDomains,
}

func init() {
	rootCmd.AddCommand(domainsCmd)
}

func runDomains(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")

	err := runAgentCommand(cmd, "domains", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["domains"] {
			err := fmt.Errorf("command %q is not permitted for this project", "domains")
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

		// Coolify stores domains as a comma-separated FQDN field.
		var domains []string
		if app.FQDN != "" {
			for _, d := range strings.Split(app.FQDN, ",") {
				d = strings.TrimSpace(d)
				if d != "" {
					domains = append(domains, d)
				}
			}
		}

		type domainsData struct {
			UUID    string   `json:"uuid"`
			Name    string   `json:"name"`
			Domains []string `json:"domains"`
		}
		data := domainsData{
			UUID:    app.UUID,
			Name:    app.Name,
			Domains: domains,
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Application: %s (%s)\n", app.Name, app.UUID)
			if len(domains) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No domains configured.")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Domains:")
				for _, d := range domains {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", d)
				}
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

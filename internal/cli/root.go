package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	jsonOutput  bool
	configPath  string
	projectPath string
	appVersion  string
)

var rootCmd = &cobra.Command{
	Use:   "safe-ify",
	Short: "A safe CLI wrapper for Coolify deployments",
	Long: `safe-ify is a security-focused CLI tool that provides controlled,
auditable access to Coolify deployment operations.

It enforces deny-only permission policies at both global and project levels,
ensuring that agents can only perform explicitly permitted operations.`,
}

// Execute runs the root command with the given version string.
func Execute(version string) {
	appVersion = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to global config file (default: ~/.config/safe-ify/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&projectPath, "project", "", "Path to project config file (default: auto-discover .safe-ify.yaml)")
	rootCmd.PersistentFlags().Bool("version", false, "Print version and exit")

	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		versionFlag, _ := cmd.Flags().GetBool("version")
		if versionFlag {
			fmt.Printf("safe-ify %s\n", appVersion)
			return nil
		}
		return cmd.Help()
	}
}

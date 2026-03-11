package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/RazBrry/safe-ify/internal/config"
	"github.com/RazBrry/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

func init() {
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Manage application environment variables",
		Long: `Read and write environment variables for the application configured in .safe-ify.yaml.

Requires a Coolify API token with 'read:sensitive' (for reading) and 'write' (for modifying) scopes.

Permissions: 'env-read' controls list/get, 'env-write' controls set/delete.
Add these to the deny list in .safe-ify.yaml to restrict access per app.`,
	}

	envListCmd := &cobra.Command{
		Use:   "list",
		Short: "List environment variables",
		Long:  "List all environment variable keys for the application. Values are shown only with --show-values.",
		RunE:  runEnvList,
	}
	envListCmd.Flags().Bool("show-values", false, "Include values in output (requires read:sensitive token scope)")
	envListCmd.Flags().Bool("preview", false, "Show only preview environment variables")

	envGetCmd := &cobra.Command{
		Use:   "get",
		Short: "Get an environment variable value",
		Long:  "Get the value of a specific environment variable by key.",
		RunE:  runEnvGet,
	}
	envGetCmd.Flags().String("key", "", "Environment variable key (required)")
	_ = envGetCmd.MarkFlagRequired("key")
	envGetCmd.Flags().Bool("preview", false, "Get from preview environment")

	envSetCmd := &cobra.Command{
		Use:   "set",
		Short: "Set an environment variable",
		Long:  "Create or update an environment variable. Creates if it doesn't exist, updates if it does.",
		RunE:  runEnvSet,
	}
	envSetCmd.Flags().String("key", "", "Environment variable key (required)")
	_ = envSetCmd.MarkFlagRequired("key")
	envSetCmd.Flags().String("value", "", "Environment variable value (required)")
	_ = envSetCmd.MarkFlagRequired("value")
	envSetCmd.Flags().Bool("preview", false, "Set for preview environment")
	envSetCmd.Flags().Bool("build-only", false, "Available at build time only (not runtime)")
	envSetCmd.Flags().Bool("runtime-only", false, "Available at runtime only (not build time)")

	envDeleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an environment variable",
		Long:  "Delete an environment variable by key. Looks up the env var UUID by key, then deletes it.",
		RunE:  runEnvDelete,
	}
	envDeleteCmd.Flags().String("key", "", "Environment variable key (required)")
	_ = envDeleteCmd.MarkFlagRequired("key")
	envDeleteCmd.Flags().Bool("preview", false, "Delete from preview environment")

	envCmd.AddCommand(envListCmd, envGetCmd, envSetCmd, envDeleteCmd)
	rootCmd.AddCommand(envCmd)
}

func runEnvList(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")
	showValues, _ := cmd.Flags().GetBool("show-values")
	preview, _ := cmd.Flags().GetBool("preview")

	err := runAgentCommand(cmd, "env-read", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["env-read"] {
			return nil, permissionDenied(cmd, useJSON, "env-read")
		}

		envs, err := client.ListEnvs(context.Background(), cfg.AppUUID)
		if err != nil {
			return nil, apiError(cmd, useJSON, err)
		}

		// Filter by preview flag.
		var filtered []coolify.EnvVar
		for _, e := range envs {
			if e.IsPreview == preview {
				filtered = append(filtered, e)
			}
		}

		type envEntry struct {
			Key         string `json:"key"`
			Value       string `json:"value,omitempty"`
			IsRuntime   bool   `json:"is_runtime"`
			IsBuildtime bool   `json:"is_buildtime"`
			IsPreview   bool   `json:"is_preview"`
			UUID        string `json:"uuid"`
		}
		type envListData struct {
			Envs  []envEntry `json:"envs"`
			Count int        `json:"count"`
		}

		entries := make([]envEntry, len(filtered))
		for i, e := range filtered {
			entries[i] = envEntry{
				Key:         e.Key,
				IsRuntime:   e.IsRuntime,
				IsBuildtime: e.IsBuildtime,
				IsPreview:   e.IsPreview,
				UUID:        e.UUID,
			}
			if showValues {
				entries[i].Value = e.Value
			}
		}

		data := envListData{Envs: entries, Count: len(entries)}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
		} else {
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No environment variables found.")
				return data, nil
			}
			scope := "production"
			if preview {
				scope = "preview"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Environment variables (%s) for %s:\n\n", scope, cfg.AppName)
			if showValues {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %-50s %s\n", "Key", "Value", "Scope")
				fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %-50s %s\n", "---", "-----", "-----")
				for _, e := range entries {
					val := e.Value
					if len(val) > 47 {
						val = val[:47] + "..."
					}
					fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %-50s %s\n", e.Key, val, envScope(e.IsRuntime, e.IsBuildtime))
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %s\n", "Key", "Scope")
				fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %s\n", "---", "-----")
				for _, e := range entries {
					fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %s\n", e.Key, envScope(e.IsRuntime, e.IsBuildtime))
				}
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprintln(cmd.OutOrStdout(), "Use --show-values to include values (requires read:sensitive token scope).")
			}
		}
		return data, nil
	})
	return wrapAgentError(cmd, useJSON, err)
}

func runEnvGet(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")
	key, _ := cmd.Flags().GetString("key")
	preview, _ := cmd.Flags().GetBool("preview")

	err := runAgentCommand(cmd, "env-read", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["env-read"] {
			return nil, permissionDenied(cmd, useJSON, "env-read")
		}

		envs, err := client.ListEnvs(context.Background(), cfg.AppUUID)
		if err != nil {
			return nil, apiError(cmd, useJSON, err)
		}

		for _, e := range envs {
			if e.Key == key && e.IsPreview == preview {
				type envGetData struct {
					Key         string `json:"key"`
					Value       string `json:"value"`
					IsRuntime   bool   `json:"is_runtime"`
					IsBuildtime bool   `json:"is_buildtime"`
					IsPreview   bool   `json:"is_preview"`
					UUID        string `json:"uuid"`
				}
				data := envGetData{
					Key:         e.Key,
					Value:       e.Value,
					IsRuntime:   e.IsRuntime,
					IsBuildtime: e.IsBuildtime,
					IsPreview:   e.IsPreview,
					UUID:        e.UUID,
				}
				if useJSON {
					OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", e.Key, e.Value)
				}
				return data, nil
			}
		}

		msg := fmt.Sprintf("environment variable %q not found", key)
		if useJSON {
			OutputError(cmd.OutOrStdout(), "ENV_NOT_FOUND", msg)
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", msg)
		}
		return nil, errExitCode1
	})
	return wrapAgentError(cmd, useJSON, err)
}

func runEnvSet(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")
	key, _ := cmd.Flags().GetString("key")
	value, _ := cmd.Flags().GetString("value")
	preview, _ := cmd.Flags().GetBool("preview")
	buildOnly, _ := cmd.Flags().GetBool("build-only")
	runtimeOnly, _ := cmd.Flags().GetBool("runtime-only")

	err := runAgentCommand(cmd, "env-write", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["env-write"] {
			return nil, permissionDenied(cmd, useJSON, "env-write")
		}

		isRuntime := true
		isBuildtime := true
		if buildOnly {
			isRuntime = false
		}
		if runtimeOnly {
			isBuildtime = false
		}

		// Try create first; if 409 (already exists), update.
		createReq := coolify.CreateEnvRequest{
			Key:         key,
			Value:       value,
			IsPreview:   &preview,
			IsRuntime:   &isRuntime,
			IsBuildtime: &isBuildtime,
		}

		envUUID, createErr := client.CreateEnv(context.Background(), cfg.AppUUID, createReq)
		action := "created"
		if createErr != nil {
			if coolifyErr, ok := createErr.(*coolify.CoolifyError); ok && coolifyErr.StatusCode == 409 {
				updateReq := coolify.UpdateEnvRequest{
					Key:         key,
					Value:       value,
					IsPreview:   &preview,
					IsRuntime:   &isRuntime,
					IsBuildtime: &isBuildtime,
				}
				if updateErr := client.UpdateEnv(context.Background(), cfg.AppUUID, updateReq); updateErr != nil {
					return nil, apiError(cmd, useJSON, updateErr)
				}
				action = "updated"
				envUUID = ""
			} else {
				return nil, apiError(cmd, useJSON, createErr)
			}
		}

		type envSetData struct {
			Message string `json:"message"`
			Key     string `json:"key"`
			Action  string `json:"action"`
			UUID    string `json:"uuid,omitempty"`
		}
		data := envSetData{
			Message: fmt.Sprintf("Environment variable %q %s.", key, action),
			Key:     key,
			Action:  action,
			UUID:    envUUID,
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Environment variable %q %s.\n", key, action)
		}
		return data, nil
	})
	return wrapAgentError(cmd, useJSON, err)
}

func runEnvDelete(cmd *cobra.Command, args []string) error {
	useJSON, _ := cmd.Root().PersistentFlags().GetBool("json")
	key, _ := cmd.Flags().GetString("key")
	preview, _ := cmd.Flags().GetBool("preview")

	err := runAgentCommand(cmd, "env-write", true, func(cfg *config.RuntimeConfig, client *coolify.Client) (interface{}, error) {
		if !cfg.AllowedCmds["env-write"] {
			return nil, permissionDenied(cmd, useJSON, "env-write")
		}

		// Look up env UUID by key.
		envs, err := client.ListEnvs(context.Background(), cfg.AppUUID)
		if err != nil {
			return nil, apiError(cmd, useJSON, err)
		}

		var envUUID string
		for _, e := range envs {
			if e.Key == key && e.IsPreview == preview {
				envUUID = e.UUID
				break
			}
		}

		if envUUID == "" {
			msg := fmt.Sprintf("environment variable %q not found", key)
			if useJSON {
				OutputError(cmd.OutOrStdout(), "ENV_NOT_FOUND", msg)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", msg)
			}
			return nil, errExitCode1
		}

		if err := client.DeleteEnv(context.Background(), cfg.AppUUID, envUUID); err != nil {
			return nil, apiError(cmd, useJSON, err)
		}

		type envDeleteData struct {
			Message string `json:"message"`
			Key     string `json:"key"`
		}
		data := envDeleteData{
			Message: fmt.Sprintf("Environment variable %q deleted.", key),
			Key:     key,
		}

		if useJSON {
			OutputJSON(cmd.OutOrStdout(), Response{OK: true, Data: data})
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Environment variable %q deleted.\n", key)
		}
		return data, nil
	})
	return wrapAgentError(cmd, useJSON, err)
}

// wrapAgentError handles config resolution errors from runAgentCommand.
func wrapAgentError(cmd *cobra.Command, useJSON bool, err error) error {
	if err == nil || err == errExitCode1 {
		return err
	}
	if useJSON {
		OutputError(cmd.OutOrStdout(), mapConfigError(err), err.Error())
	} else {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
	}
	return errExitCode1
}

// permissionDenied outputs a permission denied error and returns errExitCode1.
func permissionDenied(cmd *cobra.Command, useJSON bool, command string) error {
	err := fmt.Errorf("command %q is not permitted for this project", command)
	if useJSON {
		OutputError(cmd.OutOrStdout(), ErrCodePermissionDenied, err.Error())
	} else {
		fmt.Fprintf(cmd.ErrOrStderr(), "Permission denied: %s\n", err)
	}
	return errExitCode1
}

// apiError outputs a Coolify API error and returns errExitCode1.
func apiError(cmd *cobra.Command, useJSON bool, err error) error {
	if useJSON {
		OutputError(cmd.OutOrStdout(), mapCoolifyError(err), err.Error())
	} else {
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
	}
	return errExitCode1
}

// envScope returns a human-readable scope string.
func envScope(isRuntime, isBuildtime bool) string {
	var parts []string
	if isRuntime {
		parts = append(parts, "runtime")
	}
	if isBuildtime {
		parts = append(parts, "build")
	}
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, "+")
}

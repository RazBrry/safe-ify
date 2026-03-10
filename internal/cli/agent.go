package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/erwinmaasbach/safe-ify/internal/config"
	"github.com/erwinmaasbach/safe-ify/internal/coolify"
	"github.com/spf13/cobra"
)

// errExitCode1 is a sentinel error that signals the command should exit with
// code 1 without printing an additional error message (output was already written).
var errExitCode1 = errors.New("")

// resolveAgentConfig loads project + global configs, resolves the runtime
// config, and returns a ready-to-use Coolify client.
func resolveAgentConfig(cmd *cobra.Command) (*config.RuntimeConfig, *coolify.Client, error) {
	// 1. Find and load project config (with parent traversal).
	projectOverride, _ := cmd.Root().PersistentFlags().GetString("project")
	var (
		projectPath string
		err         error
	)
	if projectOverride != "" {
		projectPath = projectOverride
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, nil, fmt.Errorf("cannot determine working directory: %w", err)
		}
		found, err := config.FindProjectConfig(cwd)
		if err != nil {
			return nil, nil, err
		}
		projectPath = found
	}

	projectCfg, err := config.LoadProject(projectPath)
	if err != nil {
		return nil, nil, err
	}

	// 2. Load global config.
	globalOverride, _ := cmd.Root().PersistentFlags().GetString("config")
	var globalPath string
	if globalOverride != "" {
		globalPath = globalOverride
	} else {
		globalPath, err = config.DefaultGlobalConfigPath()
		if err != nil {
			return nil, nil, err
		}
	}

	globalCfg, err := config.LoadGlobal(globalPath)
	if err != nil {
		return nil, nil, err
	}

	// 3. Resolve runtime config (merges global + project deny lists).
	runtime, err := config.ResolveRuntime(globalCfg, projectCfg)
	if err != nil {
		return nil, nil, err
	}

	// 4. Create Coolify client.
	client := coolify.NewClient(runtime.InstanceURL, runtime.Token)

	return runtime, client, nil
}

// checkPermission returns an error if the given command is denied in the
// runtime config's AllowedCmds map.
func checkPermission(runtime *config.RuntimeConfig, command string) error {
	if !runtime.AllowedCmds[command] {
		return fmt.Errorf("command %q is not permitted for this project", command)
	}
	return nil
}

// mapConfigError maps a config-layer error to the appropriate JSON error code.
func mapConfigError(err error) string {
	switch err.(type) {
	case *config.ProjectConfigNotFoundError:
		return ErrCodeConfigNotFound
	case *config.ConfigNotFoundError:
		return ErrCodeConfigNotFound
	case *config.ConfigInsecureError:
		return ErrCodeConfigInsecure
	case *config.InstanceNotFoundError:
		return ErrCodeInstanceNotFound
	default:
		return ErrCodeAPIError
	}
}

package config

import (
	"fmt"
	"os"
)

// ConfigInsecureError is returned when a config file has permissions more
// open than the expected value (e.g., 0644 instead of 0600).
type ConfigInsecureError struct {
	Path     string
	Current  os.FileMode
	Expected os.FileMode
}

func (e *ConfigInsecureError) Error() string {
	return fmt.Sprintf(
		"config file %q has insecure permissions %04o (expected %04o): "+
			"run `chmod 600 %s` to fix",
		e.Path, e.Current, e.Expected, e.Path,
	)
}

// ConfigNotFoundError is returned when the global config file does not exist.
type ConfigNotFoundError struct {
	Path string
}

func (e *ConfigNotFoundError) Error() string {
	return fmt.Sprintf(
		"global config not found at %q: run `safe-ify auth add` to create it",
		e.Path,
	)
}

// ProjectConfigNotFoundError is returned when no .safe-ify.yaml can be found
// by traversing parent directories from the given search root.
type ProjectConfigNotFoundError struct {
	SearchRoot string
}

func (e *ProjectConfigNotFoundError) Error() string {
	return fmt.Sprintf(
		"no project config (.safe-ify.yaml) found starting from %q: "+
			"run `safe-ify init` to create one",
		e.SearchRoot,
	)
}

// InstanceNotFoundError is returned when the project config references an
// instance name that is not present in the global config.
type InstanceNotFoundError struct {
	Name string
}

func (e *InstanceNotFoundError) Error() string {
	return fmt.Sprintf(
		"instance %q not found in global config: run `safe-ify auth add` to add it",
		e.Name,
	)
}

// PermissionDeniedError is returned when a command is denied by a permission policy.
type PermissionDeniedError struct {
	Command  string
	DeniedBy string // "global" or "project"
}

func (e *PermissionDeniedError) Error() string {
	return fmt.Sprintf(
		"command %q is not permitted for this project (denied by %s config)",
		e.Command, e.DeniedBy,
	)
}

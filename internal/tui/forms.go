package tui

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
)

// AppOption represents a single Coolify application for display in a picker.
type AppOption struct {
	Name string
	UUID string
}

// instanceNameRE validates that an instance name contains only alphanumeric
// characters and hyphens.
var instanceNameRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)

// AuthAddValues holds the values collected by AuthAddForm.
type AuthAddValues struct {
	Name  string
	URL   string
	Token string
}

// AuthAddForm builds and returns a huh.Form that collects an instance name,
// Coolify URL, and API token from the user.
func AuthAddForm(values *AuthAddValues) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Instance name").
				Description("A short identifier for this Coolify instance (alphanumeric + hyphens).").
				Value(&values.Name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("instance name is required")
					}
					if !instanceNameRE.MatchString(s) {
						return fmt.Errorf("instance name must contain only alphanumeric characters and hyphens")
					}
					return nil
				}),

			huh.NewInput().
				Title("Coolify URL").
				Description("The URL of your Coolify instance (e.g., coolify.example.com).").
				Value(&values.URL).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("URL is required")
					}
					// For validation, test with scheme prepended
					test := s
					if !strings.Contains(test, "://") {
						test = "https://" + test
					}
					u, err := url.ParseRequestURI(test)
					if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
						return fmt.Errorf("must be a valid URL (e.g., coolify.example.com)")
					}
					return nil
				}),

			huh.NewInput().
				Title("API token").
				Description("Your Coolify API token.").
				Password(true).
				Value(&values.Token).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("API token is required")
					}
					return nil
				}),
		),
	)
}

// AuthRemoveValues holds the values collected by AuthRemoveForm.
type AuthRemoveValues struct {
	Selected string
	Confirm  bool
}

// AuthRemoveForm builds and returns a huh.Form that presents a select list of
// instance names and a confirmation prompt.
func AuthRemoveForm(instances []string, values *AuthRemoveValues) *huh.Form {
	options := make([]huh.Option[string], len(instances))
	for i, name := range instances {
		options[i] = huh.NewOption(name, name)
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select instance to remove").
				Options(options...).
				Value(&values.Selected),

			huh.NewConfirm().
				Title("Are you sure you want to remove this instance?").
				Description("This will delete the credentials from your global config.").
				Value(&values.Confirm),
		),
	)
}

// InitSelectInstanceForm builds and returns a huh.Form that presents a select
// list of configured instance names.
func InitSelectInstanceForm(instances []string, selected *string) *huh.Form {
	options := make([]huh.Option[string], len(instances))
	for i, name := range instances {
		options[i] = huh.NewOption(name, name)
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Coolify instance").
				Description("Choose which Coolify instance to link this project to.").
				Options(options...).
				Value(selected),
		),
	)
}

// InitSelectAppForm builds and returns a huh.Form that presents a select list
// of applications. Each option displays the app name together with its UUID.
func InitSelectAppForm(apps []AppOption, selected *string) *huh.Form {
	options := make([]huh.Option[string], len(apps))
	for i, app := range apps {
		label := fmt.Sprintf("%s (%s)", app.Name, app.UUID)
		options[i] = huh.NewOption(label, app.UUID)
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select application").
				Description("Choose the Coolify application to deploy for this project.").
				Options(options...).
				Value(selected),
		),
	)
}

// InitPermissionsForm builds and returns a huh.Form with a multi-select of all
// agent commands. Selected commands will be DENIED for this project.
func InitPermissionsForm(allCommands []string, denied *[]string) *huh.Form {
	options := make([]huh.Option[string], len(allCommands))
	for i, cmd := range allCommands {
		options[i] = huh.NewOption(cmd, cmd)
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Permission deny list").
				Description("Select commands to DENY for this project. Leave empty to allow all commands.").
				Options(options...).
				Value(denied),
		),
	)
}

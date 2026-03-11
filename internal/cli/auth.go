package cli

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/erwinmaasbach/safe-ify/internal/config"
	"github.com/erwinmaasbach/safe-ify/internal/tui"
)

func init() {
	authCmd := newAuthCmd()
	rootCmd.AddCommand(authCmd)
}

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Coolify instance credentials",
		Long:  "Add, remove, and list Coolify instance credentials stored in the global config.",
	}

	cmd.AddCommand(newAuthAddCmd())
	cmd.AddCommand(newAuthRemoveCmd())
	cmd.AddCommand(newAuthListCmd())

	return cmd
}

// resolveConfigPath returns the effective global config path: use the --config
// flag value when set, otherwise fall back to the default.
func resolveConfigPath() (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	return config.DefaultGlobalConfigPath()
}

// loadOrCreateGlobal loads the global config if it exists; if not found it
// returns an empty, initialised GlobalConfig.
func loadOrCreateGlobal(path string) (*config.GlobalConfig, error) {
	cfg, err := config.LoadGlobal(path)
	if err != nil {
		var notFound *config.ConfigNotFoundError
		if errors.As(err, &notFound) {
			return &config.GlobalConfig{
				Instances: make(map[string]config.Instance),
			}, nil
		}
		return nil, err
	}
	return cfg, nil
}

// validateToken verifies connectivity and authentication by trying
// /api/v1/version (authenticated). Falls back to /api/v1/healthcheck if
// version returns 404 (older Coolify versions).
func validateToken(rawURL, token string) error {
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	base := strings.TrimRight(rawURL, "/")
	client := &http.Client{Timeout: 15 * time.Second}

	// Try /api/v1/version first (requires valid token).
	for _, path := range []string{"/api/v1/version", "/api/v1/healthcheck"} {
		endpoint := base + path

		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return fmt.Errorf("cannot build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("User-Agent", "safe-ify/1.0")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("cannot reach Coolify at %s: %w", rawURL, err)
		}
		resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK, http.StatusNoContent:
			return nil
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed (401): check your API token")
		case http.StatusNotFound:
			continue // try next endpoint
		default:
			return fmt.Errorf("%s returned unexpected status %d", path, resp.StatusCode)
		}
	}

	return fmt.Errorf("could not validate token: neither /api/v1/version nor /api/v1/healthcheck responded (both returned 404)")
}

// maskToken returns the first 4 characters of the token followed by "****",
// or "****" if the token is shorter than 4 characters.
func maskToken(token string) string {
	if len(token) <= 4 {
		return "****"
	}
	return token[:4] + "****"
}

// ------------------------------------------------------------------
// auth add
// ------------------------------------------------------------------

func newAuthAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a Coolify instance",
		Long:  "Interactively add a Coolify instance by providing a name, URL, and API token.",
		RunE:  runAuthAdd,
	}
}

func runAuthAdd(cmd *cobra.Command, args []string) error {
	cfgPath, err := resolveConfigPath()
	if err != nil {
		return err
	}

	values := &tui.AuthAddValues{}

	if err := tui.AuthAddForm(values).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("form error: %w", err)
	}

	// Normalize URL: ensure scheme is present.
	if !strings.Contains(values.URL, "://") {
		values.URL = "https://" + values.URL
	}
	values.URL = strings.TrimRight(values.URL, "/")

	// Validate token via healthcheck.
	fmt.Println(tui.InfoStyle.Render("Validating token..."))
	if err := validateToken(values.URL, values.Token); err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	// Load or create global config.
	cfg, err := loadOrCreateGlobal(cfgPath)
	if err != nil {
		return err
	}

	// Handle existing instance name.
	if _, exists := cfg.Instances[values.Name]; exists {
		var overwrite bool
		confirm := huh.NewConfirm().
			Title(fmt.Sprintf("Instance %q already exists. Overwrite?", values.Name)).
			Value(&overwrite)
		if err := confirm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Println(tui.InfoStyle.Render("Aborted."))
				return nil
			}
			return fmt.Errorf("confirmation error: %w", err)
		}
		if !overwrite {
			fmt.Println(tui.InfoStyle.Render("Cancelled. Use a different instance name and try again."))
			return nil
		}
	}

	// Add instance and save.
	cfg.Instances[values.Name] = config.Instance{
		URL:   values.URL,
		Token: values.Token,
	}

	if err := config.SaveGlobal(cfgPath, cfg); err != nil {
		return fmt.Errorf("cannot save global config: %w", err)
	}

	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("Instance %q added successfully.", values.Name)))
	return nil
}

// ------------------------------------------------------------------
// auth remove
// ------------------------------------------------------------------

func newAuthRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove a Coolify instance",
		Long:  "Interactively remove a Coolify instance from the global config.",
		RunE:  runAuthRemove,
	}
}

func runAuthRemove(cmd *cobra.Command, args []string) error {
	cfgPath, err := resolveConfigPath()
	if err != nil {
		return err
	}

	cfg, err := config.LoadGlobal(cfgPath)
	if err != nil {
		var notFound *config.ConfigNotFoundError
		if errors.As(err, &notFound) {
			fmt.Println(tui.InfoStyle.Render("No instances configured. Run `safe-ify auth add` to add one."))
			return nil
		}
		return err
	}

	if len(cfg.Instances) == 0 {
		fmt.Println(tui.InfoStyle.Render("No instances configured. Run `safe-ify auth add` to add one."))
		return nil
	}

	// Collect instance names for the picker.
	names := make([]string, 0, len(cfg.Instances))
	for name := range cfg.Instances {
		names = append(names, name)
	}

	values := &tui.AuthRemoveValues{}
	if err := tui.AuthRemoveForm(names, values).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println(tui.InfoStyle.Render("Aborted."))
			return nil
		}
		return fmt.Errorf("form error: %w", err)
	}

	if !values.Confirm {
		fmt.Println(tui.InfoStyle.Render("Removal cancelled."))
		return nil
	}

	delete(cfg.Instances, values.Selected)

	if err := config.SaveGlobal(cfgPath, cfg); err != nil {
		return fmt.Errorf("cannot save global config: %w", err)
	}

	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("Instance %q removed.", values.Selected)))
	return nil
}

// ------------------------------------------------------------------
// auth list
// ------------------------------------------------------------------

func newAuthListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured Coolify instances",
		Long:  "Display all configured Coolify instances with their URL and masked token.",
		RunE:  runAuthList,
	}
}

func runAuthList(cmd *cobra.Command, args []string) error {
	cfgPath, err := resolveConfigPath()
	if err != nil {
		return err
	}

	cfg, err := config.LoadGlobal(cfgPath)
	if err != nil {
		var notFound *config.ConfigNotFoundError
		if errors.As(err, &notFound) {
			fmt.Println(tui.InfoStyle.Render("No instances configured. Run `safe-ify auth add` to add one."))
			return nil
		}
		return err
	}

	if len(cfg.Instances) == 0 {
		fmt.Println(tui.InfoStyle.Render("No instances configured. Run `safe-ify auth add` to add one."))
		return nil
	}

	// Compute column widths.
	const (
		colName  = "NAME"
		colURL   = "URL"
		colToken = "TOKEN"
	)

	maxName := len(colName)
	maxURL := len(colURL)
	for name, inst := range cfg.Instances {
		if len(name) > maxName {
			maxName = len(name)
		}
		if len(inst.URL) > maxURL {
			maxURL = len(inst.URL)
		}
	}

	// Build format string with padding.
	fmtRow := func(name, url, token string) string {
		return tui.TableCellStyle.Render(fmt.Sprintf("%-*s", maxName+2, name)) +
			tui.TableCellStyle.Render(fmt.Sprintf("%-*s", maxURL+2, url)) +
			tui.TableCellStyle.Render(token)
	}

	// Header.
	header := tui.TableHeaderStyle.Render(fmt.Sprintf("%-*s", maxName+2, colName)) +
		tui.TableHeaderStyle.Render(fmt.Sprintf("%-*s", maxURL+2, colURL)) +
		tui.TableHeaderStyle.Render(colToken)

	sep := tui.TableBorderStyle.Render(strings.Repeat("-", maxName+maxURL+len(colToken)+6))

	fmt.Println(header)
	fmt.Println(sep)

	for name, inst := range cfg.Instances {
		fmt.Println(fmtRow(name, inst.URL, maskToken(inst.Token)))
	}

	return nil
}

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/RazBrry/safe-ify/internal/tui"
)

const (
	githubRepo   = "RazBrry/safe-ify"
	goImportPath = "github.com/RazBrry/safe-ify/cmd/safe-ify"
)

func init() {
	rootCmd.AddCommand(newUpdateCmd())
}

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update safe-ify to the latest version",
		Long:  "Checks for the latest version on GitHub and updates the binary in place.",
		RunE:  runUpdate,
	}
}

// githubRelease is the minimal shape of a GitHub release API response.
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

// githubAsset is a single asset attached to a GitHub release.
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Printf("  Current version: %s\n", appVersion)

	// Check latest release from GitHub.
	latest, err := fetchLatestRelease()
	if err != nil {
		// If no releases exist yet, fall back to go install without version comparison.
		if strings.Contains(err.Error(), "no releases found") {
			fmt.Println(tui.InfoStyle.Render("No GitHub releases found. Attempting go install..."))
			return updateViaGoInstall()
		}
		return fmt.Errorf("cannot check for updates: %w", err)
	}

	fmt.Printf("  Latest version:  %s\n", latest.TagName)

	latestNorm := strings.TrimPrefix(latest.TagName, "v")
	currentNorm := strings.TrimPrefix(appVersion, "v")
	if latestNorm == currentNorm {
		fmt.Println(tui.SuccessStyle.Render("Already up to date."))
		return nil
	}

	// Strategy 1: if Go is available, use go install.
	if _, err := exec.LookPath("go"); err == nil {
		return updateViaGoInstall()
	}

	// Strategy 2: download binary from GitHub release.
	return updateFromRelease(latest)
}

func updateViaGoInstall() error {
	fmt.Println(tui.InfoStyle.Render("Updating via go install..."))

	goCmd := exec.Command("go", "install", goImportPath+"@latest")
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr

	if err := goCmd.Run(); err != nil {
		return fmt.Errorf("go install failed: %w", err)
	}

	fmt.Println(tui.SuccessStyle.Render("Updated successfully."))
	return nil
}

func updateFromRelease(release *githubRelease) error {
	// Match asset for current OS/arch.
	assetName := fmt.Sprintf("safe-ify_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName += ".exe"
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no release binary found for %s/%s — install Go and run `go install %s@latest` instead", runtime.GOOS, runtime.GOARCH, goImportPath)
	}

	fmt.Println(tui.InfoStyle.Render(fmt.Sprintf("Downloading %s...", assetName)))

	// Download to a temp file, then replace current binary.
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine current binary path: %w", err)
	}
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("cannot resolve binary path: %w", err)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(currentBinary), "safe-ify-update-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("download write failed: %w", err)
	}
	tmpFile.Close()

	// Make executable.
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot set permissions: %w", err)
	}

	// Replace the current binary.
	if err := os.Rename(tmpPath, currentBinary); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot replace binary: %w", err)
	}

	fmt.Println(tui.SuccessStyle.Render(fmt.Sprintf("Updated to %s.", release.TagName)))
	return nil
}

func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "safe-ify/update")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found for %s", githubRepo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("cannot parse release response: %w", err)
	}

	return &release, nil
}

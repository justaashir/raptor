package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

const githubRepo = "justaashir/raptor"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update raptor to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check latest version from GitHub
		url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to check version: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to check latest release: %s", resp.Status)
		}

		var release struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return fmt.Errorf("failed to parse release: %w", err)
		}

		latest := release.TagName
		if len(latest) > 0 && latest[0] == 'v' {
			latest = latest[1:]
		}

		if latest == Version {
			fmt.Printf("Already up to date (%s)\n", Version)
			return nil
		}

		fmt.Printf("Updating %s → %s...\n", Version, latest)

		// Download binary from GitHub release
		assetName := fmt.Sprintf("raptor-%s-%s", runtime.GOOS, runtime.GOARCH)
		dlURL := fmt.Sprintf("https://github.com/%s/releases/latest/download/%s", githubRepo, assetName)
		dlResp, err := http.Get(dlURL)
		if err != nil {
			return fmt.Errorf("failed to download: %w", err)
		}
		defer dlResp.Body.Close()

		if dlResp.StatusCode != http.StatusOK {
			return fmt.Errorf("download failed: %s", dlResp.Status)
		}

		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to find executable path: %w", err)
		}

		tmp, err := os.CreateTemp("", "raptor-update-*")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmp.Name())

		if _, err := io.Copy(tmp, dlResp.Body); err != nil {
			tmp.Close()
			return fmt.Errorf("failed to write update: %w", err)
		}
		tmp.Close()

		if err := os.Chmod(tmp.Name(), 0o755); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}

		if err := os.Rename(tmp.Name(), exe); err != nil {
			return fmt.Errorf("failed to replace binary: %w", err)
		}

		fmt.Printf("Updated to %s\n", latest)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

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

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update raptor to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check latest version
		resp, err := http.Get(serverURL + "/api/version")
		if err != nil {
			return fmt.Errorf("failed to check version: %w", err)
		}
		defer resp.Body.Close()

		var info struct {
			Version string `json:"version"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return fmt.Errorf("failed to parse version: %w", err)
		}

		if info.Version == Version {
			fmt.Printf("Already up to date (%s)\n", Version)
			return nil
		}

		fmt.Printf("Updating %s → %s...\n", Version, info.Version)

		// Download new binary
		url := fmt.Sprintf("%s/releases/%s/%s", serverURL, runtime.GOOS, runtime.GOARCH)
		dlResp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to download: %w", err)
		}
		defer dlResp.Body.Close()

		if dlResp.StatusCode != http.StatusOK {
			return fmt.Errorf("download failed: %s", dlResp.Status)
		}

		// Get current binary path
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to find executable path: %w", err)
		}

		// Write to temp file next to current binary
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

		// Replace current binary
		if err := os.Rename(tmp.Name(), exe); err != nil {
			return fmt.Errorf("failed to replace binary: %w", err)
		}

		fmt.Printf("Updated to %s\n", info.Version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

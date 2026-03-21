package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const githubRepo = "justaashir/raptor"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update raptor to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check latest version from server (same source as update notification)
		latest := fetchServerVersion()
		if latest == "" {
			return fmt.Errorf("failed to check version from server (%s)", serverURL)
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

		tmp, err := os.CreateTemp(filepath.Dir(exe), "raptor-update-*")
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

		// Install shell completions
		installCompletion()

		// Update Claude Code skill (enforce HTTPS, size limit)
		skillURL := serverURL + "/api/skill"
		if strings.HasPrefix(skillURL, "http://") && !strings.Contains(skillURL, "localhost") && !strings.Contains(skillURL, "127.0.0.1") {
			fmt.Println("Skipping skill update: server is not HTTPS")
		} else {
			skillResp, err := http.Get(skillURL)
			if err == nil && skillResp.StatusCode == http.StatusOK {
				defer skillResp.Body.Close()
				const maxSkillSize = 64 * 1024 // 64KB
				skillData, _ := io.ReadAll(io.LimitReader(skillResp.Body, maxSkillSize))
				if len(skillData) > 0 {
					homeDir, _ := os.UserHomeDir()
					skillDir := homeDir + "/.claude/skills/raptor"
					os.MkdirAll(skillDir, 0o755)
					os.WriteFile(skillDir+"/SKILL.md", skillData, 0o644)
					fmt.Println("Claude Code skill updated.")
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

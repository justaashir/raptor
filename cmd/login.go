package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"raptor/client"
	"strings"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with GitHub",
	Long:  "Uses `gh api user` to get your GitHub username and obtains a token from the server.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get GitHub username via gh CLI
		username, err := getGitHubUsername()
		if err != nil {
			return fmt.Errorf("failed to get GitHub username: %w\nMake sure `gh` is installed and you're logged in (gh auth login)", err)
		}
		fmt.Printf("GitHub user: %s\n", username)

		// Request token from server
		body, _ := json.Marshal(map[string]string{"username": username})
		resp, err := http.Post(serverURL+"/api/auth", "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("server unreachable: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("user %q is not in the server's allowlist", username)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("auth failed with status %d", resp.StatusCode)
		}

		var result struct {
			Token    string `json:"token"`
			Username string `json:"username"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse auth response: %w", err)
		}

		// Save config
		cfg := Config{
			Server:   serverURL,
			Token:    result.Token,
			Username: result.Username,
		}

		// Auto-select workspace/board if user has exactly one of each
		c := client.New(serverURL, result.Token)
		workspaces, err := c.ListWorkspaces()
		if err == nil && len(workspaces) == 1 {
			cfg.Workspace = workspaces[0].ID
			fmt.Printf("Auto-selected workspace: %s\n", workspaces[0].Name)

			boards, err := c.ListBoards(workspaces[0].ID)
			if err == nil && len(boards) == 1 {
				cfg.Board = boards[0].ID
				fmt.Printf("Auto-selected board: %s\n", boards[0].Name)
			}
		}

		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Logged in as %s. Token saved to ~/.raptor.json\n", result.Username)
		return nil
	},
}

func getGitHubUsername() (string, error) {
	out, err := exec.Command("gh", "api", "user", "--jq", ".login").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

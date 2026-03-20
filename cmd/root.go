package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"raptor/tui"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// Set via -ldflags "-X raptor/cmd.Version=... -X raptor/cmd.DefaultServer=..."
var (
	Version       = "dev"
	DefaultServer = "http://localhost:8080"
)

var serverURL string

var rootCmd = &cobra.Command{
	Use:   "raptor",
	Short: "A multiplayer kanban board",
	Long:  "Raptor is a CLI kanban board with real-time sync.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip update check for serve/version/update commands
		name := cmd.Name()
		if name == "serve" || name == "version" || name == "update" {
			return
		}
		go checkForUpdate()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := tui.NewApp(serverURL)
		p := tea.NewProgram(app, tea.WithAltScreen())
		_, err := p.Run()
		return err
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", DefaultServer, "server URL")
}

func checkForUpdate() {
	if Version == "dev" {
		return
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(serverURL + "/api/version")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var info struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return
	}
	if info.Version != "" && info.Version != Version {
		fmt.Fprintf(os.Stderr, "\nUpdate available: %s → %s (run `raptor update`)\n", Version, info.Version)
	}
}

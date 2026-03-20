package cmd

import (
	"fmt"
	"os"
	"raptor/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var serverURL string

var rootCmd = &cobra.Command{
	Use:   "raptor",
	Short: "A multiplayer kanban board",
	Long:  "Raptor is a CLI kanban board with real-time sync.",
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
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:8080", "server URL")
}

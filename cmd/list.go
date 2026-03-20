package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var listStatus string

var statusStyle = map[string]lipgloss.Style{
	"todo":        lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
	"in_progress": lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
	"done":        lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverURL)
		tickets, err := c.ListTickets(listStatus)
		if err != nil {
			return err
		}
		if len(tickets) == 0 {
			fmt.Println("No tickets found.")
			return nil
		}
		for _, t := range tickets {
			style := statusStyle[string(t.Status)]
			fmt.Printf("%s  %s  %s\n", t.ID, style.Render(string(t.Status)), t.Title)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "filter by status (todo, in_progress, done)")
	rootCmd.AddCommand(listCmd)
}

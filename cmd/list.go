package cmd

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	listStatus string
	listMine   bool
)

var statusStyle = map[string]lipgloss.Style{
	"todo":        lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
	"in_progress": lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
	"done":        lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverURL, authToken)
		tickets, err := c.ListTickets(listStatus, listMine)
		if err != nil {
			return err
		}
		if len(tickets) == 0 {
			fmt.Println("No tickets found.")
			return nil
		}
		for _, t := range tickets {
			style := statusStyle[string(t.Status)]
			line := fmt.Sprintf("%s  %s  %s", t.ID, style.Render(string(t.Status)), t.Title)
			if t.Assignee != "" {
				line += fmt.Sprintf("  [@%s]", t.Assignee)
			}
			if t.CreatedBy != "" {
				line += fmt.Sprintf("  (by %s)", t.CreatedBy)
			}
			fmt.Println(line)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "filter by status (todo, in_progress, done)")
	listCmd.Flags().BoolVar(&listMine, "mine", false, "show only my tickets")
	rootCmd.AddCommand(listCmd)
}

package cmd

import (
	"fmt"
	"raptor/client"
	"raptor/model"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

var (
	listStatus string
	listMine   bool
	listAll    bool
)

var statusColors = map[string]lipgloss.Color{
	"todo":        lipgloss.Color("12"),
	"in_progress": lipgloss.Color("11"),
	"done":        lipgloss.Color("10"),
	"closed":      lipgloss.Color("8"),
}

var headerStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
var cellStyle = lipgloss.NewStyle().Padding(0, 1)

func renderTicketTable(tickets []model.Ticket) string {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("238"))).
		Headers("ID", "Status", "Title", "Assignee").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			s := cellStyle
			// Color the status column
			if col == 1 && row >= 0 && row < len(tickets) {
				status := string(tickets[row].Status)
				if c, ok := statusColors[status]; ok {
					s = s.Foreground(c)
				}
			}
			return s
		})

	for _, tk := range tickets {
		t.Row(tk.ID, string(tk.Status), tk.Title, tk.Assignee)
	}

	rendered := t.Render()
	tableWidth := lipgloss.Width(rendered)

	title := lipgloss.NewStyle().Bold(true).Italic(true).
		Width(tableWidth).Align(lipgloss.Center).
		Render(fmt.Sprintf("Tickets (%d)", len(tickets)))

	return fmt.Sprintf("%s\n%s", title, rendered)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := client.NewScoped(serverURL, authToken, activeWS, activeBoard)
		tickets, err := c.ListTickets(client.ListOptions{Status: listStatus, Mine: listMine, All: listAll})
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(tickets)
			return nil
		}
		if len(tickets) == 0 {
			fmt.Println("No tickets found.")
			return nil
		}
		fmt.Println(renderTicketTable(tickets))
		return nil
	},
}

func init() {
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "filter by status (todo, in_progress, done, closed)")
	listCmd.Flags().BoolVar(&listMine, "mine", false, "show only my tickets")
	listCmd.Flags().BoolVar(&listAll, "all", false, "include closed tickets")
	rootCmd.AddCommand(listCmd)
}

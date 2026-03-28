package cmd

import (
	"fmt"
	"raptor/client"
	"raptor/model"
	"strings"

	"github.com/spf13/cobra"
)

func formatTicketTable(tickets []model.Ticket) string {
	var b strings.Builder
	b.WriteString("ID\tSTATUS\tASSIGNEE\tTITLE\n")
	for _, tk := range tickets {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", tk.ID, tk.Status, tk.Assignee, tk.Title)
	}
	return b.String()
}

var (
	listStatus string
	listMine   bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := newClient()
		tickets, err := c.ListTickets(client.ListOptions{Status: listStatus, Mine: listMine})
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
		for _, tk := range tickets {
			fmt.Printf("%-8s %-12s %-10s %s\n", tk.ID, tk.Status, tk.Assignee, tk.Title)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "filter by status")
	listCmd.Flags().BoolVar(&listMine, "mine", false, "show only my tickets")
	rootCmd.AddCommand(listCmd)
}

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search tickets by title or content",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		query := strings.Join(args, " ")
		c := newClient()
		tickets, err := c.SearchTickets(query)
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
	rootCmd.AddCommand(searchCmd)
}

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
		c := NewScopedClient(serverURL, authToken, activeWS, activeBoard)
		tickets, err := c.SearchTickets(query)
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
			fmt.Println(line)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

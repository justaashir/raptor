package cmd

import (
	"fmt"
	"raptor/client"
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
		c := client.NewScoped(serverURL, authToken, activeWS, activeBoard)
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
		fmt.Println(renderTicketTable(tickets))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}

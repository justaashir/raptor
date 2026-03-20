package cmd

import (
	"fmt"
	"raptor/client"

	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move <id> <status>",
	Short: "Move ticket to a new status",
	Long:  "Move a ticket. Valid statuses: todo, in_progress, done",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := client.NewScoped(serverURL, authToken, activeWS, activeBoard)
		ticket, err := c.UpdateTicket(args[0], map[string]any{"status": args[1]})
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(ticket)
		} else {
			fmt.Printf("Moved %s to %s\n", ticket.ID, ticket.Status)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(moveCmd)
}

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var reopenCmd = &cobra.Command{
	Use:   "reopen <id>",
	Short: "Reopen a closed ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := NewScopedClient(serverURL, authToken, activeWS, activeBoard)
		ticket, err := c.UpdateTicket(args[0], map[string]any{
			"status":       "todo",
			"close_reason": "",
			"closed_at":    nil,
		})
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(ticket)
		} else {
			fmt.Printf("Reopened %s\n", ticket.ID)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reopenCmd)
}

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var closeReason string

var closeCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Close a ticket (soft-delete)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := newClient()
		fields := map[string]any{
			"status": "closed",
		}
		if closeReason != "" {
			fields["close_reason"] = closeReason
		}
		ticket, err := c.UpdateTicket(args[0], fields)
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(ticket)
		} else {
			fmt.Printf("Closed %s\n", ticket.ID)
		}
		return nil
	},
}

func init() {
	closeCmd.Flags().StringVarP(&closeReason, "reason", "r", "", "reason for closing")
	rootCmd.AddCommand(closeCmd)
}

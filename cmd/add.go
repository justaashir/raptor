package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	addContent string
	addAssign  string
)

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a new ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverURL, authToken)
		ticket, err := c.CreateTicket(args[0], addContent, addAssign)
		if err != nil {
			return err
		}
		fmt.Printf("Created ticket %s: %s\n", ticket.ID, ticket.Title)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addContent, "content", "c", "", "ticket content (markdown)")
	addCmd.Flags().StringVarP(&addAssign, "assign", "a", "", "assign to user")
	rootCmd.AddCommand(addCmd)
}

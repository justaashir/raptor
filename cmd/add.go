package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addContent string

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a new ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverURL)
		ticket, err := c.CreateTicket(args[0], addContent)
		if err != nil {
			return err
		}
		fmt.Printf("Created ticket %s: %s\n", ticket.ID, ticket.Title)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addContent, "content", "c", "", "ticket content (markdown)")
	rootCmd.AddCommand(addCmd)
}

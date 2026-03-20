package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var editTitle string
var editContent string

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fields := map[string]any{}
		if editTitle != "" {
			fields["title"] = editTitle
		}
		if editContent != "" {
			fields["content"] = editContent
		}
		if len(fields) == 0 {
			return fmt.Errorf("specify --title or --content to edit")
		}
		c := NewClient(serverURL)
		ticket, err := c.UpdateTicket(args[0], fields)
		if err != nil {
			return err
		}
		fmt.Printf("Updated ticket %s: %s\n", ticket.ID, ticket.Title)
		return nil
	},
}

func init() {
	editCmd.Flags().StringVarP(&editTitle, "title", "t", "", "new title")
	editCmd.Flags().StringVarP(&editContent, "content", "c", "", "new content")
	rootCmd.AddCommand(editCmd)
}

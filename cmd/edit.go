package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	editTitle   string
	editContent string
	editAssign  string
)

var editCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit a ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		fields := map[string]any{}
		if cmd.Flags().Changed("title") {
			fields["title"] = editTitle
		}
		if cmd.Flags().Changed("content") {
			fields["content"] = editContent
		}
		if cmd.Flags().Changed("assign") {
			fields["assignee"] = editAssign
		}
		if len(fields) == 0 {
			return fmt.Errorf("specify --title, --content, or --assign to edit")
		}
		c := newClient()
		ticket, err := c.UpdateTicket(args[0], fields)
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(ticket)
		} else {
			fmt.Printf("Updated ticket %s: %s\n", ticket.ID, ticket.Title)
		}
		return nil
	},
}

func init() {
	editCmd.Flags().StringVarP(&editTitle, "title", "t", "", "new title")
	editCmd.Flags().StringVarP(&editContent, "content", "c", "", "new content")
	editCmd.Flags().StringVarP(&editAssign, "assign", "a", "", "assign to user")
	rootCmd.AddCommand(editCmd)
}

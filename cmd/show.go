package cmd

import (
	"fmt"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show ticket details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := newClient()
		ticket, err := c.GetTicket(args[0])
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(ticket)
			return nil
		}
		fmt.Printf("ID:       %s\n", ticket.ID)
		fmt.Printf("Title:    %s\n", ticket.Title)
		fmt.Printf("Status:   %s\n", ticket.Status)
		if ticket.CreatedBy != "" {
			fmt.Printf("Created by: %s\n", ticket.CreatedBy)
		}
		if ticket.Assignee != "" {
			fmt.Printf("Assignee: %s\n", ticket.Assignee)
		}
		if ticket.AssignedBy != "" {
			fmt.Printf("Assigned by: %s\n", ticket.AssignedBy)
		}
		fmt.Printf("Created:  %s\n", ticket.CreatedAt.Format("2006-01-02 15:04"))
		fmt.Printf("Updated:  %s\n", ticket.UpdatedAt.Format("2006-01-02 15:04"))
		if ticket.Content != "" {
			fmt.Println()
			rendered, err := glamour.Render(ticket.Content, "dark")
			if err != nil {
				fmt.Println(ticket.Content)
			} else {
				fmt.Print(rendered)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}

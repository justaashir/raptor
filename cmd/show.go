package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"raptor/model"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"
)

func renderTicketView(tk model.Ticket) (string, error) {
	md := fmt.Sprintf("# %s\n\n**ID:** %s | **Status:** %s | **Created by:** %s | **Assignee:** %s\n**Created:** %s | **Updated:** %s\n",
		tk.Title, tk.ID, tk.Status, tk.CreatedBy, tk.Assignee,
		tk.CreatedAt.Format("2006-01-02 15:04"),
		tk.UpdatedAt.Format("2006-01-02 15:04"),
	)
	if tk.Content != "" {
		md += "\n---\n\n" + tk.Content + "\n"
	}
	return glamour.Render(md, "dark")
}

func showWithPager(content string) error {
	if _, err := exec.LookPath("less"); err != nil {
		fmt.Print(content)
		return nil
	}
	cmd := exec.Command("less", "-R")
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

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
		rendered, err := renderTicketView(ticket)
		if err != nil {
			return err
		}
		return showWithPager(rendered)
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}

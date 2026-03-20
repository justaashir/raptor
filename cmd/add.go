package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func renderCreateBox(id, title string) string {
	borderColor := lipgloss.Color("238")
	border := lipgloss.RoundedBorder()

	check := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓")
	idStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true).Render("Created " + id)
	titleStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(fmt.Sprintf(`  "%s"`, title))
	body := fmt.Sprintf("%s %s\n%s", check, idStyled, titleStyled)

	// Render box without top border
	box := lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		BorderTop(false).
		Padding(0, 1).
		Render(body)

	boxWidth := lipgloss.Width(box)

	// Build top border with centered title
	label := " Create "
	bc := lipgloss.NewStyle().Foreground(borderColor)
	remaining := boxWidth - 2 - len(label) // -2 for corners
	left := remaining / 2
	right := remaining - left
	topBorder := bc.Render(border.TopLeft) +
		bc.Render(strings.Repeat(border.Top, left)) +
		label +
		bc.Render(strings.Repeat(border.Top, right)) +
		bc.Render(border.TopRight)

	return topBorder + "\n" + box
}

var (
	addContent string
	addAssign  string
)

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a new ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := newClient()
		ticket, err := c.CreateTicket(args[0], addContent, addAssign)
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(ticket)
		} else {
			fmt.Println(renderCreateBox(ticket.ID, ticket.Title))
		}
		return nil
	},
}

func init() {
	addCmd.Flags().StringVarP(&addContent, "content", "c", "", "ticket content (markdown)")
	addCmd.Flags().StringVarP(&addAssign, "assign", "a", "", "assign to user")
	rootCmd.AddCommand(addCmd)
}

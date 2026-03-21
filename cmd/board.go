package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Manage boards",
	Long:  "Create, list, and manage boards within the active workspace.",
}

var bdCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new board",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireWorkspace(); err != nil {
			return err
		}
		var statuses []string
		if cmd.Flags().Changed("statuses") {
			s, _ := cmd.Flags().GetString("statuses")
			statuses = parseStatuses(s)
		}
		c := newUnscopedClient()
		bd, err := c.CreateBoard(activeWS, args[0], statuses)
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(bd)
		} else {
			fmt.Printf("Created board %s: %s\n", bd.ID, bd.Name)
			if cmd.Flags().Changed("statuses") {
				fmt.Printf("Statuses: %s\n", bd.Statuses)
			}
		}
		return nil
	},
}

var bdListCmd = &cobra.Command{
	Use:   "list",
	Short: "List boards in the active workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireWorkspace(); err != nil {
			return err
		}
		c := newUnscopedClient()
		boards, err := c.ListBoards(activeWS)
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(boards)
			return nil
		}
		if len(boards) == 0 {
			fmt.Println("No boards found.")
			return nil
		}
		for _, bd := range boards {
			marker := ""
			if bd.ID == activeBoard {
				marker = " (active)"
			}
			fmt.Printf("%s  %s  [%s]%s\n", bd.ID, bd.Name, bd.Statuses, marker)
		}
		return nil
	},
}

var bdUseCmd = &cobra.Command{
	Use:   "use <name-or-id>",
	Short: "Set active board",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireWorkspace(); err != nil {
			return err
		}
		c := newUnscopedClient()
		boards, err := c.ListBoards(activeWS)
		if err != nil {
			return err
		}
		search := strings.ToLower(args[0])
		for _, bd := range boards {
			if bd.ID == args[0] || strings.ToLower(bd.Name) == search {
				cfg, _ := LoadConfig()
				cfg.Board = bd.ID
				if err := SaveConfig(cfg); err != nil {
					return err
				}
				fmt.Printf("Active board: %s (%s)\n", bd.Name, bd.ID)
				return nil
			}
		}
		return fmt.Errorf("board %q not found", args[0])
	},
}

var bdEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the active board (name, statuses)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		fields := map[string]any{}
		if cmd.Flags().Changed("name") {
			name, _ := cmd.Flags().GetString("name")
			fields["name"] = name
		}
		if cmd.Flags().Changed("statuses") {
			s, _ := cmd.Flags().GetString("statuses")
			fields["statuses"] = parseStatuses(s)
		}
		if len(fields) == 0 {
			return fmt.Errorf("specify --name and/or --statuses")
		}
		c := newUnscopedClient()
		bd, err := c.UpdateBoard(activeWS, activeBoard, fields)
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(bd)
		} else {
			fmt.Printf("Updated board %s: %s [%s]\n", bd.ID, bd.Name, bd.Statuses)
		}
		return nil
	},
}

func parseStatuses(raw string) []string {
	parts := strings.Split(raw, ",")
	var out []string
	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func init() {
	bdCreateCmd.Flags().String("statuses", "", "comma-separated statuses (default: todo,in_progress,done)")
	bdEditCmd.Flags().String("name", "", "new board name")
	bdEditCmd.Flags().String("statuses", "", "comma-separated statuses")
	boardCmd.AddCommand(bdCreateCmd, bdListCmd, bdUseCmd, bdEditCmd)
	rootCmd.AddCommand(boardCmd)
}

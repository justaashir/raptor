package cmd

import (
	"fmt"
	"raptor/client"
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
		if activeWS == "" {
			return fmt.Errorf("no workspace selected. Run 'raptor workspace use' first")
		}
		c := client.New(serverURL, authToken)
		bd, err := c.CreateBoard(activeWS, args[0])
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(bd)
		} else {
			fmt.Printf("Created board %s: %s\n", bd.ID, bd.Name)
		}
		return nil
	},
}

var bdListCmd = &cobra.Command{
	Use:   "list",
	Short: "List boards in the active workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		if activeWS == "" {
			return fmt.Errorf("no workspace selected. Run 'raptor workspace use' first")
		}
		c := client.New(serverURL, authToken)
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
			fmt.Printf("%s  %s%s\n", bd.ID, bd.Name, marker)
		}
		return nil
	},
}

var bdUseCmd = &cobra.Command{
	Use:   "use <name-or-id>",
	Short: "Set active board",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if activeWS == "" {
			return fmt.Errorf("no workspace selected. Run 'raptor workspace use' first")
		}
		c := client.New(serverURL, authToken)
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

var bdMembersCmd = &cobra.Command{
	Use:   "members",
	Short: "List board members",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := client.New(serverURL, authToken)
		members, err := c.ListBoardMembers(activeWS, activeBoard)
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(members)
			return nil
		}
		if len(members) == 0 {
			fmt.Println("No explicit board members. Owners and admins have implicit access.")
			return nil
		}
		for _, m := range members {
			fmt.Printf("%s\n", m.Username)
		}
		return nil
	},
}

var bdGrantCmd = &cobra.Command{
	Use:   "grant <username>",
	Short: "Grant a user access to this board",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := client.New(serverURL, authToken)
		if err := c.GrantBoardAccess(activeWS, activeBoard, args[0]); err != nil {
			return err
		}
		fmt.Printf("Granted %s access to board\n", args[0])
		return nil
	},
}

var bdRevokeCmd = &cobra.Command{
	Use:   "revoke <username>",
	Short: "Revoke a user's access to this board",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := client.New(serverURL, authToken)
		if err := c.RevokeBoardAccess(activeWS, activeBoard, args[0]); err != nil {
			return err
		}
		fmt.Printf("Revoked %s access from board\n", args[0])
		return nil
	},
}

func init() {
	boardCmd.AddCommand(bdCreateCmd, bdListCmd, bdUseCmd, bdMembersCmd, bdGrantCmd, bdRevokeCmd)
	rootCmd.AddCommand(boardCmd)
}

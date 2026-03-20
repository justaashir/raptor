package cmd

import (
	"fmt"
	"raptor/client"
	"strings"

	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage workspaces",
	Long:  "Create, list, and manage workspaces and their members.",
}

var wsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New(serverURL, authToken)
		ws, err := c.CreateWorkspace(args[0])
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(ws)
		} else {
			fmt.Printf("Created workspace %s: %s\n", ws.ID, ws.Name)
		}
		return nil
	},
}

var wsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your workspaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New(serverURL, authToken)
		workspaces, err := c.ListWorkspaces()
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(workspaces)
			return nil
		}
		if len(workspaces) == 0 {
			fmt.Println("No workspaces found.")
			return nil
		}
		for _, ws := range workspaces {
			marker := ""
			if ws.ID == activeWS {
				marker = " (active)"
			}
			fmt.Printf("%s  %s%s\n", ws.ID, ws.Name, marker)
		}
		return nil
	},
}

var wsUseCmd = &cobra.Command{
	Use:   "use <name-or-id>",
	Short: "Set active workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New(serverURL, authToken)
		workspaces, err := c.ListWorkspaces()
		if err != nil {
			return err
		}
		search := strings.ToLower(args[0])
		for _, ws := range workspaces {
			if ws.ID == args[0] || strings.ToLower(ws.Name) == search {
				cfg, _ := LoadConfig()
				cfg.Workspace = ws.ID
				if err := SaveConfig(cfg); err != nil {
					return err
				}
				fmt.Printf("Active workspace: %s (%s)\n", ws.Name, ws.ID)
				return nil
			}
		}
		return fmt.Errorf("workspace %q not found", args[0])
	},
}

var wsMembersCmd = &cobra.Command{
	Use:   "members",
	Short: "List workspace members",
	RunE: func(cmd *cobra.Command, args []string) error {
		if activeWS == "" {
			return fmt.Errorf("no workspace selected. Run 'raptor workspace use' first")
		}
		c := client.New(serverURL, authToken)
		members, err := c.ListWorkspaceMembers(activeWS)
		if err != nil {
			return err
		}
		if jsonOutput {
			printJSON(members)
			return nil
		}
		for _, m := range members {
			fmt.Printf("%s  %s\n", m.Username, m.Role)
		}
		return nil
	},
}

var inviteRole string

var wsInviteCmd = &cobra.Command{
	Use:   "invite <username>",
	Short: "Invite a user to the workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if activeWS == "" {
			return fmt.Errorf("no workspace selected. Run 'raptor workspace use' first")
		}
		c := client.New(serverURL, authToken)
		if err := c.InviteWorkspaceMember(activeWS, args[0], inviteRole); err != nil {
			return err
		}
		fmt.Printf("Invited %s as %s\n", args[0], inviteRole)
		return nil
	},
}

var wsKickCmd = &cobra.Command{
	Use:   "kick <username>",
	Short: "Remove a user from the workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if activeWS == "" {
			return fmt.Errorf("no workspace selected. Run 'raptor workspace use' first")
		}
		c := client.New(serverURL, authToken)
		if err := c.KickWorkspaceMember(activeWS, args[0]); err != nil {
			return err
		}
		fmt.Printf("Removed %s from workspace\n", args[0])
		return nil
	},
}

var wsRoleCmd = &cobra.Command{
	Use:   "role <username> <role>",
	Short: "Change a member's role (owner only)",
	Long:  "Change role. Valid roles: owner, admin, member",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if activeWS == "" {
			return fmt.Errorf("no workspace selected. Run 'raptor workspace use' first")
		}
		c := client.New(serverURL, authToken)
		if err := c.ChangeRole(activeWS, args[0], args[1]); err != nil {
			return err
		}
		fmt.Printf("Changed %s role to %s\n", args[0], args[1])
		return nil
	},
}

func init() {
	wsInviteCmd.Flags().StringVar(&inviteRole, "role", "member", "role for invited user (member, admin)")
	workspaceCmd.AddCommand(wsCreateCmd, wsListCmd, wsUseCmd, wsMembersCmd, wsInviteCmd, wsKickCmd, wsRoleCmd)
	rootCmd.AddCommand(workspaceCmd)
}

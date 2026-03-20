package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rmForce bool

var rmCmd = &cobra.Command{
	Use:   "rm <id>",
	Short: "Delete a ticket",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		if !rmForce {
			fmt.Printf("Delete ticket %s? Use --force to confirm.\n", args[0])
			return nil
		}
		c := NewScopedClient(serverURL, authToken, activeWS, activeBoard)
		if err := c.DeleteTicket(args[0]); err != nil {
			return err
		}
		fmt.Printf("Deleted ticket %s\n", args[0])
		return nil
	},
}

func init() {
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "skip confirmation")
	rootCmd.AddCommand(rmCmd)
}

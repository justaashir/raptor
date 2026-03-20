package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the raptor version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("raptor %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

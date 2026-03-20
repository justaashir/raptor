package cmd

import (
	"fmt"
	"raptor/client"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show ticket counts by status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := client.NewScoped(serverURL, authToken, activeWS, activeBoard)
		result, err := c.TicketStats()
		if err != nil {
			return err
		}

		if jsonOutput {
			printJSON(result)
			return nil
		}

		total := int(result["total"].(float64))
		counts, _ := result["counts"].(map[string]any)
		fmt.Printf("Total:       %d\n", total)
		fmt.Printf("Todo:        %0.f\n", counts["todo"])
		fmt.Printf("In Progress: %0.f\n", counts["in_progress"])
		fmt.Printf("Done:        %0.f\n", counts["done"])
		fmt.Printf("Closed:      %0.f\n", counts["closed"])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

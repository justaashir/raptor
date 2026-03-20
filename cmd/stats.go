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
		tickets, err := c.ListTickets("", false, true) // all=true to include closed
		if err != nil {
			return err
		}

		counts := map[string]int{
			"todo":        0,
			"in_progress": 0,
			"done":        0,
			"closed":      0,
		}
		for _, t := range tickets {
			counts[string(t.Status)]++
		}
		total := len(tickets)

		if jsonOutput {
			result := map[string]any{
				"total":  total,
				"counts": counts,
			}
			printJSON(result)
			return nil
		}

		fmt.Printf("Total:       %d\n", total)
		fmt.Printf("Todo:        %d\n", counts["todo"])
		fmt.Printf("In Progress: %d\n", counts["in_progress"])
		fmt.Printf("Done:        %d\n", counts["done"])
		fmt.Printf("Closed:      %d\n", counts["closed"])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

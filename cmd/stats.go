package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show ticket counts by status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoard(); err != nil {
			return err
		}
		c := newClient()

		// Try server-side stats first, fall back to client-side counting
		result, err := c.TicketStats()
		if err != nil {
			return err
		}

		var total int
		counts := map[string]int{
			"todo": 0, "in_progress": 0, "done": 0, "closed": 0,
		}

		rawCounts, ok := result["counts"].(map[string]any)
		if !ok {
			return fmt.Errorf("unexpected stats response format")
		}
		for k, v := range rawCounts {
			if n, ok := v.(float64); ok {
				counts[k] = int(n)
			}
		}
		if t, ok := result["total"].(float64); ok {
			total = int(t)
		}

		if jsonOutput {
			printJSON(map[string]any{"total": total, "counts": counts})
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

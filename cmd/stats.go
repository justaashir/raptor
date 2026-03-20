package cmd

import (
	"fmt"
	"sort"

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

		result, err := c.TicketStats()
		if err != nil {
			return err
		}

		var total int
		counts := map[string]int{}

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

		fmt.Printf("Total: %d\n", total)
		// Sort keys for consistent output
		var keys []string
		for k := range counts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %-15s %d\n", k, counts[k])
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

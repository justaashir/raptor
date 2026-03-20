package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type healthCheck struct {
	Check  string `json:"check"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check connectivity and configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		var checks []healthCheck
		allOK := true

		// Check server connectivity
		if v := fetchServerVersion(); v != "" {
			detail := fmt.Sprintf("reachable (v%s)", v)
			if v != Version && Version != "dev" {
				detail += " — client version mismatch"
			}
			checks = append(checks, healthCheck{"server", "OK", detail})
		} else {
			checks = append(checks, healthCheck{"server", "FAIL", "unreachable"})
			allOK = false
		}

		// Check auth
		if authToken != "" {
			checks = append(checks, healthCheck{"auth", "OK", fmt.Sprintf("logged in as %s", cfgUsername)})
		} else {
			checks = append(checks, healthCheck{"auth", "WARN", "not logged in"})
		}

		// Check workspace
		if activeWS != "" {
			checks = append(checks, healthCheck{"workspace", "OK", activeWS})
		} else {
			checks = append(checks, healthCheck{"workspace", "WARN", "no workspace selected"})
		}

		// Check board
		if activeBoard != "" {
			checks = append(checks, healthCheck{"board", "OK", activeBoard})
		} else {
			checks = append(checks, healthCheck{"board", "WARN", "no board selected"})
		}

		if jsonOutput {
			printJSON(checks)
			return nil
		}

		for _, c := range checks {
			icon := "✓"
			if c.Status == "FAIL" {
				icon = "✗"
			} else if c.Status == "WARN" {
				icon = "!"
			}
			fmt.Printf("%s %s: %s\n", icon, c.Check, c.Detail)
		}
		if !allOK {
			return fmt.Errorf("some checks failed")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

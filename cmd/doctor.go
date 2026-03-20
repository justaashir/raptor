package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check connectivity and configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		checks := []map[string]string{}
		allOK := true

		// Check server connectivity
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(serverURL + "/api/version")
		if err != nil {
			checks = append(checks, map[string]string{
				"check": "server", "status": "FAIL", "detail": err.Error(),
			})
			allOK = false
		} else {
			var info struct{ Version string `json:"version"` }
			json.NewDecoder(resp.Body).Decode(&info)
			resp.Body.Close()
			detail := "reachable"
			if info.Version != "" {
				detail = fmt.Sprintf("reachable (v%s)", info.Version)
				if info.Version != Version && Version != "dev" {
					detail += " — client version mismatch"
				}
			}
			checks = append(checks, map[string]string{
				"check": "server", "status": "OK", "detail": detail,
			})
		}

		// Check auth
		if authToken != "" {
			checks = append(checks, map[string]string{
				"check": "auth", "status": "OK", "detail": fmt.Sprintf("logged in as %s", cfgUsername),
			})
		} else {
			checks = append(checks, map[string]string{
				"check": "auth", "status": "WARN", "detail": "not logged in",
			})
		}

		// Check workspace
		if activeWS != "" {
			checks = append(checks, map[string]string{
				"check": "workspace", "status": "OK", "detail": activeWS,
			})
		} else {
			checks = append(checks, map[string]string{
				"check": "workspace", "status": "WARN", "detail": "no workspace selected",
			})
		}

		// Check board
		if activeBoard != "" {
			checks = append(checks, map[string]string{
				"check": "board", "status": "OK", "detail": activeBoard,
			})
		} else {
			checks = append(checks, map[string]string{
				"check": "board", "status": "WARN", "detail": "no board selected",
			})
		}

		if jsonOutput {
			printJSON(checks)
			return nil
		}

		for _, c := range checks {
			icon := "✓"
			if c["status"] == "FAIL" {
				icon = "✗"
			} else if c["status"] == "WARN" {
				icon = "!"
			}
			fmt.Printf("%s %s: %s\n", icon, c["check"], c["detail"])
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

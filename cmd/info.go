package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, _ := LoadConfig()

		info := map[string]string{
			"server":    serverURL,
			"username":  cfgUsername,
			"workspace": activeWS,
			"board":     activeBoard,
			"version":   Version,
		}

		if jsonOutput {
			printJSON(info)
			return nil
		}

		fmt.Printf("Server:    %s\n", serverURL)
		fmt.Printf("Username:  %s\n", orDefault(cfgUsername, "(not logged in)"))
		fmt.Printf("Workspace: %s\n", orDefault(activeWS, "(none)"))
		fmt.Printf("Board:     %s\n", orDefault(activeBoard, "(none)"))
		fmt.Printf("Version:   %s\n", Version)
		if cfg.Server != "" && cfg.Server != serverURL {
			fmt.Printf("Config server: %s\n", cfg.Server)
		}
		return nil
	},
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

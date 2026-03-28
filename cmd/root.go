package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"raptor/client"
	"time"

	"github.com/spf13/cobra"
)

// Set via -ldflags "-X raptor/cmd.Version=... -X raptor/cmd.DefaultServer=..."
var (
	Version       = "dev"
	DefaultServer = "http://localhost:8080"
)

var (
	serverURL     string
	authToken     string
	activeWS      string
	activeBoard   string
	cfgUsername   string
	jsonOutput    bool
)

var rootCmd = &cobra.Command{
	Use:   "raptor",
	Short: "A multiplayer kanban board",
	Long:  "Raptor is a CLI kanban board with real-time sync.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		name := cmd.Name()
		// Skip auth + update check for these commands
		if name == "serve" || name == "version" || name == "update" || name == "login" || name == "completion" {
			return
		}

		// Load config for token and server URL
		cfg, err := LoadConfig()
		if err == nil {
			if cfg.Token != "" {
				authToken = cfg.Token
			}
			if cfg.Username != "" {
				cfgUsername = cfg.Username
			}
			// Use config server if user didn't override via flag
			if cfg.Server != "" && !cmd.Flags().Changed("server") {
				serverURL = cfg.Server
			}
			// Load workspace/board from config unless overridden by flags
			if cfg.Workspace != "" && !cmd.Flags().Changed("workspace") {
				activeWS = cfg.Workspace
			}
			if cfg.Board != "" && !cmd.Flags().Changed("board") {
				activeBoard = cfg.Board
			}
		}

		if authToken == "" && name != "raptor" {
			fmt.Fprintln(os.Stderr, "Warning: not logged in. Run `raptor login` to authenticate.")
		}

		go checkForUpdate()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", DefaultServer, "server URL")
	rootCmd.PersistentFlags().StringVar(&activeWS, "workspace", "", "workspace ID")
	rootCmd.PersistentFlags().StringVar(&activeBoard, "board", "", "board ID")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")
}

// requireWorkspace returns an error if workspace is not set.
func requireWorkspace() error {
	if activeWS == "" {
		return fmt.Errorf("no workspace selected. Run 'raptor workspace use' first")
	}
	return nil
}

// requireBoard returns an error if workspace or board is not set.
func requireBoard() error {
	if activeWS == "" || activeBoard == "" {
		return fmt.Errorf("no board selected. Run 'raptor workspace use' and 'raptor board use' first")
	}
	return nil
}

// newClient returns a scoped API client using the current config.
func newClient() *client.Client {
	return client.NewScoped(serverURL, authToken, activeWS, activeBoard)
}

// newUnscopedClient returns an API client without workspace/board scope.
func newUnscopedClient() *client.Client {
	return client.New(serverURL, authToken)
}

// fetchServerVersion returns the server's version string, or "" on error.
func fetchServerVersion() string {
	resp, err := httpClient.Get(serverURL + "/api/version")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var info struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return ""
	}
	return info.Version
}

const updateCheckInterval = 24 * time.Hour

func shouldCheckUpdate(cfg Config) bool {
	if cfg.LastUpdateCheck == 0 {
		return true
	}
	return time.Since(time.Unix(cfg.LastUpdateCheck, 0)) > updateCheckInterval
}

func checkForUpdate() {
	if Version == "dev" {
		return
	}
	cfg, _ := LoadConfig()
	if !shouldCheckUpdate(cfg) {
		// Use cached version if available
		if cfg.LatestVersion != "" && cfg.LatestVersion != Version {
			fmt.Fprintf(os.Stderr, "\nUpdate available: %s → %s (run `raptor update`)\n", Version, cfg.LatestVersion)
		}
		return
	}
	v := fetchServerVersion()
	if v != "" {
		cfg.LastUpdateCheck = time.Now().Unix()
		cfg.LatestVersion = v
		_ = SaveConfig(cfg)
		if v != Version {
			fmt.Fprintf(os.Stderr, "\nUpdate available: %s → %s (run `raptor update`)\n", Version, v)
		}
	}
}

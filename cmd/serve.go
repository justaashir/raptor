package cmd

import (
	"fmt"
	"os"
	"raptor/server"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the raptor server",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := "raptor.db"
		if v := os.Getenv("DATABASE_PATH"); v != "" {
			dbPath = v
		}

		if v := os.Getenv("PORT"); v != "" {
			if p, err := strconv.Atoi(v); err == nil {
				servePort = p
			}
		}

		// Parse seed users from RAPTOR_USERS env var
		var seedUsers []string
		if users := os.Getenv("RAPTOR_USERS"); users != "" {
			for _, u := range strings.Split(users, ",") {
				seedUsers = append(seedUsers, strings.TrimSpace(u))
			}
		}

		db, err := server.NewDB(dbPath, seedUsers...)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		if v := os.Getenv("VERSION"); v != "" {
			Version = v
		}
		server.CurrentVersion = Version

		hub := server.NewHub()
		go hub.Run()

		var opts []server.Option
		if secret := os.Getenv("RAPTOR_SECRET"); secret != "" {
			opts = append(opts, server.WithSecret(secret))
			fmt.Println("Auth enabled")
		}
		if len(seedUsers) > 0 {
			opts = append(opts, server.WithAllowedUsers(seedUsers))
			fmt.Printf("Allowed users: %s\n", strings.Join(seedUsers, ", "))
		}

		srv := server.NewServer(db, hub, opts...)
		addr := fmt.Sprintf(":%d", servePort)
		fmt.Printf("Raptor server listening on %s\n", addr)
		return srv.Echo.Start(addr)
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")
	rootCmd.AddCommand(serveCmd)
}

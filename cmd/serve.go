package cmd

import (
	"fmt"
	"net/http"
	"os"
	"raptor/server"
	"strconv"

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

		db, err := server.NewDB(dbPath)
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

		srv := server.NewServer(db, hub)
		addr := fmt.Sprintf(":%d", servePort)
		fmt.Printf("Raptor server listening on %s\n", addr)
		return http.ListenAndServe(addr, srv)
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")
	rootCmd.AddCommand(serveCmd)
}

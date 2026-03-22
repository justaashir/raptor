package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"raptor/server"
	"strconv"
	"strings"
	"syscall"
	"time"

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

		var opts []server.Option
		if secret := os.Getenv("RAPTOR_SECRET"); secret != "" {
			opts = append(opts, server.WithSecret(secret))
			fmt.Println("Auth enabled")
		}
		if users := os.Getenv("RAPTOR_USERS"); users != "" {
			var allowedUsers []string
			for _, u := range strings.Split(users, ",") {
				allowedUsers = append(allowedUsers, strings.TrimSpace(u))
			}
			opts = append(opts, server.WithAllowedUsers(allowedUsers))
			fmt.Printf("Allowed users: %s\n", strings.Join(allowedUsers, ", "))
		}

		srv := server.NewServer(db, hub, opts...)
		addr := fmt.Sprintf(":%d", servePort)

		// Log DB path and file info for debugging volume persistence
		fmt.Printf("Database path: %s\n", dbPath)
		if info, err := os.Stat(dbPath); err == nil {
			fmt.Printf("Database file size: %d bytes\n", info.Size())
		} else {
			fmt.Printf("Database file: new (will be created)\n")
		}

		// Graceful shutdown: handle SIGTERM/SIGINT to checkpoint WAL before exit
		go func() {
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
			sig := <-quit
			fmt.Printf("\nReceived %s, shutting down gracefully...\n", sig)

			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()

			if err := srv.Echo.Shutdown(ctx); err != nil {
				fmt.Printf("Echo shutdown error: %v\n", err)
			}

			fmt.Println("Checkpointing WAL...")
			if err := db.Checkpoint(); err != nil {
				fmt.Printf("WAL checkpoint error: %v\n", err)
			}
			fmt.Println("Shutdown complete")
		}()

		fmt.Printf("Raptor server listening on %s\n", addr)
		return srv.Echo.Start(addr)
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "port to listen on")
	rootCmd.AddCommand(serveCmd)
}

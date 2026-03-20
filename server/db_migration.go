package server

import (
	"database/sql"

	"github.com/google/uuid"
)

func migrate(conn *sql.DB, seedUsers []string) error {
	// Create new tables for workspaces/boards
	_, err := conn.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`)
	if err != nil {
		return err
	}
	_, err = conn.Exec(`CREATE TABLE IF NOT EXISTS workspaces (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_by TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}
	_, err = conn.Exec(`CREATE TABLE IF NOT EXISTS workspace_members (
		workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
		username     TEXT NOT NULL,
		role         TEXT NOT NULL CHECK(role IN ('owner','admin','member')),
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (workspace_id, username)
	)`)
	if err != nil {
		return err
	}
	_, err = conn.Exec(`CREATE TABLE IF NOT EXISTS boards (
		id           TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
		name         TEXT NOT NULL,
		created_by   TEXT NOT NULL,
		created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}
	_, err = conn.Exec(`CREATE TABLE IF NOT EXISTS board_members (
		board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
		username TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (board_id, username)
	)`)
	if err != nil {
		return err
	}

	// Add board_id to tickets if missing
	conn.Exec(`ALTER TABLE tickets ADD COLUMN board_id TEXT DEFAULT ''`)

	// Check if we already migrated
	var version int
	err = conn.QueryRow(`SELECT version FROM schema_version LIMIT 1`).Scan(&version)
	if err == nil {
		return nil // already at version 1+
	}

	// Seed default workspace/board if seed users provided
	if len(seedUsers) > 0 {
		wsID := uuid.New().String()[:8]
		bdID := uuid.New().String()[:8]
		owner := seedUsers[0]

		conn.Exec(`INSERT INTO workspaces (id, name, created_by) VALUES (?, 'Default', ?)`, wsID, owner)
		conn.Exec(`INSERT INTO workspace_members (workspace_id, username, role) VALUES (?, ?, 'owner')`, wsID, owner)

		for _, u := range seedUsers[1:] {
			conn.Exec(`INSERT INTO workspace_members (workspace_id, username, role) VALUES (?, ?, 'admin')`, wsID, u)
		}

		conn.Exec(`INSERT INTO boards (id, workspace_id, name, created_by) VALUES (?, ?, 'Default', ?)`, bdID, wsID, owner)

		// Migrate existing tickets to default board
		conn.Exec(`UPDATE tickets SET board_id = ? WHERE board_id = ''`, bdID)
	}

	conn.Exec(`INSERT INTO schema_version (version) VALUES (1)`)
	return nil
}

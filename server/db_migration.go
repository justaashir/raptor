package server

import "database/sql"

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

	return nil
}

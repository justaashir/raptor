package server

import (
	"database/sql"
	"fmt"
	"raptor/model"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func NewDB(dsn string, seedUsers ...string) (*DB, error) {
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	_, err = conn.Exec(`CREATE TABLE IF NOT EXISTS tickets (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		content TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT 'todo',
		created_by TEXT DEFAULT '',
		assignee TEXT DEFAULT '',
		assigned_by TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		conn.Close()
		return nil, err
	}
	// Migrate existing tables
	conn.Exec(`ALTER TABLE tickets ADD COLUMN created_by TEXT DEFAULT ''`)
	conn.Exec(`ALTER TABLE tickets ADD COLUMN assignee TEXT DEFAULT ''`)
	conn.Exec(`ALTER TABLE tickets ADD COLUMN assigned_by TEXT DEFAULT ''`)

	if err := migrate(conn, seedUsers); err != nil {
		conn.Close()
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// Workspace methods

func (db *DB) CreateWorkspace(id, name, createdBy string) error {
	_, err := db.conn.Exec(
		`INSERT INTO workspaces (id, name, created_by) VALUES (?, ?, ?)`,
		id, name, createdBy,
	)
	if err != nil {
		return err
	}
	// Creator becomes owner
	_, err = db.conn.Exec(
		`INSERT INTO workspace_members (workspace_id, username, role) VALUES (?, ?, 'owner')`,
		id, createdBy,
	)
	return err
}

func (db *DB) AddWorkspaceMember(workspaceID, username, role string) error {
	_, err := db.conn.Exec(
		`INSERT INTO workspace_members (workspace_id, username, role) VALUES (?, ?, ?)`,
		workspaceID, username, role,
	)
	return err
}

func (db *DB) ListWorkspaceMembers(workspaceID string) ([]model.WorkspaceMember, error) {
	rows, err := db.conn.Query(
		`SELECT workspace_id, username, role, created_at FROM workspace_members WHERE workspace_id = ? ORDER BY created_at`,
		workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []model.WorkspaceMember
	for rows.Next() {
		var m model.WorkspaceMember
		if err := rows.Scan(&m.WorkspaceID, &m.Username, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (db *DB) GetMemberRole(workspaceID, username string) (string, error) {
	var role string
	err := db.conn.QueryRow(
		`SELECT role FROM workspace_members WHERE workspace_id = ? AND username = ?`,
		workspaceID, username,
	).Scan(&role)
	return role, err
}

func (db *DB) UpdateMemberRole(workspaceID, username, role string) error {
	_, err := db.conn.Exec(
		`UPDATE workspace_members SET role = ? WHERE workspace_id = ? AND username = ?`,
		role, workspaceID, username,
	)
	return err
}

func (db *DB) RemoveWorkspaceMember(workspaceID, username string) error {
	_, err := db.conn.Exec(
		`DELETE FROM workspace_members WHERE workspace_id = ? AND username = ?`,
		workspaceID, username,
	)
	return err
}

func (db *DB) DeleteWorkspace(id string) error {
	_, err := db.conn.Exec(`DELETE FROM workspaces WHERE id = ?`, id)
	return err
}

// IsWorkspaceMember checks if a user belongs to any workspace (for auth)
func (db *DB) IsWorkspaceMember(username string) (bool, error) {
	var count int
	err := db.conn.QueryRow(
		`SELECT COUNT(*) FROM workspace_members WHERE username = ?`, username,
	).Scan(&count)
	return count > 0, err
}

func (db *DB) ListWorkspacesForUser(username string) ([]model.Workspace, error) {
	rows, err := db.conn.Query(
		`SELECT w.id, w.name, w.created_by, w.created_at FROM workspaces w
		 JOIN workspace_members wm ON w.id = wm.workspace_id
		 WHERE wm.username = ?
		 ORDER BY w.created_at`, username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var workspaces []model.Workspace
	for rows.Next() {
		var w model.Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.CreatedBy, &w.CreatedAt); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, rows.Err()
}

// Board methods

func (db *DB) CreateBoard(id, workspaceID, name, createdBy string) error {
	_, err := db.conn.Exec(
		`INSERT INTO boards (id, workspace_id, name, created_by) VALUES (?, ?, ?, ?)`,
		id, workspaceID, name, createdBy,
	)
	return err
}

func (db *DB) ListBoardsForUser(workspaceID, username string) ([]model.Board, error) {
	// Owners/admins see all boards in workspace; members only see granted boards
	role, err := db.GetMemberRole(workspaceID, username)
	if err != nil {
		return nil, nil // not a member
	}

	var rows *sql.Rows
	if role == "owner" || role == "admin" {
		rows, err = db.conn.Query(
			`SELECT id, workspace_id, name, created_by, created_at FROM boards WHERE workspace_id = ? ORDER BY created_at`,
			workspaceID,
		)
	} else {
		rows, err = db.conn.Query(
			`SELECT b.id, b.workspace_id, b.name, b.created_by, b.created_at FROM boards b
			 JOIN board_members bm ON b.id = bm.board_id
			 WHERE b.workspace_id = ? AND bm.username = ?
			 ORDER BY b.created_at`,
			workspaceID, username,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var boards []model.Board
	for rows.Next() {
		var b model.Board
		if err := rows.Scan(&b.ID, &b.WorkspaceID, &b.Name, &b.CreatedBy, &b.CreatedAt); err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	return boards, rows.Err()
}

func (db *DB) DeleteBoard(id string) error {
	_, err := db.conn.Exec(`DELETE FROM boards WHERE id = ?`, id)
	return err
}

func (db *DB) GetBoard(id string) (model.Board, error) {
	var b model.Board
	err := db.conn.QueryRow(
		`SELECT id, workspace_id, name, created_by, created_at FROM boards WHERE id = ?`, id,
	).Scan(&b.ID, &b.WorkspaceID, &b.Name, &b.CreatedBy, &b.CreatedAt)
	return b, err
}

// Board member methods

func (db *DB) AddBoardMember(boardID, username string) error {
	_, err := db.conn.Exec(
		`INSERT INTO board_members (board_id, username) VALUES (?, ?)`,
		boardID, username,
	)
	return err
}

func (db *DB) ListBoardMembers(boardID string) ([]model.BoardMember, error) {
	rows, err := db.conn.Query(
		`SELECT board_id, username, created_at FROM board_members WHERE board_id = ? ORDER BY created_at`,
		boardID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []model.BoardMember
	for rows.Next() {
		var m model.BoardMember
		if err := rows.Scan(&m.BoardID, &m.Username, &m.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (db *DB) IsBoardMember(boardID, username string) (bool, error) {
	// Check explicit board_members grant
	var count int
	err := db.conn.QueryRow(
		`SELECT COUNT(*) FROM board_members WHERE board_id = ? AND username = ?`,
		boardID, username,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	// Owners/admins have implicit access to all boards in their workspace
	var role string
	err = db.conn.QueryRow(
		`SELECT wm.role FROM workspace_members wm
		 JOIN boards b ON b.workspace_id = wm.workspace_id
		 WHERE b.id = ? AND wm.username = ?`,
		boardID, username,
	).Scan(&role)
	if err != nil {
		return false, nil
	}
	return role == "owner" || role == "admin", nil
}

func (db *DB) RemoveBoardMember(boardID, username string) error {
	_, err := db.conn.Exec(
		`DELETE FROM board_members WHERE board_id = ? AND username = ?`,
		boardID, username,
	)
	return err
}

// Ticket methods

func (db *DB) CreateTicket(t model.Ticket) error {
	_, err := db.conn.Exec(
		`INSERT INTO tickets (id, title, content, status, board_id, created_by, assignee, assigned_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Content, t.Status, t.BoardID, t.CreatedBy, t.Assignee, t.AssignedBy, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (db *DB) ListTickets(boardID, status string) ([]model.Ticket, error) {
	query := `SELECT id, title, content, status, board_id, created_by, assignee, assigned_by, created_at, updated_at FROM tickets`
	var conditions []string
	var args []any
	if boardID != "" {
		conditions = append(conditions, "board_id = ?")
		args = append(args, boardID)
	}
	if status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tickets []model.Ticket
	for rows.Next() {
		var t model.Ticket
		if err := rows.Scan(&t.ID, &t.Title, &t.Content, &t.Status, &t.BoardID, &t.CreatedBy, &t.Assignee, &t.AssignedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (db *DB) UpdateTicket(id string, fields map[string]any) error {
	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}
	var setClauses []string
	var args []any
	for k, v := range fields {
		setClauses = append(setClauses, k+" = ?")
		args = append(args, v)
	}
	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)
	_, err := db.conn.Exec(
		fmt.Sprintf("UPDATE tickets SET %s WHERE id = ?", strings.Join(setClauses, ", ")),
		args...,
	)
	return err
}

func (db *DB) DeleteTicket(id string) error {
	_, err := db.conn.Exec(`DELETE FROM tickets WHERE id = ?`, id)
	return err
}

func (db *DB) GetTicket(id string) (model.Ticket, error) {
	var t model.Ticket
	err := db.conn.QueryRow(
		`SELECT id, title, content, status, board_id, created_by, assignee, assigned_by, created_at, updated_at FROM tickets WHERE id = ?`, id,
	).Scan(&t.ID, &t.Title, &t.Content, &t.Status, &t.BoardID, &t.CreatedBy, &t.Assignee, &t.AssignedBy, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

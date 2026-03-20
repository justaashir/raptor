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

func NewDB(dsn string) (*DB, error) {
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	_, err = conn.Exec(`CREATE TABLE IF NOT EXISTS tickets (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		content TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT 'todo',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) CreateTicket(t model.Ticket) error {
	_, err := db.conn.Exec(
		`INSERT INTO tickets (id, title, content, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		t.ID, t.Title, t.Content, t.Status, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (db *DB) ListTickets(status string) ([]model.Ticket, error) {
	query := `SELECT id, title, content, status, created_at, updated_at FROM tickets`
	var args []any
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
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
		if err := rows.Scan(&t.ID, &t.Title, &t.Content, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
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
		`SELECT id, title, content, status, created_at, updated_at FROM tickets WHERE id = ?`, id,
	).Scan(&t.ID, &t.Title, &t.Content, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

package server

import (
	"database/sql"
	"raptor/model"

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

func (db *DB) GetTicket(id string) (model.Ticket, error) {
	var t model.Ticket
	err := db.conn.QueryRow(
		`SELECT id, title, content, status, created_at, updated_at FROM tickets WHERE id = ?`, id,
	).Scan(&t.ID, &t.Title, &t.Content, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

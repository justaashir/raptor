package server

import (
	"errors"
	"fmt"
	"raptor/model"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var ErrAlreadyMember = errors.New("user is already a member of this workspace")

type DB struct {
	conn *gorm.DB
}

func NewDB(dsn string) (*DB, error) {
	conn, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	// WAL mode for concurrent reads (safe before migration)
	if err := conn.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		return nil, err
	}
	if err := conn.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
		return nil, err
	}
	if err := conn.Exec("PRAGMA synchronous = NORMAL").Error; err != nil {
		return nil, err
	}

	// Explicitly disable foreign keys before migration — do not rely on
	// the SQLite default, which could change across driver versions.
	if err := conn.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
		return nil, err
	}

	if err := conn.AutoMigrate(
		&model.Workspace{},
		&model.WorkspaceMember{},
		&model.Board{},
		&model.Ticket{},
		&schemaVersion{},
	); err != nil {
		return nil, err
	}

	// Enable foreign keys AFTER migration is complete
	if err := conn.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, err
	}

	if err := seed(conn); err != nil {
		return nil, err
	}

	sqlDB, err := conn.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(4)

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	sqlDB, err := db.conn.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Workspace methods

func (db *DB) CreateWorkspace(id, name, createdBy string) error {
	return db.conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model.Workspace{ID: id, Name: name, CreatedBy: createdBy}).Error; err != nil {
			return err
		}
		return tx.Create(&model.WorkspaceMember{
			WorkspaceID: id, Username: createdBy, Role: "owner",
		}).Error
	})
}

func (db *DB) AddWorkspaceMember(workspaceID, username, role string) error {
	err := db.conn.Create(&model.WorkspaceMember{
		WorkspaceID: workspaceID, Username: username, Role: role,
	}).Error
	if err != nil && isDuplicateKeyError(err) {
		return ErrAlreadyMember
	}
	return err
}

func isDuplicateKeyError(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (db *DB) ListWorkspaceMembers(workspaceID string) ([]model.WorkspaceMember, error) {
	var members []model.WorkspaceMember
	err := db.conn.Where("workspace_id = ?", workspaceID).
		Order("created_at").Find(&members).Error
	return members, err
}

func (db *DB) GetMemberRole(workspaceID, username string) (string, error) {
	var member model.WorkspaceMember
	err := db.conn.Select("role").
		Where("workspace_id = ? AND username = ?", workspaceID, username).
		First(&member).Error
	return member.Role, err
}

func (db *DB) CountOwners(workspaceID string) (int64, error) {
	var count int64
	err := db.conn.Model(&model.WorkspaceMember{}).
		Where("workspace_id = ? AND role = ?", workspaceID, "owner").
		Count(&count).Error
	return count, err
}

func (db *DB) RemoveWorkspaceMember(workspaceID, username string) error {
	return db.conn.Where("workspace_id = ? AND username = ?", workspaceID, username).
		Delete(&model.WorkspaceMember{}).Error
}

func (db *DB) DeleteWorkspace(id string) error {
	return db.conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("board_id IN (?)",
			tx.Model(&model.Board{}).Select("id").Where("workspace_id = ?", id),
		).Delete(&model.Ticket{}).Error; err != nil {
			return err
		}
		if err := tx.Where("workspace_id = ?", id).Delete(&model.Board{}).Error; err != nil {
			return err
		}
		if err := tx.Where("workspace_id = ?", id).Delete(&model.WorkspaceMember{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&model.Workspace{}).Error
	})
}

func (db *DB) IsWorkspaceMember(username string) (bool, error) {
	var count int64
	err := db.conn.Model(&model.WorkspaceMember{}).
		Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (db *DB) ListWorkspacesForUser(username string) ([]model.Workspace, error) {
	var workspaces []model.Workspace
	err := db.conn.Joins("JOIN workspace_members ON workspaces.id = workspace_members.workspace_id").
		Where("workspace_members.username = ?", username).
		Order("workspaces.created_at").Find(&workspaces).Error
	return workspaces, err
}

// Board methods

func (db *DB) CreateBoard(id, workspaceID, name, createdBy string, statuses []string) error {
	statusStr := strings.Join(statuses, ",")
	return db.conn.Create(&model.Board{
		ID: id, WorkspaceID: workspaceID, Name: name, Statuses: statusStr, CreatedBy: createdBy,
	}).Error
}

func (db *DB) ListBoardsForUser(workspaceID string) ([]model.Board, error) {
	// All workspace members see all boards — no board-level ACL.
	// Authorization is handled at the handler layer.
	var boards []model.Board
	err := db.conn.Where("workspace_id = ?", workspaceID).
		Order("created_at").Find(&boards).Error
	return boards, err
}

func (db *DB) DeleteBoard(id string) error {
	return db.conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("board_id = ?", id).Delete(&model.Ticket{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Board{}, "id = ?", id).Error
	})
}

func (db *DB) GetBoard(id string) (model.Board, error) {
	var b model.Board
	err := db.conn.First(&b, "id = ?", id).Error
	return b, err
}

func (db *DB) UpdateBoard(id string, fields map[string]any) error {
	return db.conn.Model(&model.Board{}).Where("id = ?", id).Updates(fields).Error
}

// Ticket methods

func (db *DB) CreateTicket(t model.Ticket) error {
	return db.conn.Create(&t).Error
}

func (db *DB) ListTickets(boardID, status string) ([]model.Ticket, error) {
	var tickets []model.Ticket
	q := db.conn.Model(&model.Ticket{})
	if boardID != "" {
		q = q.Where("board_id = ?", boardID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Order("created_at DESC").Find(&tickets).Error
	return tickets, err
}

// likeEscaper escapes SQL LIKE wildcards in user input.
var likeEscaper = strings.NewReplacer("%", "\\%", "_", "\\_")

func (db *DB) SearchTickets(boardID, query string) ([]model.Ticket, error) {
	var tickets []model.Ticket
	// Escape LIKE wildcards in user input
	escaped := likeEscaper.Replace(query)
	q := db.conn.Model(&model.Ticket{}).
		Where("(title LIKE ? ESCAPE '\\' OR content LIKE ? ESCAPE '\\')", "%"+escaped+"%", "%"+escaped+"%")
	if boardID != "" {
		q = q.Where("board_id = ?", boardID)
	}
	err := q.Order("created_at DESC").Find(&tickets).Error
	return tickets, err
}

func (db *DB) TicketStats(boardID string) (map[string]int, error) {
	type result struct {
		Status string
		Count  int
	}
	var results []result
	q := db.conn.Model(&model.Ticket{}).Select("status, count(*) as count").Group("status")
	if boardID != "" {
		q = q.Where("board_id = ?", boardID)
	}
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, r := range results {
		counts[r.Status] = r.Count
	}
	return counts, nil
}

func (db *DB) UpdateTicket(id string, fields map[string]any) error {
	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}
	fields["updated_at"] = time.Now()
	return db.conn.Model(&model.Ticket{}).Where("id = ?", id).Updates(fields).Error
}

func (db *DB) DeleteTicket(id string) error {
	return db.conn.Delete(&model.Ticket{}, "id = ?", id).Error
}

func (db *DB) GetTicket(id string) (model.Ticket, error) {
	var t model.Ticket
	err := db.conn.First(&t, "id = ?", id).Error
	return t, err
}

func (db *DB) ListTicketsMine(boardID, username string) ([]model.Ticket, error) {
	var tickets []model.Ticket
	err := db.conn.Where("board_id = ? AND (created_by = ? OR assignee = ?)", boardID, username, username).
		Order("created_at desc").Find(&tickets).Error
	return tickets, err
}

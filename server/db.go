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

func NewDB(dsn string, seedUsers ...string) (*DB, error) {
	conn, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	// Enable foreign keys
	if err := conn.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, err
	}

	// Auto-migrate all models
	if err := conn.AutoMigrate(
		&model.Ticket{},
		&model.Workspace{},
		&model.WorkspaceMember{},
		&model.Board{},
		&model.BoardMember{},
		&schemaVersion{},
	); err != nil {
		return nil, err
	}

	if err := seed(conn, seedUsers); err != nil {
		return nil, err
	}

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

func (db *DB) UpdateMemberRole(workspaceID, username, role string) error {
	return db.conn.Model(&model.WorkspaceMember{}).
		Where("workspace_id = ? AND username = ?", workspaceID, username).
		Update("role", role).Error
}

func (db *DB) RemoveWorkspaceMember(workspaceID, username string) error {
	return db.conn.Where("workspace_id = ? AND username = ?", workspaceID, username).
		Delete(&model.WorkspaceMember{}).Error
}

func (db *DB) DeleteWorkspace(id string) error {
	return db.conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("workspace_id = ?", id).Delete(&model.WorkspaceMember{}).Error; err != nil {
			return err
		}
		var boards []model.Board
		if err := tx.Where("workspace_id = ?", id).Find(&boards).Error; err != nil {
			return err
		}
		for _, b := range boards {
			if err := tx.Where("board_id = ?", b.ID).Delete(&model.BoardMember{}).Error; err != nil {
				return err
			}
			if err := tx.Where("board_id = ?", b.ID).Delete(&model.Ticket{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("workspace_id = ?", id).Delete(&model.Board{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Workspace{}, "id = ?", id).Error
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

func (db *DB) CreateBoard(id, workspaceID, name, createdBy string) error {
	return db.conn.Create(&model.Board{
		ID: id, WorkspaceID: workspaceID, Name: name, CreatedBy: createdBy,
	}).Error
}

func (db *DB) ListBoardsForUser(workspaceID, username string) ([]model.Board, error) {
	role, err := db.GetMemberRole(workspaceID, username)
	if err != nil {
		return nil, nil
	}

	var boards []model.Board
	if role == "owner" || role == "admin" {
		err = db.conn.Where("workspace_id = ?", workspaceID).
			Order("created_at").Find(&boards).Error
	} else {
		err = db.conn.Joins("JOIN board_members ON boards.id = board_members.board_id").
			Where("boards.workspace_id = ? AND board_members.username = ?", workspaceID, username).
			Order("boards.created_at").Find(&boards).Error
	}
	return boards, err
}

func (db *DB) DeleteBoard(id string) error {
	return db.conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("board_id = ?", id).Delete(&model.BoardMember{}).Error; err != nil {
			return err
		}
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

// Board member methods

func (db *DB) AddBoardMember(boardID, username string) error {
	return db.conn.Create(&model.BoardMember{
		BoardID: boardID, Username: username,
	}).Error
}

func (db *DB) ListBoardMembers(boardID string) ([]model.BoardMember, error) {
	var members []model.BoardMember
	err := db.conn.Where("board_id = ?", boardID).
		Order("created_at").Find(&members).Error
	return members, err
}

func (db *DB) IsBoardMember(boardID, username string) (bool, error) {
	var count int64
	err := db.conn.Model(&model.BoardMember{}).
		Where("board_id = ? AND username = ?", boardID, username).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	// Owners/admins have implicit access
	var member model.WorkspaceMember
	err = db.conn.Joins("JOIN boards ON boards.workspace_id = workspace_members.workspace_id").
		Where("boards.id = ? AND workspace_members.username = ?", boardID, username).
		First(&member).Error
	if err != nil {
		return false, nil
	}
	return member.Role == "owner" || member.Role == "admin", nil
}

func (db *DB) RemoveBoardMember(boardID, username string) error {
	return db.conn.Where("board_id = ? AND username = ?", boardID, username).
		Delete(&model.BoardMember{}).Error
}

// Ticket methods

func (db *DB) CreateTicket(t model.Ticket) error {
	return db.conn.Create(&t).Error
}

func (db *DB) ListAllTickets(boardID string) ([]model.Ticket, error) {
	var tickets []model.Ticket
	q := db.conn.Model(&model.Ticket{})
	if boardID != "" {
		q = q.Where("board_id = ?", boardID)
	}
	err := q.Order("created_at DESC").Find(&tickets).Error
	return tickets, err
}

func (db *DB) ListTickets(boardID, status string) ([]model.Ticket, error) {
	var tickets []model.Ticket
	q := db.conn.Model(&model.Ticket{})
	if boardID != "" {
		q = q.Where("board_id = ?", boardID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	} else {
		q = q.Where("status != ?", model.Closed)
	}
	err := q.Order("created_at DESC").Find(&tickets).Error
	return tickets, err
}

func (db *DB) SearchTickets(boardID, query string) ([]model.Ticket, error) {
	var tickets []model.Ticket
	// Escape LIKE wildcards in user input
	escaped := strings.NewReplacer("%", "\\%", "_", "\\_").Replace(query)
	q := db.conn.Model(&model.Ticket{}).
		Where("status != ?", model.Closed).
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
	counts := map[string]int{
		"todo": 0, "in_progress": 0, "done": 0, "closed": 0,
	}
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

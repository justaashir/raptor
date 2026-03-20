package server

import (
	"fmt"
	"raptor/model"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	conn *gorm.DB
}

func NewDB(dsn string, seedUsers ...string) (*DB, error) {
	conn, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return nil, err
	}

	// Enable foreign keys
	conn.Exec("PRAGMA foreign_keys = ON")

	// Auto-migrate all models
	if err := conn.AutoMigrate(
		&model.Ticket{},
		&model.Workspace{},
		&model.WorkspaceMember{},
		&model.Board{},
		&model.BoardMember{},
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
	err := db.conn.Create(&model.Workspace{ID: id, Name: name, CreatedBy: createdBy}).Error
	if err != nil {
		return err
	}
	return db.conn.Create(&model.WorkspaceMember{
		WorkspaceID: id, Username: createdBy, Role: "owner",
	}).Error
}

func (db *DB) AddWorkspaceMember(workspaceID, username, role string) error {
	return db.conn.Create(&model.WorkspaceMember{
		WorkspaceID: workspaceID, Username: username, Role: role,
	}).Error
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
	// Delete cascade: members, boards, board_members, tickets
	db.conn.Where("workspace_id = ?", id).Delete(&model.WorkspaceMember{})
	// Get boards to cascade tickets and board_members
	var boards []model.Board
	db.conn.Where("workspace_id = ?", id).Find(&boards)
	for _, b := range boards {
		db.conn.Where("board_id = ?", b.ID).Delete(&model.BoardMember{})
		db.conn.Where("board_id = ?", b.ID).Delete(&model.Ticket{})
	}
	db.conn.Where("workspace_id = ?", id).Delete(&model.Board{})
	return db.conn.Delete(&model.Workspace{}, "id = ?", id).Error
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
	db.conn.Where("board_id = ?", id).Delete(&model.BoardMember{})
	db.conn.Where("board_id = ?", id).Delete(&model.Ticket{})
	return db.conn.Delete(&model.Board{}, "id = ?", id).Error
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

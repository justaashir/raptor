package server

import (
	"raptor/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type schemaVersion struct {
	Version int `gorm:"not null"`
}

func seed(conn *gorm.DB, seedUsers []string) error {
	var sv schemaVersion
	if err := conn.First(&sv).Error; err == nil {
		return nil // already seeded
	}

	if len(seedUsers) > 0 {
		wsID := uuid.New().String()[:8]
		bdID := uuid.New().String()[:8]
		owner := seedUsers[0]

		if err := conn.Create(&model.Workspace{ID: wsID, Name: "Default", CreatedBy: owner}).Error; err != nil {
			return err
		}
		if err := conn.Create(&model.WorkspaceMember{WorkspaceID: wsID, Username: owner, Role: "owner"}).Error; err != nil {
			return err
		}

		for _, u := range seedUsers[1:] {
			if err := conn.Create(&model.WorkspaceMember{WorkspaceID: wsID, Username: u, Role: "admin"}).Error; err != nil {
				return err
			}
		}

		if err := conn.Create(&model.Board{ID: bdID, WorkspaceID: wsID, Name: "Default", CreatedBy: owner}).Error; err != nil {
			return err
		}

		// Migrate existing tickets to default board
		conn.Model(&model.Ticket{}).Where("board_id = ''").Update("board_id", bdID)
	}

	return conn.Create(&schemaVersion{Version: 1}).Error
}

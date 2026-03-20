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
	conn.AutoMigrate(&schemaVersion{})

	var sv schemaVersion
	if err := conn.First(&sv).Error; err == nil {
		return nil // already seeded
	}

	if len(seedUsers) > 0 {
		wsID := uuid.New().String()[:8]
		bdID := uuid.New().String()[:8]
		owner := seedUsers[0]

		conn.Create(&model.Workspace{ID: wsID, Name: "Default", CreatedBy: owner})
		conn.Create(&model.WorkspaceMember{WorkspaceID: wsID, Username: owner, Role: "owner"})

		for _, u := range seedUsers[1:] {
			conn.Create(&model.WorkspaceMember{WorkspaceID: wsID, Username: u, Role: "admin"})
		}

		conn.Create(&model.Board{ID: bdID, WorkspaceID: wsID, Name: "Default", CreatedBy: owner})

		// Migrate existing tickets to default board
		conn.Model(&model.Ticket{}).Where("board_id = ''").Update("board_id", bdID)
	}

	conn.Create(&schemaVersion{Version: 1})
	return nil
}

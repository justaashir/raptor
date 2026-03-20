package server

import (
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

	// No default workspace/board seeding — users create their own on first login

	return conn.Create(&schemaVersion{Version: 2}).Error
}

package model

import "time"

type Workspace struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	CreatedBy string    `json:"created_by" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

type WorkspaceMember struct {
	WorkspaceID string    `json:"workspace_id" gorm:"primaryKey"`
	Username    string    `json:"username" gorm:"primaryKey"`
	Role        string    `json:"role" gorm:"not null;check:role IN ('owner','admin','member')"`
	CreatedAt   time.Time `json:"created_at"`
}

type Board struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"not null"`
	Name        string    `json:"name" gorm:"not null"`
	CreatedBy   string    `json:"created_by" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
}

type BoardMember struct {
	BoardID   string    `json:"board_id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
}

func ValidRole(role string) bool {
	switch role {
	case "owner", "admin", "member":
		return true
	}
	return false
}

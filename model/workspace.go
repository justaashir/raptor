package model

import (
	"strings"
	"time"
)

type Workspace struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	CreatedBy string    `json:"created_by" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

type WorkspaceMember struct {
	WorkspaceID string    `json:"workspace_id" gorm:"primaryKey;constraint:OnDelete:CASCADE"`
	Username    string    `json:"username" gorm:"primaryKey"`
	Role        string    `json:"role" gorm:"not null;check:role IN ('owner','member')"`
	CreatedAt   time.Time `json:"created_at"`
}

type Board struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"not null;constraint:OnDelete:CASCADE"`
	Name        string    `json:"name" gorm:"not null"`
	Statuses    string    `json:"statuses" gorm:"not null;default:'todo,in_progress,done'"`
	CreatedBy   string    `json:"created_by" gorm:"not null"`
	CreatedAt   time.Time `json:"created_at"`
}

func (b Board) StatusList() []string {
	if b.Statuses == "" {
		return DefaultStatuses
	}
	var result []string
	for _, s := range strings.Split(b.Statuses, ",") {
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return DefaultStatuses
	}
	return result
}

func (b Board) ValidStatus(s string) bool {
	for _, st := range b.StatusList() {
		if st == s {
			return true
		}
	}
	return false
}

func ValidRole(role string) bool {
	switch role {
	case "owner", "member":
		return true
	}
	return false
}

package model

import "time"

type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type WorkspaceMember struct {
	WorkspaceID string    `json:"workspace_id"`
	Username    string    `json:"username"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

type Board struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type BoardMember struct {
	BoardID   string    `json:"board_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

func ValidRole(role string) bool {
	switch role {
	case "owner", "admin", "member":
		return true
	}
	return false
}

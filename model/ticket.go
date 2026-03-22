package model

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	Todo       Status = "todo"
	InProgress Status = "in_progress"
	Done       Status = "done"
)

var DefaultStatuses = []string{"todo", "in_progress", "done"}

type Ticket struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	Title      string    `json:"title" gorm:"not null"`
	Content    string    `json:"content" gorm:"default:''"`
	Status     Status    `json:"status" gorm:"not null;default:'todo';index:idx_board_status"`
	BoardID    string    `json:"board_id" gorm:"default:'';index:idx_board_status"`
	CreatedBy  string    `json:"created_by" gorm:"default:''"`
	Assignee   string    `json:"assignee" gorm:"default:''"`
	AssignedBy string    `json:"assigned_by" gorm:"default:''"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// GenID returns a new unique identifier.
func GenID() string {
	return uuid.New().String()
}

func NewTicket(title, content, createdBy string) Ticket {
	now := time.Now()
	return Ticket{
		ID:        GenID(),
		Title:     title,
		Content:   content,
		Status:    Todo,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

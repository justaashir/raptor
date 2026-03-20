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

type Ticket struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	Title      string    `json:"title" gorm:"not null"`
	Content    string    `json:"content" gorm:"default:''"`
	Status     Status    `json:"status" gorm:"not null;default:'todo'"`
	BoardID    string    `json:"board_id" gorm:"default:'';constraint:OnDelete:CASCADE"`
	CreatedBy  string    `json:"created_by" gorm:"default:''"`
	Assignee   string    `json:"assignee" gorm:"default:''"`
	AssignedBy string    `json:"assigned_by" gorm:"default:''"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func ValidStatus(s Status) bool {
	switch s {
	case Todo, InProgress, Done:
		return true
	}
	return false
}

func NewTicket(title, content, createdBy string) Ticket {
	now := time.Now()
	return Ticket{
		ID:        uuid.New().String()[:8],
		Title:     title,
		Content:   content,
		Status:    Todo,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

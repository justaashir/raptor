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
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Status    Status    `json:"status"`
	CreatedBy string    `json:"created_by"`
	Assignee  string    `json:"assignee"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"raptor/model"
)

type ticketCache struct {
	BoardID string         `json:"board_id"`
	Tickets []model.Ticket `json:"tickets"`
}

// DefaultCachePath returns the default path for the ticket cache file.
func DefaultCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".raptor-cache.json"), nil
}

// SaveTicketCache writes tickets to the cache file for the given board.
func SaveTicketCache(path, boardID string, tickets []model.Ticket) error {
	data, err := json.Marshal(ticketCache{BoardID: boardID, Tickets: tickets})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// LoadTicketCache reads cached tickets for the given board. Returns nil (no error)
// if the file is missing or the cached board doesn't match.
func LoadTicketCache(path, boardID string) ([]model.Ticket, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil // missing file is not an error
	}
	var c ticketCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, nil // corrupt cache is not an error
	}
	if c.BoardID != boardID {
		return nil, nil
	}
	return c.Tickets, nil
}

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"raptor/model"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{}}
}

func (c *Client) CreateTicket(title, content string) (model.Ticket, error) {
	body, _ := json.Marshal(map[string]string{"title": title, "content": content})
	resp, err := c.http.Post(c.baseURL+"/api/tickets", "application/json", bytes.NewReader(body))
	if err != nil {
		return model.Ticket{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return model.Ticket{}, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	var ticket model.Ticket
	json.NewDecoder(resp.Body).Decode(&ticket)
	return ticket, nil
}

func (c *Client) ListTickets(status string) ([]model.Ticket, error) {
	url := c.baseURL + "/api/tickets"
	if status != "" {
		url += "?status=" + status
	}
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tickets []model.Ticket
	json.NewDecoder(resp.Body).Decode(&tickets)
	return tickets, nil
}

func (c *Client) GetTicket(id string) (model.Ticket, error) {
	resp, err := c.http.Get(c.baseURL + "/api/tickets/" + id)
	if err != nil {
		return model.Ticket{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return model.Ticket{}, fmt.Errorf("ticket %s not found", id)
	}
	var ticket model.Ticket
	json.NewDecoder(resp.Body).Decode(&ticket)
	return ticket, nil
}

func (c *Client) UpdateTicket(id string, fields map[string]any) (model.Ticket, error) {
	body, _ := json.Marshal(fields)
	req, _ := http.NewRequest("PATCH", c.baseURL+"/api/tickets/"+id, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return model.Ticket{}, err
	}
	defer resp.Body.Close()
	var ticket model.Ticket
	json.NewDecoder(resp.Body).Decode(&ticket)
	return ticket, nil
}

func (c *Client) DeleteTicket(id string) error {
	req, _ := http.NewRequest("DELETE", c.baseURL+"/api/tickets/"+id, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

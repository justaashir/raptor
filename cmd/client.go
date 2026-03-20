package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"raptor/model"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) *Client {
	return &Client{baseURL: baseURL, token: token, http: &http.Client{}}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, fmt.Errorf("unauthorized — run `raptor login` to authenticate")
	}
	return resp, nil
}

func (c *Client) CreateTicket(title, content, assignee string) (model.Ticket, error) {
	payload := map[string]string{"title": title, "content": content}
	if assignee != "" {
		payload["assignee"] = assignee
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.baseURL+"/api/tickets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return model.Ticket{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(resp.Body)
		return model.Ticket{}, fmt.Errorf("unexpected status: %d: %s", resp.StatusCode, msg)
	}
	var ticket model.Ticket
	json.NewDecoder(resp.Body).Decode(&ticket)
	return ticket, nil
}

func (c *Client) ListTickets(status string, mine bool) ([]model.Ticket, error) {
	url := c.baseURL + "/api/tickets"
	sep := "?"
	if status != "" {
		url += sep + "status=" + status
		sep = "&"
	}
	if mine {
		url += sep + "mine=true"
	}
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tickets []model.Ticket
	json.NewDecoder(resp.Body).Decode(&tickets)
	return tickets, nil
}

func (c *Client) GetTicket(id string) (model.Ticket, error) {
	req, _ := http.NewRequest("GET", c.baseURL+"/api/tickets/"+id, nil)
	resp, err := c.do(req)
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
	resp, err := c.do(req)
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
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

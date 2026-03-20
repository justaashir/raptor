package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"raptor/model"
)

type Client struct {
	baseURL   string
	token     string
	workspace string
	board     string
	http      *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{baseURL: baseURL, token: token, http: &http.Client{}}
}

func NewScoped(baseURL, token, workspace, board string) *Client {
	return &Client{baseURL: baseURL, token: token, workspace: workspace, board: board, http: &http.Client{}}
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

func (c *Client) ticketsURL() string {
	return fmt.Sprintf("%s/api/workspaces/%s/boards/%s/tickets", c.baseURL, c.workspace, c.board)
}

func (c *Client) CreateTicket(title, content, assignee string) (model.Ticket, error) {
	payload := map[string]string{"title": title, "content": content}
	if assignee != "" {
		payload["assignee"] = assignee
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.ticketsURL(), bytes.NewReader(body))
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
	url := c.ticketsURL()
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
	req, _ := http.NewRequest("GET", c.ticketsURL()+"/"+id, nil)
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
	req, _ := http.NewRequest("PATCH", c.ticketsURL()+"/"+id, bytes.NewReader(body))
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
	req, _ := http.NewRequest("DELETE", c.ticketsURL()+"/"+id, nil)
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

// Workspace methods

func (c *Client) CreateWorkspace(name string) (model.Workspace, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	req, _ := http.NewRequest("POST", c.baseURL+"/api/workspaces/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return model.Workspace{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(resp.Body)
		return model.Workspace{}, fmt.Errorf("unexpected status: %d: %s", resp.StatusCode, msg)
	}
	var ws model.Workspace
	json.NewDecoder(resp.Body).Decode(&ws)
	return ws, nil
}

func (c *Client) ListWorkspaces() ([]model.Workspace, error) {
	req, _ := http.NewRequest("GET", c.baseURL+"/api/workspaces/", nil)
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var workspaces []model.Workspace
	json.NewDecoder(resp.Body).Decode(&workspaces)
	return workspaces, nil
}

func (c *Client) DeleteWorkspace(id string) error {
	req, _ := http.NewRequest("DELETE", c.baseURL+"/api/workspaces/"+id, nil)
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

func (c *Client) ListWorkspaceMembers(wid string) ([]model.WorkspaceMember, error) {
	req, _ := http.NewRequest("GET", c.baseURL+"/api/workspaces/"+wid+"/members", nil)
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var members []model.WorkspaceMember
	json.NewDecoder(resp.Body).Decode(&members)
	return members, nil
}

func (c *Client) InviteWorkspaceMember(wid, username, role string) error {
	body, _ := json.Marshal(map[string]string{"username": username, "role": role})
	req, _ := http.NewRequest("POST", c.baseURL+"/api/workspaces/"+wid+"/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("%s is already a member of this workspace", username)
	}
	if resp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d: %s", resp.StatusCode, msg)
	}
	return nil
}

func (c *Client) KickWorkspaceMember(wid, username string) error {
	req, _ := http.NewRequest("DELETE", c.baseURL+"/api/workspaces/"+wid+"/members/"+username, nil)
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

func (c *Client) ChangeRole(wid, username, role string) error {
	body, _ := json.Marshal(map[string]string{"role": role})
	req, _ := http.NewRequest("PATCH", c.baseURL+"/api/workspaces/"+wid+"/members/"+username, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d: %s", resp.StatusCode, msg)
	}
	return nil
}

// Board methods

func (c *Client) CreateBoard(wid, name string) (model.Board, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	req, _ := http.NewRequest("POST", c.baseURL+"/api/workspaces/"+wid+"/boards", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return model.Board{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(resp.Body)
		return model.Board{}, fmt.Errorf("unexpected status: %d: %s", resp.StatusCode, msg)
	}
	var bd model.Board
	json.NewDecoder(resp.Body).Decode(&bd)
	return bd, nil
}

func (c *Client) ListBoards(wid string) ([]model.Board, error) {
	req, _ := http.NewRequest("GET", c.baseURL+"/api/workspaces/"+wid+"/boards", nil)
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var boards []model.Board
	json.NewDecoder(resp.Body).Decode(&boards)
	return boards, nil
}

func (c *Client) DeleteBoard(wid, bid string) error {
	req, _ := http.NewRequest("DELETE", c.baseURL+"/api/workspaces/"+wid+"/boards/"+bid, nil)
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

func (c *Client) ListBoardMembers(wid, bid string) ([]model.BoardMember, error) {
	req, _ := http.NewRequest("GET", c.baseURL+"/api/workspaces/"+wid+"/boards/"+bid+"/members", nil)
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var members []model.BoardMember
	json.NewDecoder(resp.Body).Decode(&members)
	return members, nil
}

func (c *Client) GrantBoardAccess(wid, bid, username string) error {
	body, _ := json.Marshal(map[string]string{"username": username})
	req, _ := http.NewRequest("POST", c.baseURL+"/api/workspaces/"+wid+"/boards/"+bid+"/members", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d: %s", resp.StatusCode, msg)
	}
	return nil
}

func (c *Client) RevokeBoardAccess(wid, bid, username string) error {
	req, _ := http.NewRequest("DELETE", c.baseURL+"/api/workspaces/"+wid+"/boards/"+bid+"/members/"+username, nil)
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

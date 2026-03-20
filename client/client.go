package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"raptor/model"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	r         *resty.Client
	baseURL   string
	workspace string
	board     string
}

func New(baseURL, token string) *Client {
	r := resty.New().SetBaseURL(baseURL)
	if token != "" {
		r.SetAuthToken(token)
	}
	return &Client{r: r, baseURL: baseURL}
}

func NewScoped(baseURL, token, workspace, board string) *Client {
	c := New(baseURL, token)
	c.workspace = workspace
	c.board = board
	return c
}

func (c *Client) ticketsURL() string {
	return fmt.Sprintf("/api/workspaces/%s/boards/%s/tickets", c.workspace, c.board)
}

// decode unmarshals the response body into dest and checks the status code.
func decode(resp *resty.Response, expected int, dest any) error {
	if resp.StatusCode() == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized — run `raptor login` to authenticate")
	}
	if resp.StatusCode() != expected {
		return fmt.Errorf("unexpected status: %d: %s", resp.StatusCode(), resp.String())
	}
	if dest != nil {
		if err := json.Unmarshal(resp.Body(), dest); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// check validates the status code without decoding.
func check(resp *resty.Response, expected int) error {
	return decode(resp, expected, nil)
}

// --- Ticket methods ---

func (c *Client) CreateTicket(title, content, assignee string) (model.Ticket, error) {
	payload := map[string]string{"title": title, "content": content}
	if assignee != "" {
		payload["assignee"] = assignee
	}
	resp, err := c.r.R().SetBody(payload).Post(c.ticketsURL())
	if err != nil {
		return model.Ticket{}, err
	}
	var ticket model.Ticket
	return ticket, decode(resp, http.StatusCreated, &ticket)
}

type ListOptions struct {
	Status string
	Mine   bool
	All    bool
}

func (c *Client) ListTickets(opts ListOptions) ([]model.Ticket, error) {
	params := map[string]string{}
	if opts.All {
		params["all"] = "true"
	}
	if opts.Status != "" {
		params["status"] = opts.Status
	}
	if opts.Mine {
		params["mine"] = "true"
	}
	resp, err := c.r.R().SetQueryParams(params).Get(c.ticketsURL())
	if err != nil {
		return nil, err
	}
	var tickets []model.Ticket
	return tickets, decode(resp, http.StatusOK, &tickets)
}

func (c *Client) SearchTickets(query string) ([]model.Ticket, error) {
	resp, err := c.r.R().SetQueryParam("q", query).Get(c.ticketsURL())
	if err != nil {
		return nil, err
	}
	var tickets []model.Ticket
	return tickets, decode(resp, http.StatusOK, &tickets)
}

func (c *Client) TicketStats() (map[string]any, error) {
	resp, err := c.r.R().SetQueryParam("stats", "true").Get(c.ticketsURL())
	if err != nil {
		return nil, err
	}
	var result map[string]any
	return result, decode(resp, http.StatusOK, &result)
}

func (c *Client) GetTicket(id string) (model.Ticket, error) {
	resp, err := c.r.R().Get(c.ticketsURL() + "/" + id)
	if err != nil {
		return model.Ticket{}, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return model.Ticket{}, fmt.Errorf("ticket %s not found", id)
	}
	var ticket model.Ticket
	return ticket, decode(resp, http.StatusOK, &ticket)
}

func (c *Client) UpdateTicket(id string, fields map[string]any) (model.Ticket, error) {
	resp, err := c.r.R().SetBody(fields).Patch(c.ticketsURL() + "/" + id)
	if err != nil {
		return model.Ticket{}, err
	}
	var ticket model.Ticket
	return ticket, decode(resp, http.StatusOK, &ticket)
}

func (c *Client) DeleteTicket(id string) error {
	resp, err := c.r.R().Delete(c.ticketsURL() + "/" + id)
	if err != nil {
		return err
	}
	return check(resp, http.StatusNoContent)
}

// --- Workspace methods ---

func (c *Client) CreateWorkspace(name string) (model.Workspace, error) {
	resp, err := c.r.R().SetBody(map[string]string{"name": name}).Post("/api/workspaces/")
	if err != nil {
		return model.Workspace{}, err
	}
	var ws model.Workspace
	return ws, decode(resp, http.StatusCreated, &ws)
}

func (c *Client) ListWorkspaces() ([]model.Workspace, error) {
	resp, err := c.r.R().Get("/api/workspaces/")
	if err != nil {
		return nil, err
	}
	var workspaces []model.Workspace
	return workspaces, decode(resp, http.StatusOK, &workspaces)
}

func (c *Client) DeleteWorkspace(id string) error {
	resp, err := c.r.R().Delete("/api/workspaces/" + id)
	if err != nil {
		return err
	}
	return check(resp, http.StatusNoContent)
}

func (c *Client) ListWorkspaceMembers(wid string) ([]model.WorkspaceMember, error) {
	resp, err := c.r.R().Get("/api/workspaces/" + wid + "/members")
	if err != nil {
		return nil, err
	}
	var members []model.WorkspaceMember
	return members, decode(resp, http.StatusOK, &members)
}

func (c *Client) InviteWorkspaceMember(wid, username, role string) error {
	resp, err := c.r.R().SetBody(map[string]string{"username": username, "role": role}).Post("/api/workspaces/" + wid + "/members")
	if err != nil {
		return err
	}
	if resp.StatusCode() == http.StatusConflict {
		return fmt.Errorf("%s is already a member of this workspace", username)
	}
	return check(resp, http.StatusCreated)
}

func (c *Client) KickWorkspaceMember(wid, username string) error {
	resp, err := c.r.R().Delete("/api/workspaces/" + wid + "/members/" + username)
	if err != nil {
		return err
	}
	return check(resp, http.StatusNoContent)
}

func (c *Client) ChangeRole(wid, username, role string) error {
	resp, err := c.r.R().SetBody(map[string]string{"role": role}).Patch("/api/workspaces/" + wid + "/members/" + username)
	if err != nil {
		return err
	}
	return check(resp, http.StatusOK)
}

// --- Board methods ---

func (c *Client) CreateBoard(wid, name string) (model.Board, error) {
	resp, err := c.r.R().SetBody(map[string]string{"name": name}).Post("/api/workspaces/" + wid + "/boards")
	if err != nil {
		return model.Board{}, err
	}
	var bd model.Board
	return bd, decode(resp, http.StatusCreated, &bd)
}

func (c *Client) ListBoards(wid string) ([]model.Board, error) {
	resp, err := c.r.R().Get("/api/workspaces/" + wid + "/boards")
	if err != nil {
		return nil, err
	}
	var boards []model.Board
	return boards, decode(resp, http.StatusOK, &boards)
}

func (c *Client) DeleteBoard(wid, bid string) error {
	resp, err := c.r.R().Delete("/api/workspaces/" + wid + "/boards/" + bid)
	if err != nil {
		return err
	}
	return check(resp, http.StatusNoContent)
}

func (c *Client) ListBoardMembers(wid, bid string) ([]model.BoardMember, error) {
	resp, err := c.r.R().Get("/api/workspaces/" + wid + "/boards/" + bid + "/members")
	if err != nil {
		return nil, err
	}
	var members []model.BoardMember
	return members, decode(resp, http.StatusOK, &members)
}

func (c *Client) GrantBoardAccess(wid, bid, username string) error {
	resp, err := c.r.R().SetBody(map[string]string{"username": username}).Post("/api/workspaces/" + wid + "/boards/" + bid + "/members")
	if err != nil {
		return err
	}
	return check(resp, http.StatusCreated)
}

func (c *Client) RevokeBoardAccess(wid, bid, username string) error {
	resp, err := c.r.R().Delete("/api/workspaces/" + wid + "/boards/" + bid + "/members/" + username)
	if err != nil {
		return err
	}
	return check(resp, http.StatusNoContent)
}

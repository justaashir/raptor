package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"raptor/model"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"nhooyr.io/websocket"
)

type viewState int

const (
	viewBoard viewState = iota
	viewDetail
	viewAdd
	viewEdit
	viewBoardSelect
)

type App struct {
	serverURL    string
	token        string
	workspace    string
	board        string
	boardName    string
	wsName       string
	columns      [3]*Column
	activeCol    int
	state        viewState
	detailText   string
	err          error
	width        int
	height       int
	quitting     bool
	// Board selector state
	boardChoices []model.Board
	boardCursor  int
}

type ticketsMsg []model.Ticket
type errMsg error
type wsMsg struct{}
type boardsMsg struct {
	boards    []model.Board
	workspace string
}

func NewApp(serverURL, token, workspace, board string) *App {
	app := &App{
		serverURL: serverURL,
		token:     token,
		workspace: workspace,
		board:     board,
		columns: [3]*Column{
			NewColumn("Todo", model.Todo, nil),
			NewColumn("In Progress", model.InProgress, nil),
			NewColumn("Done", model.Done, nil),
		},
	}
	app.columns[0].SetFocused(true)
	return app
}

func (a *App) ActiveColumn() int       { return a.activeCol }
func (a *App) Columns() [3]*Column     { return a.columns }

func (a *App) SetTickets(tickets []model.Ticket) {
	todo := filterByStatus(tickets, model.Todo)
	inProgress := filterByStatus(tickets, model.InProgress)
	done := filterByStatus(tickets, model.Done)
	a.columns[0].SetTickets(todo)
	a.columns[1].SetTickets(inProgress)
	a.columns[2].SetTickets(done)
}

func (a *App) MoveRight() {
	if a.activeCol < 2 {
		a.columns[a.activeCol].SetFocused(false)
		a.activeCol++
		a.columns[a.activeCol].SetFocused(true)
	}
}

func (a *App) MoveLeft() {
	if a.activeCol > 0 {
		a.columns[a.activeCol].SetFocused(false)
		a.activeCol--
		a.columns[a.activeCol].SetFocused(true)
	}
}

func (a *App) SelectedTicket() *model.Ticket {
	return a.columns[a.activeCol].SelectedTicket()
}

func filterByStatus(tickets []model.Ticket, status model.Status) []model.Ticket {
	var result []model.Ticket
	for _, t := range tickets {
		if t.Status == status {
			result = append(result, t)
		}
	}
	return result
}

// Bubble Tea interface

func (a *App) Init() tea.Cmd {
	if a.workspace == "" || a.board == "" {
		return a.fetchBoards
	}
	return tea.Batch(a.fetchTickets, a.listenWS)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case ticketsMsg:
		a.SetTickets([]model.Ticket(msg))

	case wsMsg:
		return a, a.fetchTickets

	case boardsMsg:
		a.boardChoices = msg.boards
		a.wsName = msg.workspace
		a.boardCursor = 0
		a.state = viewBoardSelect
		return a, nil

	case errMsg:
		a.err = msg

	case tea.KeyMsg:
		switch a.state {
		case viewBoardSelect:
			return a.updateBoardSelect(msg)
		case viewDetail:
			return a.updateDetail(msg)
		default:
			return a.updateBoard(msg)
		}
	}
	return a, nil
}

func (a *App) updateBoardSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		a.quitting = true
		return a, tea.Quit
	case key.Matches(msg, keys.Up):
		if a.boardCursor > 0 {
			a.boardCursor--
		}
	case key.Matches(msg, keys.Down):
		if a.boardCursor < len(a.boardChoices)-1 {
			a.boardCursor++
		}
	case key.Matches(msg, keys.Enter):
		if len(a.boardChoices) > 0 {
			selected := a.boardChoices[a.boardCursor]
			a.board = selected.ID
			a.boardName = selected.Name
			a.workspace = selected.WorkspaceID
			a.state = viewBoard
			return a, tea.Batch(a.fetchTickets, a.listenWS)
		}
	}
	return a, nil
}

func (a *App) updateBoard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		a.quitting = true
		return a, tea.Quit
	case key.Matches(msg, keys.Left):
		a.MoveLeft()
	case key.Matches(msg, keys.Right):
		a.MoveRight()
	case key.Matches(msg, keys.Up):
		a.columns[a.activeCol].MoveUp()
	case key.Matches(msg, keys.Down):
		a.columns[a.activeCol].MoveDown()
	case key.Matches(msg, keys.Enter):
		if t := a.SelectedTicket(); t != nil {
			a.state = viewDetail
			a.detailText = RenderTicketDetail(*t)
		}
	case key.Matches(msg, keys.Move):
		if t := a.SelectedTicket(); t != nil {
			return a, a.cycleStatus(t)
		}
	case key.Matches(msg, keys.Delete):
		if t := a.SelectedTicket(); t != nil {
			return a, a.deleteTicket(t.ID)
		}
	case key.Matches(msg, keys.Refresh):
		return a, a.fetchTickets
	case key.Matches(msg, keys.New):
		return a, a.addTicket
	case key.Matches(msg, keys.Edit):
		if t := a.SelectedTicket(); t != nil {
			return a, a.editTicket(t)
		}
	case key.Matches(msg, keys.SwitchBoard):
		return a, a.fetchBoards
	}
	return a, nil
}

func (a *App) updateDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Back), key.Matches(msg, keys.Quit):
		a.state = viewBoard
	}
	return a, nil
}

func (a *App) View() string {
	if a.quitting {
		return ""
	}

	if a.state == viewBoardSelect {
		return a.viewBoardSelector()
	}

	if a.state == viewDetail {
		return a.detailText + "\n\nPress Esc or q to go back"
	}

	// Header with workspace > board
	var header string
	if a.wsName != "" || a.boardName != "" {
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
		header = headerStyle.Render(fmt.Sprintf("%s > %s", a.wsName, a.boardName)) + "\n\n"
	}

	var cols []string
	for _, col := range a.columns {
		cols = append(cols, col.View())
	}

	board := lipgloss.JoinHorizontal(lipgloss.Top, cols...)

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
		"←/h ↑/k ↓/j →/l navigate • enter view • m move • d delete • n new • e edit • b boards • r refresh • q quit",
	)

	if a.err != nil {
		help = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error: %v", a.err))
	}

	return header + board + "\n\n" + help
}

func (a *App) viewBoardSelector() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	s := titleStyle.Render("Select a board") + "\n\n"

	if len(a.boardChoices) == 0 {
		return s + "No boards found. Create one with 'raptor board create'.\n\nPress q to quit."
	}

	for i, b := range a.boardChoices {
		cursor := "  "
		if i == a.boardCursor {
			cursor = "> "
		}
		s += fmt.Sprintf("%s%s (%s)\n", cursor, b.Name, b.ID)
	}
	s += "\n↑/↓ navigate • enter select • q quit"
	return s
}

// Commands

func (a *App) fetchTickets() tea.Msg {
	client := newHTTPClient(a.serverURL, a.token, a.workspace, a.board)
	resp, err := client.ListTickets("")
	if err != nil {
		return errMsg(err)
	}
	return ticketsMsg(resp)
}

func (a *App) fetchBoards() tea.Msg {
	// Fetch workspaces first, then boards from first workspace
	wsResp, err := httpGet(a.serverURL+"/api/workspaces/", a.token)
	if err != nil {
		return errMsg(err)
	}
	var workspaces []model.Workspace
	json.Unmarshal(wsResp, &workspaces)

	if len(workspaces) == 0 {
		return errMsg(fmt.Errorf("no workspaces found"))
	}

	// Use configured workspace or first one
	ws := workspaces[0]
	if a.workspace != "" {
		for _, w := range workspaces {
			if w.ID == a.workspace {
				ws = w
				break
			}
		}
	}

	boardsResp, err := httpGet(a.serverURL+"/api/workspaces/"+ws.ID+"/boards", a.token)
	if err != nil {
		return errMsg(err)
	}
	var boards []model.Board
	json.Unmarshal(boardsResp, &boards)

	// If exactly one workspace and one board, auto-select
	if len(workspaces) == 1 && len(boards) == 1 {
		a.workspace = ws.ID
		a.wsName = ws.Name
		a.board = boards[0].ID
		a.boardName = boards[0].Name
		a.state = viewBoard
		return a.fetchTickets()
	}

	return boardsMsg{boards: boards, workspace: ws.Name}
}

func (a *App) listenWS() tea.Msg {
	ctx := context.Background()
	wsURL := strings.Replace(a.serverURL, "http", "ws", 1) + "/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return errMsg(err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return errMsg(err)
		}
		var ev map[string]string
		json.Unmarshal(data, &ev)
		if ev["event"] == "ticket_changed" {
			return wsMsg{}
		}
	}
}

func (a *App) cycleStatus(t *model.Ticket) tea.Cmd {
	return func() tea.Msg {
		next := map[model.Status]model.Status{
			model.Todo:       model.InProgress,
			model.InProgress: model.Done,
			model.Done:       model.Todo,
		}
		newStatus := next[t.Status]
		client := newHTTPClient(a.serverURL, a.token, a.workspace, a.board)
		_, err := client.UpdateTicket(t.ID, map[string]any{"status": string(newStatus)})
		if err != nil {
			return errMsg(err)
		}
		return a.fetchTickets()
	}
}

func (a *App) addTicket() tea.Msg {
	title, content, err := NewAddForm()
	if err != nil || title == "" {
		return a.fetchTickets()
	}
	client := newHTTPClient(a.serverURL, a.token, a.workspace, a.board)
	body, _ := json.Marshal(map[string]string{"title": title, "content": content})
	_, postErr := httpPost(client.ticketsURL(), body, a.token)
	if postErr != nil {
		return errMsg(postErr)
	}
	return a.fetchTickets()
}

func (a *App) editTicket(t *model.Ticket) tea.Cmd {
	return func() tea.Msg {
		title, content, err := NewEditForm(t.Title, t.Content)
		if err != nil {
			return a.fetchTickets()
		}
		fields := map[string]any{}
		if title != t.Title {
			fields["title"] = title
		}
		if content != t.Content {
			fields["content"] = content
		}
		if len(fields) > 0 {
			client := newHTTPClient(a.serverURL, a.token, a.workspace, a.board)
			_, err := client.UpdateTicket(t.ID, fields)
			if err != nil {
				return errMsg(err)
			}
		}
		return a.fetchTickets()
	}
}

func (a *App) deleteTicket(id string) tea.Cmd {
	return func() tea.Msg {
		client := newHTTPClient(a.serverURL, a.token, a.workspace, a.board)
		err := client.DeleteTicket(id)
		if err != nil {
			return errMsg(err)
		}
		return a.fetchTickets()
	}
}

// Key bindings
type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Enter       key.Binding
	Move        key.Binding
	Delete      key.Binding
	Refresh     key.Binding
	New         key.Binding
	Edit        key.Binding
	Quit        key.Binding
	Back        key.Binding
	SwitchBoard key.Binding
}

var keys = keyMap{
	Up:          key.NewBinding(key.WithKeys("up", "k")),
	Down:        key.NewBinding(key.WithKeys("down", "j")),
	Left:        key.NewBinding(key.WithKeys("left", "h")),
	Right:       key.NewBinding(key.WithKeys("right", "l")),
	Enter:       key.NewBinding(key.WithKeys("enter")),
	Move:        key.NewBinding(key.WithKeys("m")),
	Delete:      key.NewBinding(key.WithKeys("d")),
	Refresh:     key.NewBinding(key.WithKeys("r")),
	New:         key.NewBinding(key.WithKeys("n")),
	Edit:        key.NewBinding(key.WithKeys("e")),
	Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Back:        key.NewBinding(key.WithKeys("esc")),
	SwitchBoard: key.NewBinding(key.WithKeys("b")),
}

// HTTP client adapter (reuses logic but avoids import cycle with cmd)

type httpClient struct {
	baseURL   string
	token     string
	workspace string
	board     string
}

func newHTTPClient(baseURL, token, workspace, board string) *httpClient {
	return &httpClient{baseURL: baseURL, token: token, workspace: workspace, board: board}
}

func (c *httpClient) ticketsURL() string {
	return fmt.Sprintf("%s/api/workspaces/%s/boards/%s/tickets", c.baseURL, c.workspace, c.board)
}

func (c *httpClient) ListTickets(status string) ([]model.Ticket, error) {
	url := c.ticketsURL()
	if status != "" {
		url += "?status=" + status
	}
	resp, err := httpGet(url, c.token)
	if err != nil {
		return nil, err
	}
	var tickets []model.Ticket
	json.Unmarshal(resp, &tickets)
	return tickets, nil
}

func (c *httpClient) UpdateTicket(id string, fields map[string]any) (model.Ticket, error) {
	body, _ := json.Marshal(fields)
	resp, err := httpPatch(c.ticketsURL()+"/"+id, body, c.token)
	if err != nil {
		return model.Ticket{}, err
	}
	var ticket model.Ticket
	json.Unmarshal(resp, &ticket)
	return ticket, nil
}

func (c *httpClient) DeleteTicket(id string) error {
	return httpDelete(c.ticketsURL()+"/"+id, c.token)
}

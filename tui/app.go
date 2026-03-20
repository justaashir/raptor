package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"raptor/client"
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
	viewBoardSelect
)

type App struct {
	serverURL string
	token     string
	workspace string
	board     string
	boardName string
	wsName    string
	statuses  []string
	columns   []*Column
	activeCol int
	state     viewState
	detailText string
	err       error
	width     int
	height    int
	quitting  bool
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
type boardInfoMsg struct {
	statuses []string
	name     string
}

func NewApp(serverURL, token, workspace, board string) *App {
	return &App{
		serverURL: serverURL,
		token:     token,
		workspace: workspace,
		board:     board,
		statuses:  model.DefaultStatuses,
		columns:   makeColumns(model.DefaultStatuses),
	}
}

func makeColumns(statuses []string) []*Column {
	cols := make([]*Column, len(statuses))
	for i, s := range statuses {
		title := strings.ReplaceAll(s, "_", " ")
		// Title case
		if len(title) > 0 {
			title = strings.ToUpper(title[:1]) + title[1:]
		}
		cols[i] = NewColumn(title, model.Status(s), nil)
	}
	if len(cols) > 0 {
		cols[0].SetFocused(true)
	}
	return cols
}

func (a *App) ActiveColumn() int  { return a.activeCol }
func (a *App) Columns() []*Column { return a.columns }

func (a *App) SetTickets(tickets []model.Ticket) {
	for _, col := range a.columns {
		filtered := filterByStatus(tickets, col.Status())
		col.SetTickets(filtered)
	}
}

func (a *App) MoveRight() {
	if a.activeCol < len(a.columns)-1 {
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
	if a.activeCol >= len(a.columns) {
		return nil
	}
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
	return tea.Batch(a.fetchBoardInfo, a.listenWS)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case boardInfoMsg:
		a.statuses = msg.statuses
		a.boardName = msg.name
		a.columns = makeColumns(a.statuses)
		if a.activeCol >= len(a.columns) {
			a.activeCol = 0
		}
		if len(a.columns) > 0 {
			a.columns[a.activeCol].SetFocused(true)
		}
		return a, a.fetchTickets

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
			a.statuses = selected.StatusList()
			a.columns = makeColumns(a.statuses)
			a.activeCol = 0
			if len(a.columns) > 0 {
				a.columns[0].SetFocused(true)
			}
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

func (a *App) fetchBoardInfo() tea.Msg {
	c := client.New(a.serverURL, a.token)
	boards, err := c.ListBoards(a.workspace)
	if err != nil {
		return errMsg(err)
	}
	for _, b := range boards {
		if b.ID == a.board {
			return boardInfoMsg{statuses: b.StatusList(), name: b.Name}
		}
	}
	// Board not found, use defaults
	return boardInfoMsg{statuses: model.DefaultStatuses, name: ""}
}

func (a *App) fetchTickets() tea.Msg {
	c := client.NewScoped(a.serverURL, a.token, a.workspace, a.board)
	resp, err := c.ListTickets(client.ListOptions{})
	if err != nil {
		return errMsg(err)
	}
	return ticketsMsg(resp)
}

func (a *App) fetchBoards() tea.Msg {
	c := client.New(a.serverURL, a.token)

	workspaces, err := c.ListWorkspaces()
	if err != nil {
		return errMsg(err)
	}

	if len(workspaces) == 0 {
		return errMsg(fmt.Errorf("no workspaces found — create one with 'raptor workspace create'"))
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

	boards, err := c.ListBoards(ws.ID)
	if err != nil {
		return errMsg(err)
	}

	// If exactly one workspace and one board, auto-select
	if len(workspaces) == 1 && len(boards) == 1 {
		a.workspace = ws.ID
		a.wsName = ws.Name
		a.board = boards[0].ID
		a.boardName = boards[0].Name
		a.statuses = boards[0].StatusList()
		a.columns = makeColumns(a.statuses)
		a.activeCol = 0
		if len(a.columns) > 0 {
			a.columns[0].SetFocused(true)
		}
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
		// Cycle through statuses in order
		var newStatus string
		for i, s := range a.statuses {
			if s == string(t.Status) {
				newStatus = a.statuses[(i+1)%len(a.statuses)]
				break
			}
		}
		if newStatus == "" && len(a.statuses) > 0 {
			newStatus = a.statuses[0]
		}
		c := client.NewScoped(a.serverURL, a.token, a.workspace, a.board)
		_, err := c.UpdateTicket(t.ID, map[string]any{"status": newStatus})
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
	c := client.NewScoped(a.serverURL, a.token, a.workspace, a.board)
	_, createErr := c.CreateTicket(title, content, "")
	if createErr != nil {
		return errMsg(createErr)
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
			c := client.NewScoped(a.serverURL, a.token, a.workspace, a.board)
			_, err := c.UpdateTicket(t.ID, fields)
			if err != nil {
				return errMsg(err)
			}
		}
		return a.fetchTickets()
	}
}

func (a *App) deleteTicket(id string) tea.Cmd {
	return func() tea.Msg {
		c := client.NewScoped(a.serverURL, a.token, a.workspace, a.board)
		err := c.DeleteTicket(id)
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

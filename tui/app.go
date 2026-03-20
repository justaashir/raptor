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
	viewList viewState = iota
	viewAdd
	viewEdit
	viewBoardSelect
)

type App struct {
	serverURL  string
	token      string
	workspace  string
	board      string
	boardName  string
	wsName     string
	listPane   *ListPane
	detailPane *DetailPane
	focused    focusPane
	tickets    []model.Ticket
	state      viewState
	err        error
	width      int
	height     int
	quitting   bool
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
		serverURL:  serverURL,
		token:      token,
		workspace:  workspace,
		board:      board,
		focused:    focusList,
		listPane:   NewListPane(40, 20),
		detailPane: NewDetailPane(60, 20),
	}
	return app
}

// initPanes recalculates pane sizes based on current terminal dimensions.
func (a *App) initPanes() {
	listW, detailW, h := a.paneSizes()
	a.listPane.SetSize(listW, h)
	a.detailPane.SetSize(detailW, h)
}

func (a *App) paneSizes() (listW, detailW, contentH int) {
	contentH = a.height - 2 // 1 for status bar, 1 for border
	if contentH < 5 {
		contentH = 5
	}
	totalW := a.width
	listW = totalW * 35 / 100
	if listW < 30 {
		listW = 30
	}
	detailW = totalW - listW - 3 // 3 for borders/gap
	if detailW < 20 {
		detailW = 20
	}
	return
}

func (a *App) SetTickets(tickets []model.Ticket) {
	a.tickets = tickets
	a.listPane.SetTickets(tickets)
	// Update detail pane with selected ticket
	a.updateDetail()
}

func (a *App) updateDetail() {
	selected := a.listPane.SelectedTicket()
	a.detailPane.SetTicket(selected)
}

func (a *App) SelectedTicket() *model.Ticket {
	return a.listPane.SelectedTicket()
}

func (a *App) toggleFocus() {
	if a.focused == focusList {
		a.focused = focusDetail
		a.listPane.Blur()
	} else {
		a.focused = focusList
		a.listPane.Focus()
	}
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
		a.initPanes()

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
		default:
			return a.updateList(msg)
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
			a.state = viewList
			return a, tea.Batch(a.fetchTickets, a.listenWS)
		}
	}
	return a, nil
}

func (a *App) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		a.quitting = true
		return a, tea.Quit

	case key.Matches(msg, keys.Tab):
		a.toggleFocus()
		return a, nil

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

	default:
		// Delegate to focused pane
		if a.focused == focusList {
			prevCursor := a.listPane.Cursor()
			cmd := a.listPane.Update(msg)
			if a.listPane.Cursor() != prevCursor {
				a.updateDetail()
			}
			return a, cmd
		}
		cmd := a.detailPane.Update(msg)
		return a, cmd
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

	// Split pane layout
	listStyle := UnfocusedBorderStyle
	detailStyle := UnfocusedBorderStyle
	if a.focused == focusList {
		listStyle = FocusedBorderStyle
	} else {
		detailStyle = FocusedBorderStyle
	}

	listW, detailW, contentH := a.paneSizes()

	listView := listStyle.
		Width(listW).
		Height(contentH).
		Render(a.listPane.View())

	detailView := detailStyle.
		Width(detailW).
		Height(contentH).
		Render(a.detailPane.View())

	panes := lipgloss.JoinHorizontal(lipgloss.Top, listView, detailView)

	statusBar := RenderStatusBar(a.tickets, a.boardName, a.focused, a.width)

	if a.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		statusBar = errStyle.Render(fmt.Sprintf("Error: %v", a.err))
	}

	return lipgloss.JoinVertical(lipgloss.Left, panes, statusBar)
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
		a.state = viewList
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
			model.Closed:     model.Todo,
		}
		newStatus := next[t.Status]
		c := client.NewScoped(a.serverURL, a.token, a.workspace, a.board)
		_, err := c.UpdateTicket(t.ID, map[string]any{"status": string(newStatus)})
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
	Tab         key.Binding
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
	Tab:         key.NewBinding(key.WithKeys("tab")),
}

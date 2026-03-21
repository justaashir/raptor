package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"raptor/client"
	"raptor/model"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"nhooyr.io/websocket"
)

type viewState int

const (
	viewList viewState = iota
	viewBoardSelect
	viewWorkspaceSelect
	viewCreate
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
	// Workspace selector state
	wsChoices []model.Workspace
	wsCursor  int
	// Create form state
	createForm *huh.Form
	newTitle   string
	newContent string
}

type ticketsMsg []model.Ticket
type errMsg error
type wsMsg struct{}
type boardsMsg struct {
	boards    []model.Board
	workspace string
}
type workspacesMsg struct {
	workspaces []model.Workspace
}
type ticketCreatedMsg struct{}

func NewApp(serverURL, token, workspace, board string) *App {
	return &App{
		serverURL:  serverURL,
		token:      token,
		workspace:  workspace,
		board:      board,
		focused:    focusList,
		listPane:   NewListPane(40, 20),
		detailPane: NewDetailPane(60, 20),
	}
}

func (a *App) initPanes() {
	listW, detailW, h := a.paneSizes()
	a.listPane.SetSize(listW, h)
	a.detailPane.SetSize(detailW, h)
}

func (a *App) paneSizes() (listW, detailW, contentH int) {
	contentH = a.height - 4 // header + status bar + borders
	if contentH < 5 {
		contentH = 5
	}
	totalW := a.width
	listW = totalW * 35 / 100
	if listW < 30 {
		listW = 30
	}
	detailW = totalW - listW - 3
	if detailW < 20 {
		detailW = 20
	}
	return
}

func (a *App) SetTickets(tickets []model.Ticket) {
	a.tickets = tickets
	a.listPane.SetTickets(tickets)
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
	} else {
		a.focused = focusList
	}
}

// Bubble Tea interface

func (a *App) Init() tea.Cmd {
	if a.workspace == "" {
		return a.fetchWorkspaces
	}
	if a.board == "" {
		return a.fetchBoardsForWorkspace
	}
	return tea.Batch(a.fetchTickets, a.listenWS)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Route all messages to create form when active
	if a.state == viewCreate {
		return a.updateCreate(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.initPanes()

	case ticketsMsg:
		a.SetTickets([]model.Ticket(msg))

	case wsMsg:
		return a, tea.Batch(a.fetchTickets, a.listenWS)

	case workspacesMsg:
		if len(msg.workspaces) == 1 {
			a.workspace = msg.workspaces[0].ID
			a.wsName = msg.workspaces[0].Name
			return a, a.fetchBoardsForWorkspace
		}
		a.wsChoices = msg.workspaces
		a.wsCursor = 0
		a.state = viewWorkspaceSelect
		return a, nil

	case boardsMsg:
		if len(msg.boards) == 1 {
			a.board = msg.boards[0].ID
			a.boardName = msg.boards[0].Name
			a.workspace = msg.boards[0].WorkspaceID
			a.wsName = msg.workspace
			a.state = viewList
			return a, tea.Batch(a.fetchTickets, a.listenWS)
		}
		a.boardChoices = msg.boards
		a.wsName = msg.workspace
		a.boardCursor = 0
		a.state = viewBoardSelect
		return a, nil

	case ticketCreatedMsg:
		return a, a.fetchTickets

	case errMsg:
		a.err = msg

	case tea.KeyMsg:
		switch a.state {
		case viewWorkspaceSelect:
			return a.updateWorkspaceSelect(msg)
		case viewBoardSelect:
			return a.updateBoardSelect(msg)
		default:
			return a.updateList(msg)
		}
	}
	return a, nil
}

func (a *App) updateWorkspaceSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		a.quitting = true
		return a, tea.Quit
	case key.Matches(msg, keys.Up):
		if a.wsCursor > 0 {
			a.wsCursor--
		}
	case key.Matches(msg, keys.Down):
		if a.wsCursor < len(a.wsChoices)-1 {
			a.wsCursor++
		}
	case key.Matches(msg, keys.Enter):
		if len(a.wsChoices) > 0 {
			selected := a.wsChoices[a.wsCursor]
			a.workspace = selected.ID
			a.wsName = selected.Name
			return a, a.fetchBoardsForWorkspace
		}
	}
	return a, nil
}

func (a *App) updateBoardSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		a.quitting = true
		return a, tea.Quit
	case key.Matches(msg, keys.Back):
		a.workspace = ""
		a.wsName = ""
		return a, a.fetchWorkspaces
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
	// When filtering, pass ALL keys to the list
	if a.listPane.Filtering() {
		prevCursor := a.listPane.Cursor()
		cmd := a.listPane.Update(msg)
		if a.listPane.Cursor() != prevCursor {
			a.updateDetail()
		}
		return a, cmd
	}

	switch {
	case key.Matches(msg, keys.Quit):
		a.quitting = true
		return a, tea.Quit

	case key.Matches(msg, keys.Tab):
		a.toggleFocus()
		return a, nil

	case key.Matches(msg, keys.Refresh):
		return a, a.fetchTickets

	case key.Matches(msg, keys.SwitchBoard):
		return a, a.fetchBoardsForWorkspace

	case key.Matches(msg, keys.SwitchWorkspace):
		return a, a.fetchWorkspaces

	case key.Matches(msg, keys.Create):
		return a, a.startCreateForm()

	default:
		// Delegate to focused pane — list handles j/k, /, pgup/pgdn
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
}

func (a *App) View() string {
	if a.quitting {
		return ""
	}

	if a.state == viewCreate {
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
		return titleStyle.Render("New ticket") + "\n\n" + a.createForm.View()
	}

	if a.state == viewWorkspaceSelect {
		return a.viewWorkspaceSelector()
	}

	if a.state == viewBoardSelect {
		return a.viewBoardSelector()
	}

	// Header
	var header string
	if a.wsName != "" || a.boardName != "" {
		ws := lipgloss.NewStyle().Foreground(colorComment).Render(a.wsName)
		sep := lipgloss.NewStyle().Foreground(colorComment).Render(" > ")
		board := lipgloss.NewStyle().Bold(true).Foreground(colorPink).Render(a.boardName)
		header = lipgloss.NewStyle().Padding(0, 1).Render(ws+sep+board) + "\n"
	}

	listStyle := UnfocusedBorderStyle
	detailStyle := UnfocusedBorderStyle
	if a.focused == focusList {
		listStyle = FocusedBorderStyle
	} else {
		detailStyle = FocusedBorderStyle
	}

	listW, detailW, contentH := a.paneSizes()

	// Column header + list content inside the border
	listContent := a.listPane.ColumnHeader() + "\n" + a.listPane.View()
	listView := listStyle.
		Width(listW).
		Height(contentH).
		Render(listContent)

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

	return lipgloss.JoinVertical(lipgloss.Left, header, panes, statusBar)
}

func (a *App) viewWorkspaceSelector() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	s := titleStyle.Render("Select a workspace") + "\n\n"

	if len(a.wsChoices) == 0 {
		return s + "No workspaces found. Create one with 'raptor workspace create'.\n\nPress q to quit."
	}

	for i, w := range a.wsChoices {
		cursor := "  "
		if i == a.wsCursor {
			cursor = "> "
		}
		s += fmt.Sprintf("%s%s\n", cursor, w.Name)
	}
	s += "\n↑/↓ navigate • enter select • q quit"
	return s
}

func (a *App) viewBoardSelector() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	wsStyle := lipgloss.NewStyle().Foreground(colorComment)
	s := titleStyle.Render("Select a board") + "  " + wsStyle.Render(a.wsName) + "\n\n"

	if len(a.boardChoices) == 0 {
		return s + "No boards found. Create one with 'raptor board create'.\n\nPress esc to go back • q to quit."
	}

	for i, b := range a.boardChoices {
		cursor := "  "
		if i == a.boardCursor {
			cursor = "> "
		}
		s += fmt.Sprintf("%s%s (%s)\n", cursor, b.Name, b.ID)
	}
	s += "\n↑/↓ navigate • enter select • esc back • q quit"
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

func (a *App) startCreateForm() tea.Cmd {
	a.newTitle = ""
	a.newContent = ""
	a.createForm = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Title").
				Value(&a.newTitle),
			huh.NewText().
				Title("Content (markdown)").
				Value(&a.newContent),
		),
	).WithWidth(50).WithShowHelp(true).WithShowErrors(true)
	a.state = viewCreate
	return a.createForm.Init()
}

func (a *App) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check for esc to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, keys.Quit) {
			a.state = viewList
			return a, nil
		}
	}

	form, cmd := a.createForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		a.createForm = f
	}

	if a.createForm.State == huh.StateCompleted {
		a.state = viewList
		if a.newTitle == "" {
			return a, nil
		}
		return a, a.submitTicket
	}
	if a.createForm.State == huh.StateAborted {
		a.state = viewList
		return a, nil
	}

	return a, cmd
}

func (a *App) submitTicket() tea.Msg {
	c := client.NewScoped(a.serverURL, a.token, a.workspace, a.board)
	_, err := c.CreateTicket(a.newTitle, a.newContent, "")
	if err != nil {
		return errMsg(err)
	}
	return ticketCreatedMsg{}
}

func (a *App) fetchWorkspaces() tea.Msg {
	c := client.New(a.serverURL, a.token)
	workspaces, err := c.ListWorkspaces()
	if err != nil {
		return errMsg(err)
	}
	if len(workspaces) == 0 {
		return errMsg(fmt.Errorf("no workspaces found"))
	}
	return workspacesMsg{workspaces: workspaces}
}

func (a *App) fetchBoardsForWorkspace() tea.Msg {
	c := client.New(a.serverURL, a.token)
	boards, err := c.ListBoards(a.workspace)
	if err != nil {
		return errMsg(err)
	}
	return boardsMsg{boards: boards, workspace: a.wsName}
}

func (a *App) listenWS() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := strings.Replace(a.serverURL, "http", "ws", 1) + "/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return errMsg(err)
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	// Read with a long-lived context (not the dial timeout)
	readCtx := context.Background()
	for {
		_, data, err := c.Read(readCtx)
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

// Key bindings
type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	Enter       key.Binding
	Refresh     key.Binding
	Quit        key.Binding
	Back        key.Binding
	SwitchBoard     key.Binding
	SwitchWorkspace key.Binding
	Create          key.Binding
	Tab             key.Binding
}

var keys = keyMap{
	Up:          key.NewBinding(key.WithKeys("up", "k")),
	Down:        key.NewBinding(key.WithKeys("down", "j")),
	Enter:       key.NewBinding(key.WithKeys("enter")),
	Refresh:     key.NewBinding(key.WithKeys("r")),
	Quit:        key.NewBinding(key.WithKeys("q", "ctrl+c")),
	Back:        key.NewBinding(key.WithKeys("esc")),
	SwitchBoard:     key.NewBinding(key.WithKeys("b")),
	SwitchWorkspace: key.NewBinding(key.WithKeys("w")),
	Create:          key.NewBinding(key.WithKeys("n")),
	Tab:             key.NewBinding(key.WithKeys("tab")),
}

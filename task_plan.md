# Task Plan — Raptor TUI Rebuild

## Goal
Replace the 3-column kanban TUI with a beads_viewer-inspired split-pane layout: ticket list table (left 35%) + detail viewport (right 65%) + status bar.

## Isolation Constraint
The TUI (`tui/` package) is a self-contained module. The ONLY external contract is:
```go
// cmd/root.go:70 — the sole call site
app := tui.NewApp(serverURL, authToken, activeWS, activeBoard)
// app implements tea.Model (Init, Update, View)
```
**Rule**: We can rewrite everything inside `tui/` freely. As long as `NewApp(serverURL, token, workspace, board string) *App` returns a `tea.Model`, nothing in `cmd/`, `server/`, `model/`, or `client/` is affected. Zero changes outside `tui/`.

## Phase 1: Setup & Styles
**Branch**: `feat/tui-rebuild`

### 1.1 Create feature branch
- `git checkout -b feat/tui-rebuild`

### 1.2 Create `tui/styles.go` — centralized style constants
- Color palette: status colors (todo=orange, in_progress=cyan, done=green, closed=red)
- Pane border styles (focused vs unfocused)
- Status bar styles
- Header/metadata styles for detail pane
- **Test**: styles_test.go — verify StatusColor returns correct color per status

## Phase 2: List Pane (bubbles/table)

### 2.1 Create `tui/age.go` — age formatting utility
- `FormatAge(t time.Time) string` — returns "5h", "2d", "1w", "3m", etc.
- **Test**: age_test.go — edge cases: just now, hours, days, weeks, months

### 2.2 Create `tui/list_pane.go` — table-based ticket list
- Use `github.com/charmbracelet/bubbles/table` component
- Columns: STATUS | ID | ASSIGNEE | AGE | TITLE
- STATUS: colored text (TODO, IN_PROG, DONE, CLOSED)
- ID: 8-char short ID
- ASSIGNEE: @username or "--"
- AGE: relative time (5h, 2d, 1w, 1m) via FormatAge
- TITLE: truncated to fit remaining width
- Selected row highlighted with inverted colors
- Dynamic width: table adapts to available terminal width (35% of total)
- **Test**: list_pane_test.go
  - FormatStatus() returns correct abbreviated text
  - SetTickets() populates table rows correctly
  - SelectedTicket() returns correct ticket
  - Selection preserved after data refresh (by ticket ID)

## Phase 3: Detail Pane (bubbles/viewport)

### 3.1 Create `tui/detail_pane.go` — scrollable detail viewport
- Use `github.com/charmbracelet/bubbles/viewport` component
- Renders: title (bold, purple), metadata row (status/ID/assignee/dates/creator), markdown content via Glamour
- Viewport scrollable with j/k when focused
- Updates content when selected ticket changes
- Shows "No ticket selected" when list is empty
- Dynamic height/width: fills 65% width, full height minus status bar
- **Test**: detail_pane_test.go
  - RenderDetail() with full ticket data
  - RenderDetail() with empty content
  - RenderDetail() with nil ticket shows placeholder

## Phase 4: Status Bar

### 4.1 Create `tui/statusbar.go` — bottom status bar
- Left side: filter indicator (ALL), board name, ticket counts by status (colored)
- Right side: context-aware keyboard shortcuts
- Adapts to terminal width
- **Test**: statusbar_test.go
  - Renders correct ticket counts
  - Renders correct keybind hints based on active pane

## Phase 5: Main App Rewrite

### 5.1 Rewrite `tui/app.go` — new split-pane architecture
- **Remove**: columns array, activeCol, MoveLeft/Right, viewDetail state, filterByStatus
- **Add**: listPane (table), detailPane (viewport), statusBar, focusedPane enum (list/detail)
- **States**: viewList (main split view), viewAdd, viewEdit, viewBoardSelect
- **Preserve constructor**: `NewApp(serverURL, token, workspace, board string) *App` — same signature
- **Preserve tea.Model**: Init(), Update(), View() — same interface
- Tab switches focus between list and detail pane
- j/k navigates list when list focused, scrolls viewport when detail focused
- / activates filter/search (future — stub the keybind)
- **Keep**: board selector, WebSocket listener, fetch/cycle/add/edit/delete commands, forms
- **View()**: lipgloss.JoinHorizontal for panes, JoinVertical to add status bar
- Window resize recalculates pane dimensions
- **Test**: app_test.go (rewrite)
  - NewApp creates list+detail panes
  - SetTickets populates list and auto-selects first ticket, detail shows it
  - Tab switches focused pane
  - j/k moves cursor in list, updates detail
  - Selection preserved after refresh

### 5.2 Delete old files
- Remove `tui/column.go` and `tui/column_test.go` (replaced by list_pane)
- Remove old `tui/ticket_view.go` (absorbed into detail_pane)

## Phase 6: Integration & Polish

### 6.1 Verify isolation
- `go build ./...` — full project compiles with zero changes outside `tui/`
- `go test ./...` — all tests pass including cmd/, server/, model/

### 6.2 Manual testing with live server
- `go build -o raptor . && ./raptor serve` in one terminal
- `./raptor` in another — verify split-pane layout
- Test all keybinds: j/k, Tab, n, e, m, d, b, r, q
- Test WebSocket real-time updates
- Test window resize

## Files Changed
| File | Action | Description |
|------|--------|-------------|
| `tui/styles.go` | NEW | Centralized color palette and styles |
| `tui/styles_test.go` | NEW | Style tests |
| `tui/age.go` | NEW | Age formatting utility |
| `tui/age_test.go` | NEW | Age formatting tests |
| `tui/list_pane.go` | NEW | Table-based ticket list component |
| `tui/list_pane_test.go` | NEW | List pane tests |
| `tui/detail_pane.go` | NEW | Viewport-based detail component |
| `tui/detail_pane_test.go` | NEW | Detail pane tests |
| `tui/statusbar.go` | NEW | Status bar component |
| `tui/statusbar_test.go` | NEW | Status bar tests |
| `tui/app.go` | REWRITE | Split-pane architecture |
| `tui/app_test.go` | REWRITE | New architecture tests |
| `tui/column.go` | DELETE | Replaced by list_pane |
| `tui/column_test.go` | DELETE | Replaced by list_pane tests |
| `tui/ticket_view.go` | DELETE | Absorbed into detail_pane |
| `tui/forms.go` | KEEP | No changes needed |

**Zero changes outside `tui/`** — cmd/, server/, model/, client/ are untouched.

## Dependencies
- Already have: bubbletea, bubbles, lipgloss, glamour, huh
- bubbles/table and bubbles/viewport are in the existing bubbles dependency

## Approach
- TDD: write failing tests first for each component, then implement
- Build bottom-up: styles → age → list_pane → detail_pane → statusbar → app
- Each phase should compile and pass tests before moving to the next
- Use /tdd skill for implementation

# Findings — Raptor TUI Rebuild

## Current TUI Architecture
- `tui/app.go` — Main Bubble Tea app, 3-column kanban board, 5 view states
- `tui/column.go` — Individual kanban column component (width: 30 chars)
- `tui/ticket_view.go` — Ticket detail view with Glamour markdown rendering
- `tui/forms.go` — Huh-based add/edit forms
- Server communication via HTTP client + WebSocket for real-time updates

## Data Model (model/ticket.go)
- Ticket: ID (8-char), Title, Content (markdown), Status, BoardID, CreatedBy, Assignee, AssignedBy, ClosedAt, CloseReason, CreatedAt, UpdatedAt
- Status enum: todo, in_progress, done, closed

## Current Dependencies (go.mod)
- bubbletea v1.3.10, bubbles v0.21.1, lipgloss v1.1.1, glamour v1.0.0, huh v1.0.0
- Already has bubbles — can use table and viewport components directly

## beads_viewer Inspiration
- Split-pane: list table (left) + detail viewport (right)
- Status bar at bottom with filter info, counts, keyboard hints
- Context-aware shortcuts, theme detection, selection preservation
- Key pattern: snapshot data, render only visible items, ID-based selection survives refreshes

## Design Decisions (from brainstorming)
- **Layout**: Split-pane, left 35-40%, right 60-65%
- **List columns**: STATUS | ID | ASSIGNEE | AGE | TITLE
- **Detail pane**: Title, metadata (status/ID/assignee/dates), markdown content via Glamour
- **Status bar**: Filter indicator, board name, ticket counts by status, keybind hints
- **Keybinds**: Keep j/k, n, e, m, d, b, r, q. Add Tab (pane focus), / (search/filter)
- **Approach**: TDD — tests first for each component

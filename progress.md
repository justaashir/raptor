# Progress Log — Raptor TUI Rebuild

## Session 1 — 2026-03-20

### Brainstorming Complete
- Explored current TUI architecture (3-column kanban)
- Studied beads_viewer for split-pane inspiration
- User chose: all columns (STATUS | ID | ASSIGNEE | AGE | TITLE)
- User chose: right pane bigger than left (35/65 split)
- Keybinds: keep existing + Tab for pane focus + / for search
- Visual mockup approved in browser companion

### Implementation Complete (TDD)
- Phase 1: `tui/styles.go` — StatusColor, pane/detail styles
- Phase 2.1: `tui/age.go` — FormatAge relative time utility
- Phase 2.2: `tui/list_pane.go` — bubbles/table with 5 columns, custom KeyMap, selection preservation
- Phase 3: `tui/detail_pane.go` — bubbles/viewport with Glamour markdown rendering
- Phase 4: `tui/statusbar.go` — ticket counts, filter badge, keybind hints
- Phase 5: `tui/app.go` — split-pane layout, Tab focus switching, deleted column.go/ticket_view.go
- Phase 6: `go build ./...` and `go test ./...` — all pass, zero changes outside tui/

### Stats
- 37 tests across 7 test files
- +1070 / -343 lines — all in tui/
- 6 TDD commits on feat/tui-rebuild branch

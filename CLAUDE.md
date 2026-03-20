## Raptor — Multiplayer Kanban Board

Go CLI/TUI kanban board with real-time sync via WebSocket.

### Build & Run

```sh
go build -o raptor .       # Build binary
go test ./...              # Run all tests
./raptor serve             # Start server on :8080
./raptor                   # Launch TUI (connects to server)
./raptor add "title"       # Add ticket via CLI
./raptor list              # List tickets
./raptor show <id>         # Show ticket details
./raptor move <id> <status> # Move ticket (todo, in_progress, done)
```

### Architecture

- `model/` — Ticket struct, Status enum, validation
- `server/` — HTTP REST API + WebSocket hub + SQLite persistence
- `tui/` — Bubble Tea TUI with 3-column board, Glamour markdown, Huh forms
- `cmd/` — Cobra CLI commands + HTTP client

### Dependencies

- Charm stack: bubbletea, lipgloss, glamour, huh
- cobra for CLI, nhooyr.io/websocket, modernc.org/sqlite (pure Go)

### Testing

TDD throughout. `go test ./...` runs all tests. In-memory SQLite for server tests.

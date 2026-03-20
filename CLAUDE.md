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

### Deployment

- **Server**: Railway at `https://raptor.raptorthree.com`
- **Volume**: mounted at `/data`, stores `raptor.db` + release binaries
- **Env vars on Railway**: `DATABASE_PATH=/data/raptor.db`, `VERSION=<current>`, `PORT` (set by Railway)
- Auto-deploys on push to `main`

### Releasing

```sh
scripts/release.sh           # bump minor: 0.1.0 → 0.2.0
scripts/release.sh patch     # bump patch: 0.2.0 → 0.2.1
scripts/release.sh major     # bump major: 0.2.1 → 1.0.0
```

One command does everything: bumps VERSION file, sets Railway env, cross-compiles, uploads binaries, commits, tags, creates GitHub release.

### Testing

TDD throughout. `go test ./...` runs all tests. In-memory SQLite for server tests.

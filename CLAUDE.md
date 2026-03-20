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

1. Bump version in `VERSION` file
2. Update `VERSION` env var on Railway: `railway variables set VERSION=x.y.z`
3. Run `scripts/release.sh` — reads version from `VERSION` file, cross-compiles, uploads to Railway + creates GitHub release
4. Users get update prompt on next CLI invocation, run `raptor update` to self-update

```sh
# Full release flow
echo "0.2.0" > VERSION
railway variables set VERSION=0.2.0
scripts/release.sh
```

### Testing

TDD throughout. `go test ./...` runs all tests. In-memory SQLite for server tests.

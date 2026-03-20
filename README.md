# Raptor

A multiplayer CLI kanban board with real-time sync.

## Quick Start

```sh
# Build
go build -o raptor .

# Start server
./raptor serve

# In another terminal — launch TUI
./raptor

# Or use CLI commands
./raptor add "My first task" --content "# Details here"
./raptor list
./raptor move <id> in_progress
```

## TUI Keys

| Key | Action |
|-----|--------|
| h/l, ←/→ | Switch column |
| j/k, ↑/↓ | Navigate tickets |
| Enter | View ticket detail |
| n | New ticket |
| m | Move ticket (cycle status) |
| e | Edit ticket |
| d | Delete ticket |
| r | Refresh |
| q | Quit |

## Architecture

Go server with SQLite persistence, WebSocket for real-time broadcast, Charm TUI.

Two terminals running `raptor` will see each other's changes in real-time.

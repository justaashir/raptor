# Raptor

A multiplayer CLI kanban board with real-time sync.

<img src="https://skillicons.dev/icons?i=go" alt="Go" height="40">

## Install

```bash
curl -fsSL https://raptor.raptorthree.com/install.sh | sh
```

Or build from source:

```bash
git clone https://github.com/justaashir/raptor && cd raptor
go build -o raptor .
```

## Quick Start

```bash
# Start the server
RAPTOR_SECRET=mysecret raptor serve

# Login (uses GitHub identity via `gh`)
raptor login

# Set up a workspace and board
raptor workspace create myteam
raptor board create sprint-1

# Add tickets
raptor add "Fix login bug" -c "Users see a 500 on /auth" -a alice

# View the board
raptor list
raptor list --status todo --mine

# Show a ticket with rendered markdown
raptor show abc12345

# Move tickets through the pipeline
raptor move abc12345 in_progress
raptor move abc12345 done

# Search
raptor search "login"
```

## Commands

| Command | Description |
|---------|-------------|
| `list` (alias: `ls`) | List tickets with filters |
| `add` | Create a ticket |
| `show` | View ticket with markdown rendering |
| `move` | Change ticket status |
| `edit` | Update title, content, or assignee |
| `search` | Full-text search |
| `rm` | Delete a ticket |
| `stats` | Ticket counts by status |
| `workspace` | Manage workspaces |
| `board` | Manage boards |
| `doctor` | Check connectivity and config |

Every command supports `--json` for machine-readable output.

## Self-Hosted

```bash
RAPTOR_SECRET=changeme RAPTOR_USERS=alice,bob raptor serve
```

Set `DATABASE_PATH` to persist data. Defaults to `./raptor.db` (SQLite).

## License

MIT

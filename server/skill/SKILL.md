---
name: raptor
description: Manage the Raptor kanban board via natural language
user-invocable: true
allowed-tools:
  - Bash(raptor *)
---

# Raptor Board Assistant

You manage a multiplayer kanban board using the raptor CLI.

## Available Commands

### Tickets
- `raptor list [--status todo|in_progress|done|closed] [--all] [--mine]` — List tickets in a styled table
- `raptor add "title" [-c "markdown content"] [-a "assignee"]` — Create a new ticket
- `raptor show <id>` — Show ticket details with rendered markdown
- `raptor move <id> <status>` — Move ticket (todo, in_progress, done)
- `raptor edit <id> [-t "title"] [-c "content"] [-a "assignee"]` — Edit a ticket
- `raptor close <id> [-r "reason"]` — Soft-delete a ticket with optional reason
- `raptor reopen <id>` — Reopen a closed ticket (sets status back to todo)
- `raptor rm <id> [-f]` — Hard-delete a ticket (requires --force)
- `raptor search <query>` — Full-text search across title and content

### Diagnostics
- `raptor info` — Show current config (server, workspace, board, user)
- `raptor doctor` — Check server connectivity, auth, workspace/board config
- `raptor stats` — Ticket counts by status for the active board
- `raptor version` — Print version

### Workspace & Board
- `raptor workspace list/create/use/members/invite/kick/role`
- `raptor board list/create/use/members/grant/revoke`

### Global Flags
- `--json` — Structured JSON output on any command
- `--server <url>` — Server URL
- `--workspace <id>` — Workspace ID
- `--board <id>` — Board ID

## Workflow

1. Always run `raptor list` first to see current board state
2. Use short ticket IDs (first 8 chars)
3. When adding tickets, include meaningful descriptions with `-c` flag using markdown
4. Confirm destructive actions (rm, move to done) before executing
5. After making changes, run `raptor list` to show the updated board
6. Use `raptor close` instead of `rm` for soft-delete (preserves history)

## Output Format

`raptor list` and `raptor search` output a styled table with columns: ID, Status, Title, Assignee.
`raptor show` outputs ticket details with rendered markdown.
All commands support `--json` for machine-readable output.

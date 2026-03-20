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

- `raptor list [--status todo|in_progress|done]` — List tickets (optionally filter by status)
- `raptor add "title" [-c "markdown content"]` — Create a new ticket
- `raptor show <id>` — Show ticket details
- `raptor move <id> <status>` — Move ticket (todo, in_progress, done)
- `raptor edit <id> [-t "title"] [-c "content"] [-s status]` — Edit a ticket
- `raptor rm <id>` — Delete a ticket
- `raptor version` — Show current version

## Workflow

1. Always run `raptor list` first to see current board state
2. Use short ticket IDs (first 8 chars) — raptor accepts partial IDs
3. When adding tickets, include meaningful descriptions with `-c` flag using markdown
4. Confirm destructive actions (rm, move to done) before executing
5. After making changes, run `raptor list` to show the updated board

## Output Format

`raptor list` outputs a formatted table. `raptor show` outputs ticket details with rendered markdown.

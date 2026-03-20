# Findings

## Research: beads_rust (br) vs Raptor

**Date:** 2026-03-20

### br features Raptor lacks
- Priorities (0-4 scale), labels, issue types, comments, dependencies
- Smart queries: `ready`, `blocked`, `stale`, `search`, `count`, `stats`
- `--json` output on all commands
- `close --reason` / `reopen` workflow
- Quick capture (`q` command)
- Diagnostics: `doctor`, `info`
- Defer/undefer, epics, saved queries, changelog, lint, orphans

### What we're adding (scoped down)
1. Soft-delete via `close --reason` + `reopen` (not full br priority/label/dep system)
2. `search` command
3. `--json` global flag
4. `doctor`, `info`, `stats` diagnostics

### Key architectural difference
- br is local-first (SQLite + JSONL, git-synced, no server)
- Raptor is sync-first (server + WebSocket, real-time multiplayer)
- All new features go through Raptor's existing client-server API pattern

### Current Raptor CLI commands (14)
add, list, show, move, edit, rm, login, serve, version, update, workspace, board, completion, help

### br CLI commands (~35)
create, q, show, list, search, update, close, reopen, delete, comments, label, dep, ready, blocked, stale, count, stats, defer, undefer, epic, graph, query, changelog, lint, orphans, sync, config, doctor, info, where, version, upgrade, agents, audit, schema

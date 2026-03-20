# Raptor Feature Plan

> Features inspired by beads_rust (br), adapted for Raptor's sync-first architecture.

## Phases

### Phase 1: Soft-Delete & Close with Comment
- [ ] Add `closed` status to Status enum in `model/ticket.go`
- [ ] Add `ClosedAt` and `CloseReason` fields to Ticket model
- [ ] Run DB migration for new columns
- [ ] Add `close` CLI command with `--reason` flag
- [ ] Add `reopen` CLI command
- [ ] Update `list` to hide closed tickets by default, add `--all` flag
- [ ] Update `PATCH /tickets/{id}` API to accept `close_reason`
- [ ] Update TUI to hide closed tickets
- [ ] Write tests for all of the above

### Phase 2: Search
- [ ] Add `search` CLI command with positional `<query>` arg
- [ ] Add `q` query param to list endpoint for server-side search
- [ ] SQLite case-insensitive search across title and content
- [ ] Support `--status` filter on search
- [ ] Write tests

### Phase 3: JSON Output
- [ ] Add `--json` global flag to root command
- [ ] Create JSON output helper (prints struct as JSON, errors as `{"error": "..."}`)
- [ ] Wire `--json` into `list`, `show`, `search`
- [ ] Wire `--json` into `add`, `edit`, `move`, `close`, `reopen`
- [ ] Wire `--json` into `rm` (returns `{"deleted": "<id>"}`)
- [ ] Wire `--json` into `workspace` and `board` subcommands
- [ ] Wire `--json` into `version`
- [ ] Write tests

### Phase 4: Diagnostics
- [ ] Add `doctor` command — check server connectivity, auth, versions
- [ ] Add `info` command — show current config (server, workspace, board, user)
- [ ] Add `stats` command — ticket counts by status for active board
- [ ] Write tests

## Decisions
| Decision | Choice | Reason |
|----------|--------|--------|
| Keep `rm` as hard-delete | Yes | `close` handles soft-delete; `rm` is the escape hatch |
| `closed` is a 4th status | Yes | Simpler than a separate `deleted_at` pattern |
| Search via SQLite LIKE | Yes | No need for FTS5 at current scale |
| JSON output as global flag | Yes | Consistent with br's `--json` pattern |
| Diagnostics are client-side | Mostly | `doctor` hits server, `info` reads local config, `stats` can use list endpoint |

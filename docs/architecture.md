# Architecture

## Overview

tasklin is a single-binary CLI/TUI application written in Go. It has no server, no database, and no network dependency. All state is persisted as plain YAML files on the local filesystem.

The binary is structured as a thin `cobra` CLI wrapper around a self-contained Bubble Tea TUI. Every screen, mutation, and persistence call lives in `internal/`.

---

## Component diagram

```
┌───────────────────────────────────────────────────────────┐
│                        tasklin binary                     │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐  │
│  │                     cmd/ (cobra)                    │  │
│  │                                                     │  │
│  │   root.go ──────────────────────────────────────┐   │  │
│  │   init.go   (tasklin init)                      │   │  │
│  │   transition.go  (tasklin _transition)          │   │  │
│  └─────────────────────────────────────────────────│───┘  │
│                                                    │      │
│  ┌─────────────────────────────────────────────────▼───┐  │
│  │                  internal/tui                       │  │
│  │                                                     │  │
│  │   Model (Bubble Tea)                                │  │
│  │   ├── viewBoard      ── handleBoard                 │  │
│  │   ├── viewDetail     ── handleDetail                │  │
│  │   ├── viewMove       ── handleMove                  │  │
│  │   ├── viewNew/Edit   ── handleInput                 │  │
│  │   ├── viewConfig     ── handleConfig                │  │
│  │   ├── viewStatuses   ── handleStatuses              │  │
│  │   └── viewHelp                                      │  │
│  └──────┬───────────────────────────────┬──────────────┘  │
│         │                               │                 │
│  ┌──────▼──────┐   ┌────────────┐  ┌────▼──────────────┐  │
│  │internal/    │   │internal/   │  │internal/          │  │
│  │store        │   │model       │  │git                │  │
│  │             │   │            │  │                   │  │
│  │ReadTickets  │   │Ticket      │  │RepoRoot           │  │
│  │WriteTicket  │   │Status      │  │CurrentBranch      │  │
│  │ReadConfig   │   │Config      │  └───────────────────┘  │
│  │WriteConfig  │   └────────────┘                         │
│  │NewID        │                  ┌────────────────────┐  │
│  │MigrateIfNeeded                 │internal/hooks      │  │
│  └──────┬──────┘                  │                    │  │
│         │                         │InstallCommitMsg    │  │
│         │                         │InstallPostMerge    │  │
│  ┌──────▼────────────────┐        │InstallPreCommit    │  │
│  │ .todo/ (local YAML)   │        │ReinstallIfPresent  │  │
│  │                       │        └────────────────────┘  │
│  │  config.yaml          │                                │
│  │  tickets/<id>.yaml    │                                │
│  │  deleted/<id>.yaml    │                                │
│  └───────────────────────┘                               │
└───────────────────────────────────────────────────────────┘
```

---

## Package responsibilities

### `cmd/`

| File | Responsibility |
|---|---|
| `root.go` | Cobra root command; detects if `.todo/` exists; runs migration; calls `tui.Run()` |
| `init.go` | `tasklin init` interactive setup wizard; calls `store.Init()` and `hooks` package |
| `add.go` | `tasklin add <title>` — creates a ticket from the CLI; supports `--label` and `--status` flags |
| `move.go` | `tasklin move <id> <status>` — moves a ticket to a new status; no-op if already there |
| `delete.go` | `tasklin delete <id>` — removes a ticket from tickets.yaml and archives it to deleted.yaml |
| `update.go` | `tasklin update <id>` — updates title (`--title`) and/or labels (`--add-label`, `--remove-label`); prints a change summary |
| `show.go` | `tasklin show <id>` — displays ticket status, title, and labels; `--verbose` adds full transition history |
| `transition.go` | `tasklin _transition <id> <status>` — internal command used by git hooks only |

### `internal/model/`

Pure data types with no logic beyond defaults. Nothing in this package reads or writes files.

- `Ticket` — id (string), title, status, created_at, transitions
- `Status` — id, name, color, order
- `Config` — title_limit, default_done_status, auto_commit_on_done, statuses
- `Transition` — from, to, at
- `DefaultStatuses()` / `DefaultConfig()` — sensible built-in values

### `internal/store/`

All YAML persistence. The TUI and CLI never touch the filesystem directly.

- `Store.ReadTickets()` — reads all `*.yaml` from `tickets/` directory
- `Store.WriteTicket(t)` — writes a single ticket to `tickets/<id>.yaml`
- `Store.DeleteTicketFile(id)` — removes `tickets/<id>.yaml`
- `Store.WriteDeletedTicket(t)` — writes a ticket to `deleted/<id>.yaml`
- `Store.ReadDeleted()` — reads all `*.yaml` from `deleted/` directory
- `Store.ReadConfig()` / `WriteConfig()` — project config
- `NewID()` — generates a random 8-char hex ID via `crypto/rand`
- `SortedStatuses()` — returns statuses ordered by their `Order` field
- `Store.MigrateIfNeeded()` — converts legacy single-file format to per-file on startup

### `internal/git/`

Thin wrappers around `git` shell calls. No state.

- `RepoRoot(dir)` — walks up the directory tree to find `.git/`
- `CurrentBranch(dir)` — runs `git rev-parse --abbrev-ref HEAD`
- `GitDir(root)` — returns path to `.git/` directory

### `internal/hooks/`

Generates the text content of git hook scripts. `cmd/init.go` installs them; `cmd/root.go` calls `ReinstallIfPresent` after migration.

- `InstallCommitMsg(gitDir, status)` — hook that transitions ticket on commit
- `InstallPostMerge(gitDir, status)` — hook that transitions ticket on merge
- `InstallPreCommit(gitDir)` — hook that stages `.todo/`
- `ReinstallIfPresent(gitDir, status)` — updates existing tasklin hooks to current format

### `internal/tui/`

The entire TUI lives in a single file: `tui.go`. It follows the standard Bubble Tea pattern.

- `Model` — all view state
- `Init()` — returns nil (no startup commands)
- `Update(msg)` — dispatches to per-mode `handle*` methods
- `View()` — dispatches to per-mode `view*` methods
- Data mutations call targeted store methods (`WriteTicket`, `DeleteTicketFile`, `WriteDeletedTicket`)

---

## Data flow

### Startup

```
main()
  └── cmd.Execute()
        └── root.go: store.New() → store.Initialised()?
              ├── No  → cmd/init.go: interactive init wizard
              └── Yes → store.MigrateIfNeeded()
                          └── (converts tickets.yaml → tickets/ if needed)
                        tui.New(store, projectDir)
                          ├── store.ReadTickets()
                          └── git.CurrentBranch()
                        tea.NewProgram(model).Run()
```

### Ticket mutation (e.g. move)

```
keypress (Shift+→)
  └── handleBoard()
        └── m.moveSelected(targetStatus)
              ├── finds ticket in m.tickets[]
              ├── appends Transition{from, to, at: now}
              ├── updates ticket.Status
              └── store.WriteTicket(ticket)   ← single file write
```

### Auto-commit flow

```
ticket moved to DefaultDoneStatus
  └── scheduleCommit(ticket, status)
        ├── m.committing = true   (shows amber footer banner)
        └── tea.Tick(1.2s) → commitReadyMsg
              └── autoCommitCmd()
                    └── tea.ExecProcess(bash script)
                          ├── prompt: stage new (untracked) files?
                          ├── prompt: stage deleted files?
                          ├── git add -p  (interactive patch)
                          └── git commit -m "[ID] Title"
```

### Parallel agent safety

Each ticket is stored as an independent file (`tickets/<id>.yaml`). Two agents working on different tickets modify different files — git merges cleanly with no conflicts. IDs are random hex strings from `crypto/rand`, so agents on different machines cannot produce colliding IDs.

---

## Key design decisions

| Decision | Rationale |
|---|---|
| Single binary, no daemon | Easy to install, version, and distribute |
| YAML over SQLite/JSON | Human-readable, diffable, committable alongside code |
| One file per ticket | Parallel agents on different tickets produce no merge conflicts |
| Random hex IDs | Collision-free across machines without coordination |
| All TUI in one file | Reduces navigation overhead for a tightly coupled UI |
| Value receivers on `Model` (Bubble Tea convention) | Bubble Tea requires `Update` to return a new model; pointer receivers are used only for multi-step mutations |
| `store` is the only persistence layer | Keeps the TUI testable without hitting disk |
| `bash` (not `sh`) for auto-commit script | Script uses `< <(...)` process substitution, which is bash-only |
| Soft delete to `deleted/` | Preserves history; allows recovery |

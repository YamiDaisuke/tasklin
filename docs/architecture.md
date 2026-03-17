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
│  │WriteTickets │   │Status      │  │CurrentBranch      │  │
│  │ReadConfig   │   │Config      │  │IsMainBranch       │  │
│  │WriteConfig  │   │GlobalState │  └───────────────────┘  │
│  │NextID       │   └────────────┘                         │
│  │BranchState  │                  ┌────────────────────┐  │
│  └──────┬──────┘                  │internal/hooks      │  │
│         │                         │                    │  │
│         │                         │WritePostCommit     │  │
│  ┌──────▼────────────────┐        │WritePostMerge      │  │
│  │ .todo/ (local YAML)   │        │WritePreCommit      │  │
│  │                       │        └────────────────────┘  │
│  │  config.yaml          │                                │
│  │  tickets.yaml         │  ┌──────────────────────────┐  │
│  │  deleted.yaml         │  │ ~/.config/tasklin/       │  │
│  └───────────────────────┘  │   state.yaml             │  │
│                             └──────────────────────────┘  │
└───────────────────────────────────────────────────────────┘
```

---

## Package responsibilities

### `cmd/`

| File | Responsibility |
|---|---|
| `root.go` | Cobra root command; detects if `.todo/` exists; calls `tui.Run()` |
| `init.go` | `tasklin init` interactive setup wizard; calls `store.Init()` and `hooks` package |
| `transition.go` | `tasklin _transition <id> <status>` — internal command used by git hooks only |

### `internal/model/`

Pure data types with no logic beyond defaults. Nothing in this package reads or writes files.

- `Ticket` — id, title, status, created_at, transitions
- `Status` — id, name, color, order
- `Config` — title_limit, default_done_status, auto_commit_on_done, statuses
- `Transition` — from, to, at
- `GlobalState` / `BranchTicket` — branch-level status overrides
- `DefaultStatuses()` / `DefaultConfig()` — sensible built-in values

### `internal/store/`

All YAML persistence. The TUI and CLI never touch the filesystem directly.

- `Store.ReadTickets()` / `WriteTickets()` — active tickets
- `Store.ReadDeleted()` / `WriteDeleted()` — soft-deleted tickets
- `Store.ReadConfig()` / `WriteConfig()` — project config
- `Store.NextID()` — reads both tickets and deleted to guarantee no ID reuse
- `SortedStatuses()` — returns statuses ordered by their `Order` field
- `ReadGlobalState()` / `WriteGlobalState()` — `~/.config/tasklin/state.yaml`
- `GetBranchOverrides()` / `ApplyBranchOverrides()` / `SetBranchOverride()` — branch-state helpers

### `internal/git/`

Thin wrappers around `git` shell calls. No state.

- `RepoRoot(dir)` — walks up the directory tree to find `.git/`
- `CurrentBranch(dir)` — runs `git rev-parse --abbrev-ref HEAD`
- `IsMainBranch(branch)` — returns true for `main` / `master`

### `internal/hooks/`

Generates the text content of git hook scripts. Does not write them to disk itself — `cmd/init.go` does that.

- `PostCommitHook(binary, status)` — script that transitions ticket on commit
- `PostMergeHook(binary, status)` — script that transitions ticket on merge
- `PreCommitHook()` — script that stages `.todo/`

### `internal/tui/`

The entire TUI lives in a single file: `tui.go`. It follows the standard Bubble Tea pattern.

- `Model` — all view state
- `Init()` — returns nil (no startup commands)
- `Update(msg)` — dispatches to per-mode `handle*` methods
- `View()` — dispatches to per-mode `view*` methods
- Data mutations go through `m.persist()` to `store.WriteTickets()`

---

## Data flow

### Startup

```
main()
  └── cmd.Execute()
        └── root.go: store.New() → store.Initialised()?
              ├── No  → cmd/init.go: interactive init wizard
              └── Yes → tui.New(store, projectDir)
                          ├── store.ReadConfig()
                          ├── store.ReadTickets()
                          ├── git.CurrentBranch()
                          └── store.ApplyBranchOverrides()  (if non-main branch)
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
              └── m.persist()
                    └── store.WriteTickets(m.tickets)
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

### Branch state tracking

```
TUI startup (non-main branch)
  └── store.ReadGlobalState()
        └── store.ApplyBranchOverrides(tickets, overrides)
              └── overrides shadow ticket.Status in memory only

ticket moved on non-main branch
  └── store.SetBranchOverride(gs, projectDir, branch, ticketID, newStatus)
        └── store.WriteGlobalState(gs)
              → ~/.config/tasklin/state.yaml updated
              (tickets.yaml is NOT modified)
```

---

## Key design decisions

| Decision | Rationale |
|---|---|
| Single binary, no daemon | Easy to install, version, and distribute |
| YAML over SQLite/JSON | Human-readable, diffable, committable alongside code |
| All TUI in one file | Reduces navigation overhead for a tightly coupled UI |
| Value receivers on `Model` (Bubble Tea convention) | Bubble Tea requires `Update` to return a new model; pointer receivers are used only for multi-step mutations |
| `store` is the only persistence layer | Keeps the TUI testable without hitting disk |
| `bash` (not `sh`) for auto-commit script | Script uses `< <(...)` process substitution, which is bash-only |
| Soft delete to `deleted.yaml` | Prevents ID reuse; allows recovery |
| Global state in `~/.config/tasklin/` | Branch overrides are user-scoped, not project-scoped |

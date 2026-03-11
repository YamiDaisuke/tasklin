# CLAUDE.md — tasklin

Guidelines and context for Claude Code when working on this repository.

## Project overview

`tasklin` is a keyboard-driven CLI/TUI kanban board for personal project backlogs. Data is persisted as plain YAML inside a `.todo/` directory at the project root. The TUI is built with Bubble Tea and lipgloss.

- Module: `github.com/frankcruz/tasklin`
- Go version: 1.24+
- Entry point: `main.go` → `cmd/root.go` → `internal/tui`

## Build & run

```sh
make              # build into bin/tasklin
make run          # build + run in current directory
make run-sample   # build + run inside sample/ (1 000 pre-loaded tickets)
make test         # run all tests with race detector
make build-all    # cross-compile for all platforms
```

Never use `go build .` directly — always use `make build` so ldflags (version, commit, buildDate) are injected correctly.

## Architecture

```
cmd/
  root.go        — cobra entry point; opens the TUI
  init.go        — `tasklin init` interactive setup
  transition.go  — `tasklin _transition` called by git hooks
internal/
  model/         — pure data types: Ticket, Status, Config, GlobalState
  store/         — YAML read/write, NextID, SortedStatuses, branch-state helpers
  git/           — git root detection, current branch, IsMainBranch
  hooks/         — git hook file generation (post-commit, post-merge, pre-commit)
  tui/           — all TUI logic (single file: tui.go)
resources/
  gen-sample.sh  — generates sample/ with 1 000 tickets for manual testing
```

### TUI structure (`internal/tui/tui.go`)

The TUI uses the standard Bubble Tea `Model / Init / Update / View` pattern.

**View modes** (iota):
`viewBoard` → `viewDetail` → `viewMove` → `viewNew` → `viewEdit` → `viewHelp` → `viewConfig` → `viewConfigEdit` → `viewStatuses` → `viewStatusEdit`

Each mode has a dedicated `handle*` method and a `view*` method.

**Key model fields:**
- `colIdx` / `rowIdx` — focused column and ticket within that column
- `colScroll []int` — per-column scroll offsets (one entry per status)
- `committing bool` — true while waiting to hand off to git (shows amber banner)
- `inputBuf string` — shared text input buffer used across edit/new/config screens
- `cfgRowIdx` / `statusRowIdx` — focused row in config and status management screens

**Scroll implementation:**
- `ticketRows()` returns visible row count (`m.height - 6`)
- `clampScroll()` adjusts `colScroll[colIdx]` after any `rowIdx` change
- Scrollbars rendered as `╎` (track) / `┃` (thumb, status-coloured) on column right edge

**Auto-commit flow:**
1. Ticket moved to Done → `scheduleCommit()` sets `m.committing = true`, returns `tea.Tick(1.2s)`
2. `commitReadyMsg` fires → `autoCommitCmd()` returns `tea.ExecProcess` (suspends alt-screen)
3. bash script: prompt for untracked files → prompt for deleted files → `git add -p` → commit

**Styling conventions:**
- Amber accent: `lipgloss.Color("214")`
- Dark header/footer background: `lipgloss.Color("235")`
- Dim separators: `lipgloss.Color("238")`
- Dim text: `lipgloss.Color("240")`
- Body text: `lipgloss.Color("252")`
- Status colours resolved via `ansiColor()` helper (maps name strings to ANSI 256 codes)

## Key conventions

### Data layer
- All persistence goes through `internal/store` — never read/write YAML files directly from the TUI
- `store.NextID()` always reads both `tickets.yaml` and `deleted.yaml` to avoid ID reuse
- `store.SortedStatuses()` must be called whenever `m.cfg.Statuses` is mutated to keep `m.statuses` consistent
- After renaming a status, migrate all tickets referencing the old name before persisting

### TUI mutations
- Status mutations (`addStatus`, `deleteStatus`, etc.) must also reset `m.colScroll` to `make([]int, len(m.statuses))` to avoid stale offsets
- `clampScroll()` must be called after any change to `m.rowIdx` or `m.colIdx` on the board
- `m.persist()` saves the current ticket slice to `tickets.yaml`; call it after any ticket mutation

### Shell scripts in auto-commit
- Always use `bash -c` (not `sh -c`) — the script uses process substitution `< <(...)` which is bash-only
- Pass values with spaces via environment variables (`GIT_ROOT`, `COMMIT_MSG`), never via shell interpolation

### Testing
- Tests live alongside their packages: `internal/store/store_test.go`, `internal/hooks/hooks_test.go`, `internal/tui/tui_test.go`
- Run `make test` before committing; CI uses `make test-ci` (writes `coverage.out`)
- The `internal/tui` package has a small test file — keep it passing
- Add unit tests whenever a non-trivial behavior is introduced or fixed, even if not explicitly requested — prefer table-driven tests

### Documentation
- Update `README.md` whenever user-facing behavior, keyboard shortcuts, config fields, or build commands change
- Update any relevant files in `docs/` when architecture or features change
- Keep `CLAUDE.md` itself accurate — update it when conventions, file structure, or key patterns change

### Dependencies
- Avoid adding third-party dependencies; prefer the Go standard library
- If a dependency is genuinely necessary, verify it is already used in `go.mod` before reaching for a new one

## Development sample

`sample/` (gitignored) is a full test environment with 1 000 tickets and a real git history.

```sh
make sample          # generate (no-op if already exists)
make sample CLEAN=1  # wipe and regenerate
make run-sample      # build + ensure sample exists + launch TUI inside it
```

## What to avoid

- Do not add error handling for impossible cases — trust internal invariants
- Do not create new files unless strictly necessary; prefer editing existing ones
- Do not skip `clampScroll()` after `rowIdx`/`colIdx` changes — it will break scrolling
- Do not use `sh` to run the auto-commit script — use `bash`
- Do not commit the `sample/` directory — it is gitignored by design

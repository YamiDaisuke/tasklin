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
`viewBoard` → `viewDetail` → `viewMove` → `viewNew` → `viewEdit` → `viewHelp` → `viewConfig` → `viewConfigEdit` → `viewStatuses` → `viewStatusEdit` → `viewLabelEdit` → `viewFilter`

Each mode has a dedicated `handle*` method and a `view*` method.

**Key model fields:**
- `colIdx` / `rowIdx` — focused column and ticket within that column
- `colScroll []int` — per-column scroll offsets (one entry per status)
- `committing bool` — true while waiting to hand off to git (shows amber banner)
- `inputBuf string` — shared text input buffer used across edit/new/config/label screens
- `cfgRowIdx` / `statusRowIdx` — focused row in config and status management screens
- `knownLabels []string` — all labels seen across all tickets; persisted to `labels.yaml`
- `filterLabels []string` — active label filters (AND semantics); survives mode changes
- `labelSuggestions []string` — autocomplete candidates for the current `inputBuf` prefix
- `acIdx int` — index of the selected autocomplete suggestion (-1 = none)

**Scroll implementation:**
- `ticketRows()` returns visible row count (`m.height - 6`)
- `clampScroll()` adjusts `colScroll[colIdx]` after any `rowIdx` change
- Each ticket occupies multiple display rows (title wrap lines + up to 2 label chip rows + 1 separator), so `clampScroll` counts actual display rows rather than assuming 1 ticket = 1 row
- Scrollbars rendered as `╎` (track) / `┃` (thumb, status-coloured) on column right edge

**Label chip rendering:**
- `chipRows(labels, width)` distributes labels into at most 2 display rows given the column width
- `renderChipRow(labels, selected)` renders a row of `[label]` chips in cyan (`color 6`) or bright-cyan (`color 14`) when the ticket is selected
- The amber `▌` selection indicator spans all rows of the focused ticket (title lines and chip rows)

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
- `store.ReadLabels()` / `store.WriteLabels()` manage `.todo/labels.yaml`; call `updateKnownLabels()` (not `WriteLabels` directly) from the TUI so the in-memory slice stays consistent

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
- **Documentation must be updated in the same change as the code — no exceptions.** This applies even when the user does not explicitly ask for it.
- Update `README.md` whenever user-facing behavior, keyboard shortcuts, config fields, or build commands change
- Update `docs/ui-reference.md` whenever a TUI screen, keyboard shortcut, or visual behavior changes
- Update `docs/data-model.md` whenever a data structure, YAML schema, or persistence file changes
- Update `docs/architecture.md` whenever the component structure, data flow, or key patterns change
- Update `docs/developer-guide.md` whenever conventions, helper functions, or implementation patterns change
- Keep `CLAUDE.md` itself accurate — update it when conventions, file structure, view modes, model fields, or key patterns change

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

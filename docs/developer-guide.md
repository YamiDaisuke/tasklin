# Developer Guide

A practical guide for anyone adding features, fixing bugs, or debugging tasklin.

---

## Table of contents

- [Getting started](#getting-started)
- [Project conventions](#project-conventions)
- [How the TUI works](#how-the-tui-works)
- [Adding a new TUI screen](#adding-a-new-tui-screen)
- [Adding a new config field](#adding-a-new-config-field)
- [Adding a new ticket field](#adding-a-new-ticket-field)
- [Working with the store](#working-with-the-store)
- [Adding a git hook](#adding-a-git-hook)
- [Testing](#testing)
- [Debugging](#debugging)
- [Common pitfalls](#common-pitfalls)

---

## Getting started

```sh
git clone https://github.com/frankcruz/tasklin
cd tasklin
make build        # compile
make run-sample   # launch against 1 000 pre-loaded tickets
make test         # run all tests
```

The sample project is regenerated automatically on first run. Use `make sample CLEAN=1` to wipe and regenerate it.

---

## Project conventions

### No direct filesystem access from the TUI

The TUI (`internal/tui/tui.go`) never reads or writes files directly. All persistence goes through `internal/store`:

```go
// correct
m.persist()  // calls store.WriteTickets(m.tickets)

// wrong — don't do this
yaml.Marshal(m.tickets)
os.WriteFile(...)
```

### Always use `make build`, not `go build`

The Makefile injects version metadata via `-ldflags`. Using `go build .` directly produces a binary with no version info.

### Use `bash`, not `sh`, for shell scripts in the TUI

The auto-commit script uses process substitution `< <(...)` which is a bash-only feature. Always pass scripts to `exec.Command("bash", "-c", script)`.

### Call `clampScroll()` after any `rowIdx` or `colIdx` change

`clampScroll()` keeps `colScroll[colIdx]` in sync with the cursor position. Skipping it causes the board to display the wrong window of tickets.

### Call `SortedStatuses()` after mutating `m.cfg.Statuses`

`m.statuses` is a sorted copy of `m.cfg.Statuses`. After any status add, rename, reorder, or delete, call:

```go
m.statuses = store.SortedStatuses(m.cfg.Statuses)
```

### Reset `colScroll` when the number of statuses changes

```go
m.colScroll = make([]int, len(m.statuses))
```

This prevents out-of-bounds panics and stale scroll positions.

### Migrate ticket status strings when renaming a status

`Ticket.Status` stores the status name, not an ID. Rename requires iterating all tickets:

```go
for k := range m.tickets {
    if m.tickets[k].Status == oldName {
        m.tickets[k].Status = newName
    }
}
```

---

## How the TUI works

The TUI uses the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework, which follows the Elm architecture: `Model`, `Init`, `Update`, `View`.

### State machine

The TUI is driven by a `viewMode` iota. Every keypress flows through `Update` → `handleKey` → the handler for the current mode:

```
Update(tea.KeyMsg)
  └── handleKey()
        ├── viewNew / viewEdit   → handleInput()
        ├── viewMove             → handleMove()
        ├── viewDetail           → handleDetail()
        ├── viewHelp             → (inline: return to viewBoard)
        ├── viewConfig           → handleConfig()
        ├── viewConfigEdit       → handleConfig()
        ├── viewStatuses         → handleStatuses()
        ├── viewStatusEdit       → handleStatuses()
        └── viewBoard (default)  → handleBoard()
```

### View rendering

`View()` calls the appropriate `view*` method for the current mode:

```
View()
  ├── viewBoard      → viewBoard()
  ├── viewDetail     → viewDetail()
  ├── viewMove       → viewMove()
  ├── viewNew        → viewInputScreen("New ticket")
  ├── viewEdit       → viewInputScreen("Edit ticket")
  ├── viewHelp       → viewHelp()
  ├── viewConfig     → viewConfigScreen()
  ├── viewConfigEdit → viewConfigScreen()   (same view, editing state differs)
  ├── viewStatuses   → viewStatusesScreen()
  └── viewStatusEdit → viewStatusesScreen() (same view, editing state differs)
```

### Shared input buffer

`m.inputBuf` is a single string field used by all text-input modes (new ticket, edit ticket, config field edit, status name/color edit). Always clear it before entering any input mode:

```go
m.mode = viewNew
m.inputBuf = ""
```

---

## Adding a new TUI screen

Follow these steps to add a new screen (e.g. `viewSearch`):

### 1. Add the view mode constant

```go
// internal/tui/tui.go — viewMode iota
const (
    viewBoard viewMode = iota
    viewDetail
    // ... existing modes ...
    viewSearch   // ← add here
)
```

### 2. Add a keyboard trigger

In `handleBoard()`, add a case for the key that opens your screen:

```go
case "/":
    m.mode = viewSearch
    m.inputBuf = ""
```

### 3. Add a handler

```go
func (m Model) handleSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "esc", "q":
        m.mode = viewBoard
        m.inputBuf = ""
    case "enter":
        // act on m.inputBuf
    case "backspace":
        if len(m.inputBuf) > 0 {
            _, size := utf8.DecodeLastRuneInString(m.inputBuf)
            m.inputBuf = m.inputBuf[:len(m.inputBuf)-size]
        }
    default:
        if msg.Type == tea.KeyRunes {
            m.inputBuf += msg.String()
        }
    }
    return m, nil
}
```

### 4. Wire into `handleKey`

```go
case viewSearch:
    return m.handleSearch(msg)
```

### 5. Add a view function

```go
func (m Model) viewSearch() string {
    // use lipgloss for styling
    // return a string that fills m.width × m.height
}
```

### 6. Wire into `View`

```go
case viewSearch:
    return m.viewSearch()
```

### 7. Add to the help overlay

Update `viewHelp()` to include the new key binding.

### 8. Update README and docs

- Add the key to the keyboard shortcuts table in `README.md`
- Add an ASCII art mockup in `docs/ui-reference.md`

---

## Adding a new config field

Config fields are defined in `configFields`:

```go
var configFields = []cfgFieldDef{
    {"Auto-commit on Done",          "bool"},
    {"Default Done status",          "string"},
    {"Title limit (0 = unlimited)",  "int"},
    {"Manage statuses",              "statuses"},
    // ← add new field here
}
```

### 1. Add the field to `model.Config`

```go
// internal/model/model.go
type Config struct {
    // ... existing fields ...
    MyNewField string `yaml:"my_new_field"`
}
```

### 2. Add a default value

```go
func DefaultConfig() Config {
    return Config{
        // ... existing defaults ...
        MyNewField: "default-value",
    }
}
```

### 3. Add the field to `configFields` in `tui.go`

```go
{"My new field", "string"},  // or "bool" / "int"
```

### 4. Wire reading and writing in `handleConfig`

Find the switch on `configFields[m.cfgRowIdx].kind` and ensure your new type is handled. For `string`, `int`, and `bool`, the existing cases likely cover it. Verify the index lines up with your field's position in `configFields`.

### 5. Add to the `config.yaml` field reference table in `README.md` and `docs/data-model.md`

---

## Adding a new ticket field

### 1. Add the field to `model.Ticket`

```go
type Ticket struct {
    // ... existing fields ...
    Priority int `yaml:"priority,omitempty"`
}
```

### 2. Update the TUI where tickets are rendered

In `viewBoard()`, the title line is built as:

```go
title := truncate(fmt.Sprintf("[%d] %s", t.ID, t.Title), contentWidth-3)
```

Adjust this to include your field if appropriate.

### 3. Update `viewDetail()` to show the new field

### 4. Update the data model doc

Add the field to the YAML schema example in `docs/data-model.md`.

---

## Working with the store

### Reading

```go
tickets, err := s.ReadTickets()
cfg, err := s.ReadConfig()
deleted, err := s.ReadDeleted()
```

### Writing

```go
_ = s.WriteTickets(tickets)
_ = s.WriteConfig(cfg)
```

### Getting the next ID

```go
id, err := s.NextID()  // reads both tickets.yaml and deleted.yaml
```

### Branch state

```go
gs, err := store.ReadGlobalState()
overrides := store.GetBranchOverrides(gs, projectDir, branch)
tickets = store.ApplyBranchOverrides(tickets, overrides)

// after a move on a non-main branch:
store.SetBranchOverride(gs, projectDir, branch, ticketID, newStatus)
_ = store.WriteGlobalState(gs)
```

---

## Adding a git hook

Hook scripts are generated in `internal/hooks/hooks.go` and installed by `cmd/init.go`.

### 1. Add a generator function in `hooks.go`

```go
func MyNewHook(binaryPath string) string {
    return fmt.Sprintf(`#!/bin/sh
# my hook logic
%s _transition ...
`, binaryPath)
}
```

### 2. Write the file in `cmd/init.go`

```go
hookPath := filepath.Join(gitDir, "hooks", "my-hook-name")
if err := os.WriteFile(hookPath, []byte(hooks.MyNewHook(binaryPath)), 0755); err != nil {
    return err
}
```

### 3. Add a test in `internal/hooks/hooks_test.go`

---

## Testing

Tests live next to the packages they test:

```
internal/store/store_test.go
internal/hooks/hooks_test.go
internal/model/model_test.go
internal/tui/tui_test.go
```

Run all tests:

```sh
make test        # with race detector and coverage
make test-ci     # CI mode, writes coverage.out
```

### Writing a test

Prefer table-driven tests:

```go
func TestNextID(t *testing.T) {
    cases := []struct {
        name    string
        active  []model.Ticket
        deleted []model.Ticket
        want    int
    }{
        {"empty store", nil, nil, 1},
        {"active only", tickets(1, 2, 3), nil, 4},
        {"with deleted", tickets(1, 2), tickets(3, 5), 6},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            // ...
        })
    }
}
```

### TUI tests

The TUI tests (`tui_test.go`) create a `Model` directly and fire synthetic key messages:

```go
m, _ := tui.New(store, dir)
m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
// assert m.Mode() == viewNew, etc.
```

Use the exported accessor methods (`ColIdx()`, `RowIdx()`, `Mode()`) rather than accessing model fields directly from tests.

---

## Debugging

### Inspect the raw YAML

tasklin stores everything as human-readable YAML. When something looks wrong in the TUI, check the files directly:

```sh
cat .todo/tickets.yaml
cat .todo/config.yaml
cat ~/.config/tasklin/state.yaml
```

### Run against a clean sample

```sh
make sample CLEAN=1
make run-sample
```

This gives you 1 000 tickets across 4 statuses with a known-good initial state.

### Add temporary stderr logging

The TUI owns the terminal, so `fmt.Println` won't appear on screen. Write debug output to stderr or a log file:

```go
fmt.Fprintln(os.Stderr, "debug:", someValue)
// or
f, _ := os.OpenFile("/tmp/tasklin-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
fmt.Fprintln(f, "rowIdx:", m.RowIdx())
```

Then run:

```sh
make run 2>/tmp/tasklin-debug.log
tail -f /tmp/tasklin-debug.log
```

### Check the view mode

`Model.Mode()` (exported) returns the current `viewMode`. Useful for asserting TUI state in tests and for adding conditional debug output.

---

## Common pitfalls

| Pitfall | Symptom | Fix |
|---|---|---|
| Forgot `clampScroll()` after `rowIdx` change | Board scrolls to wrong position | Call `m.clampScroll()` after every cursor move |
| Forgot `SortedStatuses()` after mutating statuses | Columns out of order or stale | Call `m.statuses = store.SortedStatuses(m.cfg.Statuses)` |
| Forgot `colScroll` resize after status add/delete | Index out of range panic | Call `m.colScroll = make([]int, len(m.statuses))` |
| Using `sh` for the auto-commit script | `syntax error near unexpected token '<'` | Use `exec.Command("bash", "-c", script)` |
| Status rename without ticket migration | Tickets disappear from board | Iterate `m.tickets` and update `Status` strings before persisting |
| `go build .` instead of `make build` | Binary has no version/commit info | Always use `make build` |
| Mutating `m.cfg.Statuses` without saving | Config changes lost on restart | Call `m.store.WriteConfig(m.cfg)` after any config mutation |

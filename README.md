# tasklin

A lightweight, portable CLI tool for managing personal project backlogs. Data is stored in human-readable YAML inside a `.todo/` folder at the project root. Includes a keyboard-driven TUI kanban board.

## Requirements

- Go 1.24+

## Build

```sh
go build -o tasklin .
```

To install it to your `$GOPATH/bin` (so `tasklin` is available everywhere):

```sh
go install .
```

## Run without installing

During development you can run the app directly from source without a build step:

```sh
go run . [command]
```

Examples:

```sh
go run . init        # run the init flow
go run .             # open the TUI directly
go run . --help      # show help
```

`go run .` recompiles on every invocation, so any source change is picked up immediately.

## Test

```sh
go test ./...
```

## Try it out

### 1. Initialise a project backlog

Run `tasklin init` inside any directory (git repo or not):

```sh
cd /your/project
tasklin init
```

You will be prompted to:
- Keep or customise the default statuses (To Do / In Progress / Done)
- Optionally install git hooks that auto-transition tickets on commit/merge

This creates a `.todo/` folder with `config.yaml` and an empty `tickets.yaml`.

### 2. Open the TUI

Run `tasklin` with no arguments to open the kanban board:

```sh
tasklin
```

If `.todo/` does not exist yet, the init flow runs first automatically.

### 3. Keyboard shortcuts

| Key | Action |
|---|---|
| `←` / `→` or `h` / `l` | Move between columns |
| `↑` / `↓` or `k` / `j` | Move between tickets in a column |
| `Enter` | View ticket detail (full history) |
| `n` | Create a new ticket in the focused column |
| `e` | Edit the selected ticket's title |
| `m` | Move the selected ticket to another status |
| `d` | Delete the selected ticket (moved to `deleted.yaml`) |
| `?` | Show help overlay |
| `q` / `Ctrl+C` | Quit |

### 4. Git hooks (optional)

When you run `tasklin init` inside a git repository you are offered the option to install hooks:

- **post-commit** — if the commit message starts with `[ID]`, transitions that ticket to the done status
- **post-merge** — if the merged branch name contains `[ID]`, transitions that ticket
- **pre-commit** *(optional)* — automatically stages `.todo/` in every commit

### 5. Data files

All data lives in `.todo/` at the project root and is plain YAML — safe to commit.

```
.todo/
├── config.yaml    # statuses, title limit, default done status
├── tickets.yaml   # active tickets
└── deleted.yaml   # soft-deleted tickets (never permanently removed)
```

Global branch-state tracking (used when working on non-main branches) is stored at:

```
~/.config/tasklin/state.yaml
```

## Project structure

```
.
├── main.go
├── cmd/
│   ├── root.go        # entry point, opens TUI
│   ├── init.go        # `tasklin init` command
│   └── transition.go  # internal `tasklin _transition` (used by git hooks)
└── internal/
    ├── model/         # data types (Ticket, Status, Config, GlobalState)
    ├── store/         # YAML persistence and branch-state helpers
    ├── git/           # git root / branch detection
    ├── hooks/         # git hook file generation
    └── tui/           # Bubble Tea TUI (board, detail, move, input views)
```

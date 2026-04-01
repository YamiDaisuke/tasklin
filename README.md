# tasklin

A lightweight, portable CLI tool for managing personal project backlogs. Data is stored in human-readable YAML inside a `.todo/` folder at the project root. Includes a keyboard-driven TUI kanban board.

---

## Table of contents

- [Requirements](#requirements)
- [Installation](#installation)
- [Quick start](#quick-start)
- [Keyboard shortcuts](#keyboard-shortcuts)
- [Configuration](#configuration)
- [Git integration](#git-integration)
- [Building & development](#building--development)
- [Documentation](#documentation)
- [Project structure](#project-structure)

---

## Requirements

- Go 1.24+

---

## Installation

### From source

```sh
git clone https://github.com/frankcruz/tasklin
cd tasklin
make install   # installs to $GOBIN / $GOPATH/bin
```

### Build only

```sh
make build     # produces bin/tasklin
```

---

## Quick start

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

If `.todo/` does not exist yet, the init flow runs automatically.

---

## Keyboard shortcuts

### Board

| Key | Action |
|---|---|
| `←` / `→` or `h` / `l` | Move focus between columns |
| `↑` / `↓` or `k` / `j` | Move focus between tickets within a column |
| `Shift+←` / `Shift+→` | Move the selected ticket one column left / right |
| `Enter` | View ticket detail (full transition history) |
| `n` | Create a new ticket in the focused column |
| `e` | Edit the selected ticket's title |
| `l` | Edit labels on the selected ticket |
| `/` | Filter board by label |
| `m` | Open the move dialog to pick a target status |
| `d` | Delete the selected ticket (soft-deleted to `deleted.yaml`) |
| `c` | Open the config screen |
| `?` | Show help overlay |
| `q` / `Ctrl+C` | Quit |

### Move dialog (`m`)

| Key | Action |
|---|---|
| `↑` / `↓` or `k` / `j` | Select target status |
| `Enter` | Confirm move |
| `Esc` / `q` | Cancel |

### Config screen (`c`)

| Key | Action |
|---|---|
| `↑` / `↓` or `k` / `j` | Navigate fields |
| `Enter` / `Space` | Edit the focused field (or open status management) |
| `Esc` / `q` | Go back to the board |

### Status management (reachable from config)

| Key | Action |
|---|---|
| `↑` / `↓` or `k` / `j` | Navigate statuses |
| `Shift+↑` / `Shift+↓` | Reorder the focused status |
| `n` | Add a new status |
| `e` | Edit name and colour of the focused status |
| `d` | Delete the focused status (minimum 2 required) |
| `Esc` / `q` | Go back to config |

---

## Configuration

All data lives in `.todo/` at the project root and is plain YAML — safe to commit.

```
.todo/
├── config.yaml    # statuses, title limit, default done status, auto-commit flag
├── tickets.yaml   # active tickets
├── deleted.yaml   # soft-deleted tickets (never permanently removed)
└── labels.yaml    # index of all known labels (used for autocomplete)
```

### config.yaml fields

| Field | Type | Default | Description |
|---|---|---|---|
| `title_limit` | int | `0` | Max ticket title length (0 = unlimited) |
| `default_done_status` | string | `"Done"` | Status name treated as "done" for hooks and auto-commit |
| `auto_commit_on_done` | bool | `false` | Trigger interactive git commit when a ticket reaches done |
| `statuses` | list | To Do / In Progress / Done | Ordered list of status columns |

### Labels

Tickets can have zero or more labels. Labels follow identifier rules: must start with a letter, followed by any combination of letters, digits, and underscores (`[A-Za-z][A-Za-z0-9_]*`).

- Press `l` on any ticket to open the label editor
- Type a label name; `Tab` / `Shift+Tab` cycles through autocomplete suggestions drawn from previously used labels
- `Enter` adds the label; `Backspace` at an empty input removes the last label on the ticket
- Labels are displayed as `[chip]` rows below the ticket title in the board (up to 2 rows)
- Press `/` to open the label filter — add one or more labels; only tickets matching **all** active filters are shown
- Active filters are shown in the footer as `▼ label1 label2`
- `Ctrl+U` inside the filter screen clears all active filters

All known labels are persisted in `.todo/labels.yaml` to power autocomplete across sessions.

### Auto-commit on Done

When `auto_commit_on_done` is enabled, moving a ticket to the Done status triggers an interactive git commit flow:

1. Any **new (untracked) files** are listed — confirm each with `y/N`
2. Any **deleted files** are listed — confirm each with `y/N`
3. `git add -p` runs for interactive hunk selection on modified files
4. A commit is created with the message `[ID] Title` if anything was staged

Enable it from the in-app config screen (`c`) or by editing `.todo/config.yaml` directly.

### Global state

Branch-state tracking (used when working on non-main branches) is stored at:

```
~/.config/tasklin/state.yaml
```

---

## Git integration

When you run `tasklin init` inside a git repository you are offered the option to install hooks:

- **post-commit** — if the commit message starts with `[ID]`, transitions that ticket to the done status
- **post-merge** — if the merged branch name contains `[ID]`, transitions that ticket
- **pre-commit** *(optional)* — automatically stages `.todo/` in every commit

---

## Building & development

All common operations are covered by the [Makefile](Makefile). Run `make help` to list every available target.

### Targets

| Target | Description |
|---|---|
| `make` / `make build` | Compile for the current OS/ARCH into `bin/tasklin` |
| `make build-all` | Cross-compile for all platforms (see below) |
| `make run` | Build and launch tasklin in the current directory |
| `make run-sample` | Build and launch tasklin inside the generated sample project |
| `make sample` | Generate the `sample/` test project (skips if already present) |
| `make sample CLEAN=1` | Wipe and regenerate `sample/` from scratch |
| `make test` | Run all unit tests with the race detector and coverage |
| `make test-ci` | Run tests in CI mode, writing `coverage.out` |
| `make install` | Install `tasklin` to `$GOBIN` / `$GOPATH/bin` |
| `make clean` | Remove `bin/` and `coverage.out` |

### Cross-compilation targets

`make build-all` produces one binary per platform under `bin/`:

| Platform | Output |
|---|---|
| Linux amd64 | `bin/tasklin-linux-amd64` |
| Linux arm64 | `bin/tasklin-linux-arm64` |
| macOS amd64 | `bin/tasklin-darwin-amd64` |
| macOS arm64 | `bin/tasklin-darwin-arm64` |
| Windows amd64 | `bin/tasklin-windows-amd64.exe` |

Version metadata (`version`, `commit`, `buildDate`) is injected at link time via `-ldflags` and can be overridden:

```sh
make build VERSION=1.2.0 COMMIT=abc1234
```

### Development sample project

A script in [`resources/gen-sample.sh`](resources/gen-sample.sh) generates a self-contained test environment under `sample/` (gitignored):

```sh
make sample          # generate (skips if sample/ already exists)
make sample CLEAN=1  # wipe and regenerate from scratch
make run-sample      # build + generate + launch tasklin inside sample/
```

---

## Documentation

Detailed developer documentation lives in [`docs/`](docs/):

| Document | Description |
|---|---|
| [Architecture](docs/architecture.md) | System design, component diagram, data flow |
| [UI Reference](docs/ui-reference.md) | All TUI screens documented with ASCII art |
| [Data Model](docs/data-model.md) | Data structures, YAML schemas, UML class diagram |
| [Developer Guide](docs/developer-guide.md) | How to implement features, conventions, debugging |
| [Requirements](docs/REQUIREMENTS.md) | Original requirements and acceptance criteria |

---

## Project structure

```
.
├── main.go
├── cmd/
│   ├── root.go        # entry point, opens TUI
│   ├── init.go        # `tasklin init` command
│   └── transition.go  # internal `tasklin _transition` (used by git hooks)
├── internal/
│   ├── model/         # data types (Ticket, Status, Config, GlobalState)
│   ├── store/         # YAML persistence and branch-state helpers
│   ├── git/           # git root / branch detection
│   ├── hooks/         # git hook file generation
│   └── tui/           # Bubble Tea TUI (all screens in tui.go)
├── resources/
│   └── gen-sample.sh  # generates sample/ with 1 000 tickets for testing
└── docs/
    ├── architecture.md
    ├── ui-reference.md
    ├── data-model.md
    ├── developer-guide.md
    └── REQUIREMENTS.md
```

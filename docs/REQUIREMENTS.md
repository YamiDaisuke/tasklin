# Project Backlog CLI Tool — Requirements Document

## Overview

A lightweight, portable CLI tool written in **Go** for managing personal project backlogs. Data is stored in human-readable YAML format inside a `.todo/` folder at the project root. The tool features both a command-line interface and a terminal UI (TUI).

---

## Technology Stack

| Concern | Choice | Rationale |
|---|---|---|
| Language | Go | Lightweight, portable, single binary |
| Data format | YAML | Human-readable, easily portable |
| TUI library | [Bubble Tea](https://github.com/charmbracelet/bubbletea) | Idiomatic Go TUI framework |
| YAML parsing | `gopkg.in/yaml.v3` | Standard Go YAML library |
| CLI parsing | `cobra` | Standard Go CLI framework |

---

## Project Structure

```
<project-root>/
└── .todo/
    ├── config.yaml       # Project config: statuses, title limit, etc.
    ├── tickets.yaml      # Active tickets
    └── deleted.yaml      # Soft-deleted tickets (backup)

~/.config/todo/
└── state.yaml            # Global state: per-project branch/ticket tracking (tmp file)
```

---

## Data Models

### Status

```yaml
statuses:
  - id: 1
    name: "To Do"
    color: "red"      # ANSI color name or code
    order: 0
  - id: 2
    name: "In Progress"
    color: "yellow"
    order: 1
  - id: 3
    name: "Done"
    color: "green"
    order: 2
```

**Rules:**
- Minimum of **2 statuses** required.
- `order` determines column order in the TUI.
- Colors must be ANSI-compatible (named or escape code).

---

### Ticket

```yaml
tickets:
  - id: 1
    title: "Set up CI pipeline"
    status: "In Progress"
    created_at: "2025-01-15T10:30:00Z"
    transitions:
      - from: "To Do"
        to: "In Progress"
        at: "2025-01-16T09:00:00Z"
```

**Rules:**
- `id` is **auto-numeric**, globally unique per project, never reused.
- `title` has a configurable character limit (set in `config.yaml`; no limit by default).
- A ticket must **always be in exactly one status**.
- Transition history is append-only and immutable.
- Deleted tickets are moved to `deleted.yaml` — never permanently removed.

---

### Config

```yaml
title_limit: 120          # 0 = no limit
default_done_status: "Done"
```

---

### Global State File (`~/.config/todo/state.yaml`)

Tracks ticket statuses keyed by project path and git branch, ensuring consistency when switching branches.

```yaml
projects:
  /home/user/my-project:
    main:
      - ticket_id: 3
        status: "Done"
    feature/my-branch:
      - ticket_id: 5
        status: "In Progress"
```

---

## Commands

### `todo init`

Initialises the `.todo/` folder in the current directory.

**Flow:**
1. Check if `.todo/` already exists — if so, prompt: _"Already initialised. Re-initialise? [y/N]"_.
2. Display default statuses:
   - To Do — red — order 0
   - In Progress — yellow — order 1
   - Done — green — order 2
3. Prompt: _"Use default statuses? [Y/n]"_
   - If **yes**: write defaults.
   - If **no**: interactively collect statuses (name, color, order) until the user is done. Enforce minimum of 2.
4. **Git hook prompt** (if `.git/` is detected in the project root):
   - Prompt: _"Git repository detected. Add git hooks for automatic ticket transitions? [y/N]"_
   - If **yes**:
     - Ask which status to transition to on merge (default: `Done`).
     - Install hooks for:
       - `post-commit`: if commit message starts with `[TICKET-ID]`, transition that ticket to the chosen status.
       - `post-merge`: if the merged branch name contains `[TICKET-ID]`, transition that ticket.
     - Prompt: _"Also stage `.todo/` folder in all commits? [Y/n]"_ — if yes, add a `pre-commit` hook that runs `git add .todo/`.
5. Write `config.yaml` and an empty `tickets.yaml`.

---

### `todo` (no arguments)

- If `.todo/` does not exist → run the `init` flow first, then open the TUI.
- If `.todo/` exists → open the TUI directly.

---

## TUI — Terminal User Interface

Built with **Bubble Tea**. Fully keyboard-driven; mouse support is optional.

### Layout

```
╔══════════════════════════════════════════════════════╗
║  my-project backlog                    branch: main  ║
╠══════════════╦══════════════╦════════════════════════╣
║   TO DO (2)  ║ IN PROGRESS  ║       DONE (1)         ║
║──────────────║──────────────║────────────────────────║
║ [1] Set up   ║ [3] Write    ║ [2] Init repo          ║
║     CI       ║     tests    ║                        ║
║              ║              ║                        ║
╠══════════════╩══════════════╩════════════════════════╣
║ [n]ew  [d]elete  [m]ove  [e]dit  [c]onfig  [q]uit    ║
╚══════════════════════════════════════════════════════╝
```

**Columns:**
- One column per status, ordered by `status.order`.
- Column header shows status name and ticket count, rendered in the status's ANSI color.
- Tickets shown as `[ID] Title` (title truncated to column width if needed).

**Navigation:**

| Key | Action |
|---|---|
| `←` / `→` or `h` / `l` | Move between columns |
| `↑` / `↓` or `k` / `j` | Move between tickets in column |
| `Enter` | View ticket detail |
| `n` | New ticket (prompts for title) |
| `m` | Move selected ticket to another status |
| `e` | Edit selected ticket title |
| `d` | Delete selected ticket (with confirmation) |
| `c` | Open config menu |
| `?` | Show help overlay |
| `q` / `Ctrl+C` | Quit |

**Ticket Detail View:**
- Shows full title, ID, created date, current status, full transition history.
- Press `Esc` to go back.

**Command Bar:**
- Persistent footer showing available key bindings for current context.

---

## Git Hooks

### `post-commit`

```bash
#!/bin/sh
MSG=$(git log -1 --pretty=%B)
if echo "$MSG" | grep -qE '^\[([0-9]+)\]'; then
  TICKET_ID=$(echo "$MSG" | grep -oE '^\[([0-9]+)\]' | tr -d '[]')
  todo _transition $TICKET_ID "<done-status>"
fi
```

### `post-merge`

```bash
#!/bin/sh
BRANCH=$(git reflog | awk 'NR==1{print $6}' | sed 's/.*\///')
if echo "$BRANCH" | grep -qE '\[([0-9]+)\]'; then
  TICKET_ID=$(echo "$BRANCH" | grep -oE '\[([0-9]+)\]' | tr -d '[]')
  todo _transition $TICKET_ID "<done-status>"
fi
```

### `pre-commit` (optional)

```bash
#!/bin/sh
git add .todo/
```

> **Note:** `todo _transition` is an internal subcommand (prefixed with `_`) used only by hooks — not part of the public API.

---

## Branch State Tracking

- On **TUI launch**, the tool reads the current git branch (if applicable) and loads any override statuses from `~/.config/todo/state.yaml`.
- When a ticket is **moved** while on a non-main branch, that transition is recorded in global state (not written back to `tickets.yaml` until merged).
- On **merge to main**, tickets in global state are reconciled with `tickets.yaml`.

> **Open question:** Should branch-level overrides shadow the ticket's `status` field, or should they be stored as a separate `branch_status` field? Recommendation: shadow at runtime, write to global state only — keeping `tickets.yaml` clean.

---

## Error Handling & Edge Cases

- Running `init` inside a nested subfolder of a project that already has `.todo/` → warn user and ask if they want to init a new nested project or use the parent one.
- Title character limit enforcement is done at input time in both CLI prompts and the TUI.
- If `tickets.yaml` is malformed, display a clear error with the line/column and exit gracefully.
- Deleted ticket IDs are never reused — the next ID is always `max(all_ever_created_ids) + 1`.

---

## Future Considerations (Out of Scope for v1)

- Multi-line ticket descriptions / notes
- Due dates and priority fields
- Filtering and search in TUI
- Export to CSV / Markdown
- Sync with remote (GitHub Issues, Linear, etc.)
- Tags / labels

---

## Acceptance Criteria Summary

| # | Requirement | Priority |
|---|---|---|
| 1 | `todo init` creates `.todo/` with config and default statuses | Must |
| 2 | Init prompts for confirmation / custom statuses | Must |
| 3 | Minimum 2 statuses enforced | Must |
| 4 | Tickets have auto-numeric unique IDs | Must |
| 5 | Tickets always have a status | Must |
| 6 | Transition history is logged on every status change | Must |
| 7 | Deleted tickets saved to `deleted.yaml` | Must |
| 8 | TUI opens on `todo` with no arguments | Must |
| 9 | TUI is fully keyboard-navigable | Must |
| 10 | Git hook setup offered when `.git/` detected | Should |
| 11 | Hooks auto-transition tickets on commit/merge | Should |
| 12 | `.todo/` auto-staged in commits (optional hook) | Should |
| 13 | Global state file tracks branch-level ticket status | Should |
| 14 | `title_limit` configurable in `config.yaml` | Could |
| 15 | Mouse support in TUI | Could |

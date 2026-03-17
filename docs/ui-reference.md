# UI Reference

All screens in the tasklin TUI documented with ASCII art mockups and behavioral notes.

The TUI uses Bubble Tea with an alt-screen buffer. The terminal is never cleared between frames — Bubble Tea diffs and redraws only what changed.

---

## Board (default view)

The main screen. Three or more columns, one per status, rendered side by side.

```
╔╦╗╔═╗╔═╗╦╔═╦  ╦╔╗╔
 ║ ╠═╣╚═╗╠╩╗║  ║║║║
 ╩ ╩ ╩╚═╝╩ ╩╩═╝╩╝╚╝  ⎇ main
────────────────────────────────────────────────────────────
 TO DO (4)            IN PROGRESS (2)        DONE (3)
────────────────────────────────────────────────────────────
▌ [1] Set up CI       [4] Write unit tests    [2] Init repo
  [5] Add search      [7] Auth middleware     [3] Add models
  [6] Dark mode                               [9] Migrations
  [8] Export CSV
  n new  │  d del  │  m move  │  e edit  │  c config  │  q quit
```

**Layout rules:**
- Terminal width is divided evenly across columns; the last column absorbs any remainder
- Column headers show the status name (uppercase) and ticket count
- The focused ticket is marked with an amber `▌` indicator and bold white text
- All other tickets render in dim gray
- Tickets are sorted by ID (ascending) within each column
- A slim scrollbar appears on the right edge of any column that overflows:
  - `╎` = track (very dark)
  - `┃` = thumb (status color)

**Header:**
- 3-line ASCII art "TASKLIN" in amber (`color 214`)
- Branch name prefixed with `⎇` shown on the third line (dim gray)
- Dark background bar (`color 235`) spanning full width

**Footer:**
- Key hints separated by `│`
- Key names in amber, labels in dim gray
- Replaced by an amber `⎆ preparing commit — launching git add -p ...` banner while auto-commit is pending

---

## Ticket detail view (`Enter`)

Shows the full history of the selected ticket.

```
╔╦╗╔═╗╔═╗╦╔═╦  ╦╔╗╔
 ║ ╠═╣╚═╗╠╩╗║  ║║║║
 ╩ ╩ ╩╚═╝╩ ╩╩═╝╩╝╚╝  ⎇ main
────────────────────────────────────────────────────────────
  Ticket #7

  Title      Auth middleware
  Status     In Progress
  Created    2026-01-14 09:00

  History
  ──────────────────────────────
  2026-01-14 09:00   created in To Do
  2026-01-15 11:30   To Do → In Progress

  esc / q  go back
────────────────────────────────────────────────────────────
```

**Notes:**
- Transition history is append-only; every status change is recorded
- Press `Esc` or `q` to return to the board

---

## New ticket (`n`)

An inline text input rendered in place of the focused column's first row.

```
╔╦╗╔═╗╔═╗╦╔═╦  ╦╔╗╔
 ║ ╠═╣╚═╗╠╩╗║  ║║║║
 ╩ ╩ ╩╚═╝╩ ╩╩═╝╩╝╚╝  ⎇ main
────────────────────────────────────────────────────────────
 TO DO (4)            IN PROGRESS (2)        DONE (3)
────────────────────────────────────────────────────────────
  New ticket: Add OAuth support_
  [1] Set up CI       [4] Write unit tests    [2] Init repo
  [5] Add search      [7] Auth middleware     [3] Add models

  enter confirm  │  esc cancel
────────────────────────────────────────────────────────────
```

**Notes:**
- Ticket is created in the currently focused column's status
- If `title_limit` is set, characters beyond the limit are rejected at input time
- Pressing `Enter` with an empty buffer is a no-op

---

## Edit ticket title (`e`)

Same layout as new ticket, but pre-populated with the existing title.

```
  Edit: Auth middleware_
```

**Notes:**
- `Backspace` deletes the last character
- `Enter` confirms; `Esc` cancels without saving

---

## Move dialog (`m`)

A status picker overlaid in the board's footer area.

```
╔╦╗╔═╗╔═╗╦╔═╦  ╦╔╗╔
 ║ ╠═╣╚═╗╠╩╗║  ║║║║
 ╩ ╩ ╩╚═╝╩ ╩╩═╝╩╝╚╝  ⎇ main
────────────────────────────────────────────────────────────
  Move "[7] Auth middleware" to:

  ○ To Do
  ● In Progress         ← current / focused
  ○ Review
  ○ Done

  enter confirm  │  esc cancel
────────────────────────────────────────────────────────────
```

**Notes:**
- The current status of the ticket is highlighted
- Navigate with `↑` / `↓` or `k` / `j`
- `Enter` confirms; `Esc` / `q` cancels
- If moved to the `default_done_status` and `auto_commit_on_done` is enabled, the auto-commit flow begins after a 1.2-second delay

---

## Config screen (`c`)

Editable list of project settings.

```
╔╦╗╔═╗╔═╗╦╔═╦  ╦╔╗╔
 ║ ╠═╣╚═╗╠╩╗║  ║║║║
 ╩ ╩ ╩╚═╝╩ ╩╩═╝╩╝╚╝  ⎇ main
────────────────────────────────────────────────────────────
  Configuration

  ▌ Auto-commit on Done          false
    Default Done status          Done
    Title limit (0 = unlimited)  0
    Manage statuses              →

  enter / space  edit  │  esc / q  back
────────────────────────────────────────────────────────────
```

**Field types:**
- `bool` — `Enter`/`Space` toggles the value in place
- `string` / `int` — `Enter` opens an inline text editor; confirm with `Enter`, cancel with `Esc`
- `statuses` — `Enter` navigates to the status management screen

---

## Config field edit

When editing a string or int field an inline editor opens.

```
  Default Done status:  Done_
```

**Notes:**
- `Backspace` deletes the last character
- `Enter` saves; `Esc` cancels

---

## Status management (from config → Manage statuses)

Full CRUD for status columns.

```
╔╦╗╔═╗╔═╗╦╔═╦  ╦╔╗╔
 ║ ╠═╣╚═╗╠╩╗║  ║║║║
 ╩ ╩ ╩╚═╝╩ ╩╩═╝╩╝╚╝  ⎇ main
────────────────────────────────────────────────────────────
  Statuses

  ▌ To Do          red
    In Progress    yellow
    Review         blue
    Done           green

  n new  │  e edit  │  d delete  │  shift+↑↓ reorder  │  esc back
────────────────────────────────────────────────────────────
```

**Notes:**
- Minimum of 2 statuses enforced — `d` is a no-op when only 2 remain
- Reordering updates the `order` field on all affected statuses and saves immediately
- Renaming a status automatically migrates all tickets that reference the old name

---

## Status edit (name step)

Two-step editor: name first, then color.

```
  Status name:  In Progress_
```

---

## Status edit (color step)

```
  Color for "In Progress":  yellow_
  (red, green, yellow, blue, magenta, cyan, white)
```

**Supported color names:** `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`

---

## Help overlay (`?`)

```
╔╦╗╔═╗╔═╗╦╔═╦  ╦╔╗╔
 ║ ╠═╣╚═╗╠╩╗║  ║║║║
 ╩ ╩ ╩╚═╝╩ ╩╩═╝╩╝╚╝  ⎇ main
────────────────────────────────────────────────────────────
  Keyboard shortcuts

  Navigation
    ←/→  h/l       move between columns
    ↑/↓  k/j       move between tickets
    shift+←/→      move ticket to adjacent column

  Actions
    enter          view ticket detail
    n              new ticket
    e              edit ticket title
    m              move ticket (pick any status)
    d              delete ticket
    c              config
    ?              this screen
    q  ctrl+c      quit

  any key  close
────────────────────────────────────────────────────────────
```

---

## Auto-commit banner

Replaces the footer for ~1.2 seconds after a ticket is moved to Done with `auto_commit_on_done` enabled.

```
  ⎆  preparing commit — launching git add -p ...
```

Rendered in amber bold across the full footer width. After the delay, the TUI suspends and the terminal hands off to the interactive bash script.

---

## Scrollbars

When a column contains more tickets than fit in the visible area, a 1-character scrollbar appears on its right edge.

```
  [1] Set up CI       ╎
  [5] Add search      ┃   ← thumb (status color)
  [6] Dark mode       ┃
  [8] Export CSV      ╎
  [10] Refactor auth  ╎
```

- `╎` — track character, very dark (`color 236`)
- `┃` — thumb character, status color
- Thumb height is proportional to `(visible rows)² / total tickets`
- Thumb position tracks the scroll offset

---

## Color palette

| Usage | Color code | Appearance |
|---|---|---|
| Amber accent (title, keys, focused indicator) | `214` | Orange-amber |
| Dark background (header / footer bars) | `235` | Near-black gray |
| Column / section separators | `238` | Dark gray |
| Dim text (metadata, labels) | `240` | Medium gray |
| Body text (ticket titles) | `252` | Light gray |
| Focused ticket text | `15` | Bright white |
| Error messages | `9` | Bright red |
| Scrollbar track | `236` | Very dark gray |

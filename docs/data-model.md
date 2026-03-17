# Data Model

## Overview

All tasklin data is plain YAML. There is no database — the files are designed to be committed alongside your code.

```
<project-root>/
└── .todo/
    ├── config.yaml      ← project configuration and statuses
    ├── tickets.yaml     ← active tickets
    └── deleted.yaml     ← soft-deleted tickets (never purged)

~/.config/tasklin/
└── state.yaml           ← branch-level status overrides (global, user-scoped)
```

---

## UML class diagram

```
┌─────────────────────────────────────────────────────────────┐
│                          Config                             │
├─────────────────────────────────────────────────────────────┤
│ TitleLimit        int                                       │
│ DefaultDoneStatus string                                    │
│ AutoCommitOnDone  bool                                      │
│ Statuses          []Status                                  │
└───────────────────────────┬─────────────────────────────────┘
                            │ 1
                            │ contains 2..*
                            ▼
              ┌─────────────────────────┐
              │          Status         │
              ├─────────────────────────┤
              │ ID    int               │
              │ Name  string            │
              │ Color string            │
              │ Order int               │
              └─────────────────────────┘


┌─────────────────────────────────────────────────────────────┐
│                         TicketFile                          │
├─────────────────────────────────────────────────────────────┤
│ Tickets []Ticket                                            │
└───────────────────────────┬─────────────────────────────────┘
                            │ contains 0..*
                            ▼
              ┌──────────────────────────────┐
              │            Ticket            │
              ├──────────────────────────────┤
              │ ID          int              │
              │ Title       string           │
              │ Status      string  ─────────┼──► Status.Name (soft ref)
              │ CreatedAt   time.Time        │
              │ Transitions []Transition     │
              └──────────────┬───────────────┘
                             │ contains 0..*
                             ▼
               ┌──────────────────────────┐
               │        Transition        │
               ├──────────────────────────┤
               │ From  string             │
               │ To    string             │
               │ At    time.Time          │
               └──────────────────────────┘


┌──────────────────────────────────────────────────────────────┐
│                        GlobalState                           │
├──────────────────────────────────────────────────────────────┤
│ Projects  map[projectPath]map[branch][]BranchTicket          │
└────────────────────────────┬─────────────────────────────────┘
                             │
                             ▼
               ┌──────────────────────────┐
               │       BranchTicket       │
               ├──────────────────────────┤
               │ TicketID  int            │
               │ Status    string         │
               └──────────────────────────┘
```

**Note:** `Ticket.Status` holds a status **name string**, not an ID. This is intentional — YAML files remain readable without a lookup table. When a status is renamed, a migration loop updates all ticket status strings to match.

---

## config.yaml

Full schema with all fields:

```yaml
title_limit: 0                  # int — 0 means no limit
default_done_status: "Done"     # string — must match a status name
auto_commit_on_done: false      # bool — triggers git commit flow on done
statuses:
  - id: 1
    name: "To Do"
    color: "red"
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

**Status color names:** `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`

**Rules:**
- Minimum 2 statuses required
- `order` determines column order in the TUI (ascending)
- Status `id` values are never reused within a project
- `default_done_status` must match an existing status name exactly

---

## tickets.yaml

```yaml
tickets:
  - id: 1
    title: "Set up CI pipeline"
    status: "In Progress"
    created_at: 2026-01-14T09:00:00Z
    transitions:
      - from: "To Do"
        to: "In Progress"
        at: 2026-01-15T11:30:00Z
  - id: 2
    title: "Initialise repository"
    status: "Done"
    created_at: 2026-01-10T08:00:00Z
    transitions:
      - from: "To Do"
        to: "Done"
        at: 2026-01-10T16:00:00Z
```

**Rules:**
- `id` is globally unique and monotonically increasing
- `id` is never reused, even after deletion (`NextID` reads `deleted.yaml` too)
- `transitions` is omitted when empty (YAML `omitempty`)
- Transition history is append-only — never mutated after the fact

---

## deleted.yaml

Same schema as `tickets.yaml`. Tickets are moved here when deleted from the TUI. The file is created on first deletion.

```yaml
tickets:
  - id: 3
    title: "Old spike task"
    status: "To Do"
    created_at: 2026-01-12T10:00:00Z
```

---

## state.yaml (`~/.config/tasklin/state.yaml`)

Tracks branch-level status overrides. Written when a ticket is moved on a non-main branch. Does **not** modify `tickets.yaml` — overrides are applied in memory at TUI startup.

```yaml
projects:
  /home/user/my-project:
    main: []
    feature/auth-refactor:
      - ticket_id: 7
        status: "In Progress"
      - ticket_id: 12
        status: "Review"
```

**Rules:**
- Keyed by absolute project path, then by branch name
- When the TUI starts on a non-main branch, `ApplyBranchOverrides` shadows `Ticket.Status` in memory
- `tickets.yaml` is only updated when mutations happen on the main branch

---

## ID generation

`store.NextID()` guarantees uniqueness across the full lifetime of a project:

```
NextID()
  ├── ReadTickets()    → active ticket IDs
  ├── ReadDeleted()    → deleted ticket IDs
  └── return max(all IDs) + 1
```

This means if tickets 1–10 exist and tickets 3 and 7 were deleted, the next ID is 11, not 3 or 7.

---

## Status name as foreign key

`Ticket.Status` stores the status name as a plain string, not an integer ID. This has two implications:

1. **Human-readable YAML** — you can read and edit `tickets.yaml` without a lookup table
2. **Rename migration required** — when a status is renamed in the TUI, `updateStatusName()` iterates all tickets and updates the string in-place before persisting

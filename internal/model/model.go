package model

import "time"

// Status represents a ticket status column.
type Status struct {
	ID    int    `yaml:"id"`
	Name  string `yaml:"name"`
	Color string `yaml:"color"`
	Order int    `yaml:"order"`
}

// Transition records a single status change on a ticket.
type Transition struct {
	From string    `yaml:"from"`
	To   string    `yaml:"to"`
	At   time.Time `yaml:"at"`
}

// Ticket is a single backlog item.
type Ticket struct {
	ID          int          `yaml:"id"`
	Title       string       `yaml:"title"`
	Status      string       `yaml:"status"`
	Labels      []string     `yaml:"labels,omitempty"`
	CreatedAt   time.Time    `yaml:"created_at"`
	Transitions []Transition `yaml:"transitions,omitempty"`
}

// Config holds project-level configuration.
type Config struct {
	TitleLimit        int      `yaml:"title_limit"`
	DefaultDoneStatus string   `yaml:"default_done_status"`
	AutoCommitOnDone  bool     `yaml:"auto_commit_on_done"`
	MinColWidth       int      `yaml:"min_col_width"`
	Statuses          []Status `yaml:"statuses"`
}

// TicketFile is the top-level structure for tickets.yaml / deleted.yaml.
type TicketFile struct {
	Tickets []Ticket `yaml:"tickets"`
}

// BranchTicket records a ticket status override for a branch.
type BranchTicket struct {
	TicketID int    `yaml:"ticket_id"`
	Status   string `yaml:"status"`
}

// GlobalState is the top-level structure for ~/.config/tasklin/state.yaml.
// Structure: projects[projectPath][branch] = []BranchTicket
type GlobalState struct {
	Projects map[string]map[string][]BranchTicket `yaml:"projects"`
}

// DefaultStatuses returns the built-in status set.
func DefaultStatuses() []Status {
	return []Status{
		{ID: 1, Name: "To Do", Color: "red", Order: 0},
		{ID: 2, Name: "In Progress", Color: "yellow", Order: 1},
		{ID: 3, Name: "Done", Color: "green", Order: 2},
	}
}

// DefaultConfig returns a config with defaults applied.
func DefaultConfig() Config {
	return Config{
		TitleLimit:        0,
		DefaultDoneStatus: "Done",
		MinColWidth:       16,
		Statuses:          DefaultStatuses(),
	}
}

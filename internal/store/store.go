package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/frankcruz/tasklin/internal/model"
	"gopkg.in/yaml.v3"
)

const (
	TodoDir     = ".todo"
	ConfigFile  = "config.yaml"
	TicketsFile = "tickets.yaml"
	DeletedFile = "deleted.yaml"
)

// Store manages reading and writing project data.
type Store struct {
	root string
}

// New returns a Store rooted at the given directory.
func New(root string) *Store {
	return &Store{root: root}
}

// TodoPath returns the path to the .todo directory.
func (s *Store) TodoPath() string {
	return filepath.Join(s.root, TodoDir)
}

// Initialised returns true if .todo/ exists.
func (s *Store) Initialised() bool {
	_, err := os.Stat(s.TodoPath())
	return err == nil
}

// Init creates the .todo/ directory and writes initial files.
func (s *Store) Init(cfg model.Config) error {
	if err := os.MkdirAll(s.TodoPath(), 0755); err != nil {
		return fmt.Errorf("create .todo/: %w", err)
	}
	if err := s.WriteConfig(cfg); err != nil {
		return err
	}
	// Write empty tickets.yaml if it doesn't exist.
	tp := filepath.Join(s.TodoPath(), TicketsFile)
	if _, err := os.Stat(tp); os.IsNotExist(err) {
		if err := writeYAML(tp, model.TicketFile{}); err != nil {
			return err
		}
	}
	return nil
}

// ReadConfig reads config.yaml.
func (s *Store) ReadConfig() (model.Config, error) {
	var cfg model.Config
	if err := readYAML(filepath.Join(s.TodoPath(), ConfigFile), &cfg); err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	return cfg, nil
}

// WriteConfig writes config.yaml.
func (s *Store) WriteConfig(cfg model.Config) error {
	return writeYAML(filepath.Join(s.TodoPath(), ConfigFile), cfg)
}

// ReadTickets reads tickets.yaml.
func (s *Store) ReadTickets() ([]model.Ticket, error) {
	var tf model.TicketFile
	if err := readYAML(filepath.Join(s.TodoPath(), TicketsFile), &tf); err != nil {
		return nil, fmt.Errorf("read tickets: %w", err)
	}
	return tf.Tickets, nil
}

// WriteTickets writes tickets.yaml.
func (s *Store) WriteTickets(tickets []model.Ticket) error {
	return writeYAML(filepath.Join(s.TodoPath(), TicketsFile), model.TicketFile{Tickets: tickets})
}

// ReadDeleted reads deleted.yaml.
func (s *Store) ReadDeleted() ([]model.Ticket, error) {
	var tf model.TicketFile
	p := filepath.Join(s.TodoPath(), DeletedFile)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil, nil
	}
	if err := readYAML(p, &tf); err != nil {
		return nil, fmt.Errorf("read deleted: %w", err)
	}
	return tf.Tickets, nil
}

// WriteDeleted writes deleted.yaml.
func (s *Store) WriteDeleted(tickets []model.Ticket) error {
	return writeYAML(filepath.Join(s.TodoPath(), DeletedFile), model.TicketFile{Tickets: tickets})
}

// NextID returns the next unique ticket ID (max ever created + 1).
func (s *Store) NextID() (int, error) {
	active, err := s.ReadTickets()
	if err != nil {
		return 0, err
	}
	deleted, err := s.ReadDeleted()
	if err != nil {
		return 0, err
	}
	max := 0
	for _, t := range append(active, deleted...) {
		if t.ID > max {
			max = t.ID
		}
	}
	return max + 1, nil
}

// SortedStatuses returns statuses ordered by their Order field.
func SortedStatuses(statuses []model.Status) []model.Status {
	sorted := make([]model.Status, len(statuses))
	copy(sorted, statuses)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Order < sorted[j].Order
	})
	return sorted
}

// --- helpers ---

func readYAML(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}

func writeYAML(path string, in interface{}) error {
	data, err := yaml.Marshal(in)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

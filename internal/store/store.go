package store

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/frankcruz/tasklin/internal/model"
	"gopkg.in/yaml.v3"
)

const (
	TodoDir    = ".todo"
	ConfigFile = "config.yaml"
	TicketsDir = "tickets"
	DeletedDir = "deleted"
	LabelsFile = "labels.yaml"
)

type labelsIndex struct {
	Labels []string `yaml:"labels"`
}

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

func (s *Store) ticketsPath() string { return filepath.Join(s.TodoPath(), TicketsDir) }
func (s *Store) deletedPath() string { return filepath.Join(s.TodoPath(), DeletedDir) }

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
	if err := os.MkdirAll(s.ticketsPath(), 0755); err != nil {
		return fmt.Errorf("create tickets/: %w", err)
	}
	if err := os.MkdirAll(s.deletedPath(), 0755); err != nil {
		return fmt.Errorf("create deleted/: %w", err)
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

// ReadTickets reads all tickets from the tickets/ directory.
func (s *Store) ReadTickets() ([]model.Ticket, error) {
	entries, err := os.ReadDir(s.ticketsPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read tickets: %w", err)
	}
	var tickets []model.Ticket
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		var t model.Ticket
		if err := readYAML(filepath.Join(s.ticketsPath(), e.Name()), &t); err != nil {
			return nil, fmt.Errorf("read ticket %s: %w", e.Name(), err)
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// WriteTicket writes a single ticket to tickets/<id>.yaml.
func (s *Store) WriteTicket(t model.Ticket) error {
	if err := os.MkdirAll(s.ticketsPath(), 0755); err != nil {
		return err
	}
	return writeYAML(filepath.Join(s.ticketsPath(), t.ID+".yaml"), t)
}

// DeleteTicketFile removes tickets/<id>.yaml.
func (s *Store) DeleteTicketFile(id string) error {
	return os.Remove(filepath.Join(s.ticketsPath(), id+".yaml"))
}

// ReadDeleted reads all tickets from the deleted/ directory.
func (s *Store) ReadDeleted() ([]model.Ticket, error) {
	entries, err := os.ReadDir(s.deletedPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read deleted: %w", err)
	}
	var tickets []model.Ticket
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		var t model.Ticket
		if err := readYAML(filepath.Join(s.deletedPath(), e.Name()), &t); err != nil {
			return nil, fmt.Errorf("read deleted ticket %s: %w", e.Name(), err)
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// WriteDeletedTicket writes a ticket to deleted/<id>.yaml.
func (s *Store) WriteDeletedTicket(t model.Ticket) error {
	if err := os.MkdirAll(s.deletedPath(), 0755); err != nil {
		return err
	}
	return writeYAML(filepath.Join(s.deletedPath(), t.ID+".yaml"), t)
}

// ReadLabels reads labels.yaml (the set of known labels for autocomplete).
func (s *Store) ReadLabels() ([]string, error) {
	p := filepath.Join(s.TodoPath(), LabelsFile)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil, nil
	}
	var idx labelsIndex
	if err := readYAML(p, &idx); err != nil {
		return nil, fmt.Errorf("read labels: %w", err)
	}
	return idx.Labels, nil
}

// WriteLabels writes labels.yaml.
func (s *Store) WriteLabels(labels []string) error {
	return writeYAML(filepath.Join(s.TodoPath(), LabelsFile), labelsIndex{Labels: labels})
}

// NewID returns a random 8-character lowercase hex string.
func NewID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%08x", b), nil
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

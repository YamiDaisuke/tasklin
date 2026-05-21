package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/frankcruz/tasklin/internal/model"
)

// legacyTicket parses the old integer-ID ticket format.
type legacyTicket struct {
	ID          int                `yaml:"id"`
	Title       string             `yaml:"title"`
	Status      string             `yaml:"status"`
	Labels      []string           `yaml:"labels,omitempty"`
	CreatedAt   time.Time          `yaml:"created_at"`
	Transitions []model.Transition `yaml:"transitions,omitempty"`
}

type legacyTicketFile struct {
	Tickets []legacyTicket `yaml:"tickets"`
}

// MigrateIfNeeded migrates from the legacy single-file format to per-file storage.
// Returns true if any migration was performed.
func (s *Store) MigrateIfNeeded() (bool, error) {
	migratedTickets, err1 := s.migrateTickets()
	migratedDeleted, err2 := s.migrateDeleted()
	if migratedTickets || migratedDeleted {
		deleteGlobalState()
	}
	return migratedTickets || migratedDeleted, errors.Join(err1, err2)
}

func (s *Store) migrateTickets() (bool, error) {
	oldPath := filepath.Join(s.TodoPath(), "tickets.yaml")
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return false, nil
	}
	var ltf legacyTicketFile
	if err := readYAML(oldPath, &ltf); err != nil {
		return false, fmt.Errorf("migrate tickets: %w", err)
	}
	if err := os.MkdirAll(s.ticketsPath(), 0755); err != nil {
		return false, err
	}
	for _, lt := range ltf.Tickets {
		id, err := NewID()
		if err != nil {
			return false, fmt.Errorf("migrate tickets: generate id: %w", err)
		}
		t := model.Ticket{
			ID:          id,
			Title:       lt.Title,
			Status:      lt.Status,
			Labels:      lt.Labels,
			CreatedAt:   lt.CreatedAt,
			Transitions: lt.Transitions,
		}
		if err := s.WriteTicket(t); err != nil {
			return false, err
		}
	}
	_ = os.Rename(oldPath, oldPath+".bak")
	return true, nil
}

func (s *Store) migrateDeleted() (bool, error) {
	oldPath := filepath.Join(s.TodoPath(), "deleted.yaml")
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return false, nil
	}
	var ltf legacyTicketFile
	if err := readYAML(oldPath, &ltf); err != nil {
		return false, fmt.Errorf("migrate deleted: %w", err)
	}
	if err := os.MkdirAll(s.deletedPath(), 0755); err != nil {
		return false, err
	}
	for _, lt := range ltf.Tickets {
		id, err := NewID()
		if err != nil {
			return false, fmt.Errorf("migrate deleted: generate id: %w", err)
		}
		t := model.Ticket{
			ID:          id,
			Title:       lt.Title,
			Status:      lt.Status,
			Labels:      lt.Labels,
			CreatedAt:   lt.CreatedAt,
			Transitions: lt.Transitions,
		}
		if err := s.WriteDeletedTicket(t); err != nil {
			return false, err
		}
	}
	_ = os.Rename(oldPath, oldPath+".bak")
	return true, nil
}

func deleteGlobalState() {
	dir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	_ = os.Remove(filepath.Join(dir, "tasklin", "state.yaml"))
}

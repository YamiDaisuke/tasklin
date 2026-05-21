package store_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/frankcruz/tasklin/internal/model"
	"github.com/frankcruz/tasklin/internal/store"
)

func newTempStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	return store.New(dir)
}

func TestInitialised_False(t *testing.T) {
	s := newTempStore(t)
	if s.Initialised() {
		t.Error("expected not initialised")
	}
}

func TestInit_CreatesDirectories(t *testing.T) {
	s := newTempStore(t)
	cfg := model.DefaultConfig()
	if err := s.Init(cfg); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if !s.Initialised() {
		t.Error("expected initialised after Init")
	}
	// config.yaml must exist
	configPath := filepath.Join(s.TodoPath(), "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config.yaml not found: %v", err)
	}
	// tickets/ directory must exist
	ticketsPath := filepath.Join(s.TodoPath(), "tickets")
	if _, err := os.Stat(ticketsPath); err != nil {
		t.Errorf("tickets/ dir not found: %v", err)
	}
	// deleted/ directory must exist
	deletedPath := filepath.Join(s.TodoPath(), "deleted")
	if _, err := os.Stat(deletedPath); err != nil {
		t.Errorf("deleted/ dir not found: %v", err)
	}
}

func TestWriteReadConfig(t *testing.T) {
	s := newTempStore(t)
	if err := s.Init(model.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	want := model.Config{
		TitleLimit:        80,
		DefaultDoneStatus: "Done",
		Statuses:          model.DefaultStatuses(),
	}
	if err := s.WriteConfig(want); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	got, err := s.ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}
	if got.TitleLimit != want.TitleLimit {
		t.Errorf("TitleLimit: want %d, got %d", want.TitleLimit, got.TitleLimit)
	}
	if got.DefaultDoneStatus != want.DefaultDoneStatus {
		t.Errorf("DefaultDoneStatus: want %q, got %q", want.DefaultDoneStatus, got.DefaultDoneStatus)
	}
}

func TestWriteReadTickets(t *testing.T) {
	s := newTempStore(t)
	if err := s.Init(model.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	tickets := []model.Ticket{
		{ID: "abc00001", Title: "First ticket", Status: "To Do", CreatedAt: time.Now().UTC()},
		{ID: "abc00002", Title: "Second ticket", Status: "In Progress", CreatedAt: time.Now().UTC()},
	}
	for _, tk := range tickets {
		if err := s.WriteTicket(tk); err != nil {
			t.Fatalf("WriteTicket: %v", err)
		}
	}
	got, err := s.ReadTickets()
	if err != nil {
		t.Fatalf("ReadTickets: %v", err)
	}
	if len(got) != len(tickets) {
		t.Fatalf("expected %d tickets, got %d", len(tickets), len(got))
	}
	byID := map[string]model.Ticket{}
	for _, tk := range got {
		byID[tk.ID] = tk
	}
	for _, want := range tickets {
		got, ok := byID[want.ID]
		if !ok {
			t.Errorf("ticket %s not found", want.ID)
			continue
		}
		if got.Title != want.Title {
			t.Errorf("ticket %s: Title mismatch: want %q, got %q", want.ID, want.Title, got.Title)
		}
	}
}

func TestDeleteTicketFile(t *testing.T) {
	s := newTempStore(t)
	if err := s.Init(model.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	tk := model.Ticket{ID: "abc00001", Title: "task", Status: "To Do", CreatedAt: time.Now().UTC()}
	if err := s.WriteTicket(tk); err != nil {
		t.Fatalf("WriteTicket: %v", err)
	}
	if err := s.DeleteTicketFile(tk.ID); err != nil {
		t.Fatalf("DeleteTicketFile: %v", err)
	}
	got, err := s.ReadTickets()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 tickets after delete, got %d", len(got))
	}
}

func TestReadTickets_EmptyDir(t *testing.T) {
	s := newTempStore(t)
	if err := s.Init(model.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadTickets()
	if err != nil {
		t.Fatalf("ReadTickets on empty dir: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 tickets, got %d", len(got))
	}
}

func TestNewID(t *testing.T) {
	hexRe := regexp.MustCompile(`^[0-9a-f]{8}$`)
	seen := make(map[string]bool, 200)
	for i := 0; i < 200; i++ {
		id, err := store.NewID()
		if err != nil {
			t.Fatalf("NewID call %d: %v", i, err)
		}
		if !hexRe.MatchString(id) {
			t.Errorf("NewID returned %q, want 8 lowercase hex chars", id)
		}
		if seen[id] {
			t.Errorf("NewID produced duplicate id %q after %d calls", id, i)
		}
		seen[id] = true
	}
}

func TestWriteReadDeletedTicket(t *testing.T) {
	s := newTempStore(t)
	if err := s.Init(model.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	tk := model.Ticket{ID: "dead0001", Title: "archived task", Status: "Done", CreatedAt: time.Now().UTC()}
	if err := s.WriteDeletedTicket(tk); err != nil {
		t.Fatalf("WriteDeletedTicket: %v", err)
	}
	got, err := s.ReadDeleted()
	if err != nil {
		t.Fatalf("ReadDeleted: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 deleted ticket, got %d", len(got))
	}
	if got[0].ID != tk.ID {
		t.Errorf("ID mismatch: want %q, got %q", tk.ID, got[0].ID)
	}
	if got[0].Title != tk.Title {
		t.Errorf("Title mismatch: want %q, got %q", tk.Title, got[0].Title)
	}
}

func TestMigrateIfNeeded_Tickets(t *testing.T) {
	dir := t.TempDir()
	s := store.New(dir)
	if err := s.Init(model.DefaultConfig()); err != nil {
		t.Fatal(err)
	}

	// Write a legacy tickets.yaml with integer-ID tickets.
	legacyYAML := `tickets:
  - id: 1
    title: First
    status: To Do
    created_at: 2024-01-01T00:00:00Z
  - id: 2
    title: Second
    status: In Progress
    created_at: 2024-01-02T00:00:00Z
`
	if err := os.WriteFile(filepath.Join(s.TodoPath(), "tickets.yaml"), []byte(legacyYAML), 0644); err != nil {
		t.Fatal(err)
	}

	migrated, err := s.MigrateIfNeeded()
	if err != nil {
		t.Fatalf("MigrateIfNeeded: %v", err)
	}
	if !migrated {
		t.Error("expected migration to have occurred")
	}

	tickets, err := s.ReadTickets()
	if err != nil {
		t.Fatalf("ReadTickets after migration: %v", err)
	}
	if len(tickets) != 2 {
		t.Fatalf("expected 2 tickets after migration, got %d", len(tickets))
	}

	hexRe := regexp.MustCompile(`^[0-9a-f]{8}$`)
	titles := map[string]bool{}
	for _, tk := range tickets {
		if !hexRe.MatchString(tk.ID) {
			t.Errorf("migrated ticket has non-hex ID %q; git hooks require 8-char hex IDs", tk.ID)
		}
		titles[tk.Title] = true
	}
	if !titles["First"] || !titles["Second"] {
		t.Error("migrated tickets are missing expected titles")
	}

	// Legacy file must have been renamed to .bak.
	if _, err := os.Stat(filepath.Join(s.TodoPath(), "tickets.yaml")); !os.IsNotExist(err) {
		t.Error("expected tickets.yaml to be removed after migration")
	}
	if _, err := os.Stat(filepath.Join(s.TodoPath(), "tickets.yaml.bak")); err != nil {
		t.Error("expected tickets.yaml.bak to exist after migration")
	}
}

func TestSortedStatuses(t *testing.T) {
	statuses := []model.Status{
		{ID: 3, Name: "Done", Order: 2},
		{ID: 1, Name: "To Do", Order: 0},
		{ID: 2, Name: "In Progress", Order: 1},
	}
	sorted := store.SortedStatuses(statuses)
	for i, expected := range []string{"To Do", "In Progress", "Done"} {
		if sorted[i].Name != expected {
			t.Errorf("sorted[%d]: expected %q, got %q", i, expected, sorted[i].Name)
		}
	}
}

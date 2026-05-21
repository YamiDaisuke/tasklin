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
	id, err := store.NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	if !hexRe.MatchString(id) {
		t.Errorf("NewID returned %q, want 8 lowercase hex chars", id)
	}
	// Two calls should produce different IDs.
	id2, err := store.NewID()
	if err != nil {
		t.Fatalf("NewID (second call): %v", err)
	}
	if id == id2 {
		t.Error("two NewID calls returned the same value")
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

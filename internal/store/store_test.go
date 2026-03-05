package store_test

import (
	"os"
	"path/filepath"
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

func TestInit_CreatesFiles(t *testing.T) {
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
	// tickets.yaml must exist
	ticketsPath := filepath.Join(s.TodoPath(), "tickets.yaml")
	if _, err := os.Stat(ticketsPath); err != nil {
		t.Errorf("tickets.yaml not found: %v", err)
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
		{ID: 1, Title: "First ticket", Status: "To Do", CreatedAt: time.Now().UTC()},
		{ID: 2, Title: "Second ticket", Status: "In Progress", CreatedAt: time.Now().UTC()},
	}
	if err := s.WriteTickets(tickets); err != nil {
		t.Fatalf("WriteTickets: %v", err)
	}
	got, err := s.ReadTickets()
	if err != nil {
		t.Fatalf("ReadTickets: %v", err)
	}
	if len(got) != len(tickets) {
		t.Fatalf("expected %d tickets, got %d", len(tickets), len(got))
	}
	for i, tk := range got {
		if tk.ID != tickets[i].ID {
			t.Errorf("ticket %d: ID mismatch: want %d, got %d", i, tickets[i].ID, tk.ID)
		}
		if tk.Title != tickets[i].Title {
			t.Errorf("ticket %d: Title mismatch: want %q, got %q", i, tickets[i].Title, tk.Title)
		}
	}
}

func TestNextID_Empty(t *testing.T) {
	s := newTempStore(t)
	if err := s.Init(model.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	id, err := s.NextID()
	if err != nil {
		t.Fatalf("NextID: %v", err)
	}
	if id != 1 {
		t.Errorf("expected id 1, got %d", id)
	}
}

func TestNextID_AfterTickets(t *testing.T) {
	s := newTempStore(t)
	if err := s.Init(model.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	tickets := []model.Ticket{
		{ID: 1, Title: "a", Status: "To Do"},
		{ID: 3, Title: "b", Status: "Done"},
	}
	if err := s.WriteTickets(tickets); err != nil {
		t.Fatal(err)
	}
	id, err := s.NextID()
	if err != nil {
		t.Fatalf("NextID: %v", err)
	}
	if id != 4 {
		t.Errorf("expected id 4, got %d", id)
	}
}

func TestNextID_NeverReusesDeleted(t *testing.T) {
	s := newTempStore(t)
	if err := s.Init(model.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	// Active has id 1, deleted has id 5.
	if err := s.WriteTickets([]model.Ticket{{ID: 1, Title: "a", Status: "To Do"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteDeleted([]model.Ticket{{ID: 5, Title: "deleted", Status: "Done"}}); err != nil {
		t.Fatal(err)
	}
	id, err := s.NextID()
	if err != nil {
		t.Fatal(err)
	}
	if id != 6 {
		t.Errorf("expected id 6 (max of active+deleted+1), got %d", id)
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

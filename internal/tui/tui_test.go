package tui_test

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/frankcruz/tasklin/internal/model"
	"github.com/frankcruz/tasklin/internal/store"
	"github.com/frankcruz/tasklin/internal/tui"
)

func setupModel(t *testing.T) tui.Model {
	t.Helper()
	dir := t.TempDir()
	s := store.New(dir)
	if err := s.Init(model.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	tickets := []model.Ticket{
		{ID: 1, Title: "First task", Status: "To Do", CreatedAt: time.Now()},
		{ID: 2, Title: "WIP task", Status: "In Progress", CreatedAt: time.Now()},
		{ID: 3, Title: "Completed task", Status: "Done", CreatedAt: time.Now()},
	}
	if err := s.WriteTickets(tickets); err != nil {
		t.Fatal(err)
	}
	m, err := tui.New(s, dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

func sendKey(m tui.Model, key string) tui.Model {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	if key == "right" {
		msg = tea.KeyMsg{Type: tea.KeyRight}
	} else if key == "left" {
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	} else if key == "down" {
		msg = tea.KeyMsg{Type: tea.KeyDown}
	} else if key == "up" {
		msg = tea.KeyMsg{Type: tea.KeyUp}
	} else if key == "enter" {
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	} else if key == "esc" {
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	} else if key == "backspace" {
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
	}
	result, _ := m.Update(msg)
	return result.(tui.Model)
}

func TestNew_LoadsTickets(t *testing.T) {
	m := setupModel(t)
	v := m.View()
	if v == "" {
		t.Error("View() returned empty string")
	}
}

func TestView_ContainsStatuses(t *testing.T) {
	m := setupModel(t)
	v := m.View()
	for _, name := range []string{"TO DO", "IN PROGRESS", "DONE"} {
		if !contains(v, name) {
			t.Errorf("View() missing status %q", name)
		}
	}
}

func TestView_ContainsTicketTitles(t *testing.T) {
	m := setupModel(t)
	v := m.View()
	for _, title := range []string{"First task", "WIP task", "Completed task"} {
		if !contains(v, title) {
			t.Errorf("View() missing ticket title %q", title)
		}
	}
}

func TestNavigation_MoveRight(t *testing.T) {
	m := setupModel(t)
	if m.ColIdx() != 0 {
		t.Errorf("initial colIdx should be 0, got %d", m.ColIdx())
	}
	m = sendKey(m, "right")
	if m.ColIdx() != 1 {
		t.Errorf("after right: colIdx should be 1, got %d", m.ColIdx())
	}
}

func TestNavigation_CannotGoLeftOfZero(t *testing.T) {
	m := setupModel(t)
	m = sendKey(m, "left")
	if m.ColIdx() != 0 {
		t.Errorf("colIdx should remain 0, got %d", m.ColIdx())
	}
}

func TestNavigation_MoveDown(t *testing.T) {
	m := setupModel(t)
	// col 0 = "To Do" has 1 ticket; can't go down further
	m = sendKey(m, "down")
	if m.RowIdx() != 0 {
		t.Errorf("rowIdx should remain 0 with only 1 ticket in col, got %d", m.RowIdx())
	}
}

func TestHelpMode(t *testing.T) {
	m := setupModel(t)
	m = sendKey(m, "?")
	v := m.View()
	if !contains(v, "Keyboard shortcuts") {
		t.Error("help overlay not shown")
	}
	// Any key closes help.
	m = sendKey(m, "q")
	v = m.View()
	if contains(v, "Keyboard shortcuts") {
		t.Error("help overlay should be closed")
	}
}

func TestWindowResize(t *testing.T) {
	m := setupModel(t)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m2 := result.(tui.Model)
	if m2.Width() != 120 || m2.Height() != 40 {
		t.Errorf("resize: want 120x40, got %dx%d", m2.Width(), m2.Height())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

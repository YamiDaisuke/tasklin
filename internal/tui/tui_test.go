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
	var msg tea.KeyMsg
	switch key {
	case "right":
		msg = tea.KeyMsg{Type: tea.KeyRight}
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "shift+right":
		msg = tea.KeyMsg{Type: tea.KeyRight, Alt: false, Runes: nil}
		msg = tea.KeyMsg{Type: tea.KeyShiftRight}
	case "shift+left":
		msg = tea.KeyMsg{Type: tea.KeyShiftLeft}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
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
	if !contains(v, "Keyboard Shortcuts") {
		t.Error("help overlay not shown")
	}
	// Any key closes help.
	m = sendKey(m, "q")
	v = m.View()
	if contains(v, "Keyboard shortcuts") {
		t.Error("help overlay should be closed")
	}
}

func TestShiftRight_MovesTicketToNextColumn(t *testing.T) {
	m := setupModel(t)
	// Start in col 0 ("To Do"), ticket [1] is there.
	if m.ColIdx() != 0 {
		t.Fatalf("expected col 0, got %d", m.ColIdx())
	}
	m = sendKey(m, "shift+right")
	// Ticket should now be in "In Progress"; cursor should follow to col 1.
	if m.ColIdx() != 1 {
		t.Errorf("after shift+right: expected col 1, got %d", m.ColIdx())
	}
	// "To Do" column should now be empty in the view.
	v := m.View()
	if !contains(v, "TO DO (0)") {
		t.Error("expected TO DO column to be empty after shift+right")
	}
	if !contains(v, "IN PROGRESS (2)") {
		t.Error("expected IN PROGRESS column to have 2 tickets after shift+right")
	}
}

func TestShiftLeft_MovesTicketToPrevColumn(t *testing.T) {
	m := setupModel(t)
	// Navigate to col 1 ("In Progress").
	m = sendKey(m, "right")
	if m.ColIdx() != 1 {
		t.Fatalf("expected col 1, got %d", m.ColIdx())
	}
	m = sendKey(m, "shift+left")
	// Ticket should now be in "To Do"; cursor should follow to col 0.
	if m.ColIdx() != 0 {
		t.Errorf("after shift+left: expected col 0, got %d", m.ColIdx())
	}
	if !contains(m.View(), "TO DO (2)") {
		t.Error("expected TO DO column to have 2 tickets after shift+left")
	}
}

func TestShiftLeft_NoopAtFirstColumn(t *testing.T) {
	m := setupModel(t)
	// Already at col 0; shift+left should not crash or change anything.
	m = sendKey(m, "shift+left")
	if m.ColIdx() != 0 {
		t.Errorf("expected col 0, got %d", m.ColIdx())
	}
}

func TestShiftRight_NoopAtLastColumn(t *testing.T) {
	m := setupModel(t)
	// Navigate to last column.
	m = sendKey(m, "right")
	m = sendKey(m, "right")
	lastCol := m.ColIdx()
	m = sendKey(m, "shift+right")
	if m.ColIdx() != lastCol {
		t.Errorf("expected col to stay at %d, got %d", lastCol, m.ColIdx())
	}
}

func TestEmptyFocusedColumn_ShowsPlaceholder(t *testing.T) {
	m := setupModel(t)
	// Move ticket from col 0 to col 1, leaving col 0 empty, then go back to col 0.
	m = sendKey(m, "shift+right")
	m = sendKey(m, "left")
	if m.ColIdx() != 0 {
		t.Fatalf("expected col 0, got %d", m.ColIdx())
	}
	v := m.View()
	if !contains(v, "New ticket...") {
		t.Error("empty focused column should show 'New ticket...' placeholder")
	}
}

func TestEmptyFocusedColumn_EnterTriggersNewTicket(t *testing.T) {
	m := setupModel(t)
	// Move the only ticket in col 0 away, leaving it empty.
	m = sendKey(m, "shift+right")
	m = sendKey(m, "left")
	// Press Enter on the placeholder — should open the new-ticket input.
	m = sendKey(m, "enter")
	v := m.View()
	if !contains(v, "New Ticket") {
		t.Error("Enter on empty column placeholder should open new-ticket input")
	}
}

func TestFocusedEmptyColumn_ViewContainsHeader(t *testing.T) {
	m := setupModel(t)
	// Move to "Done" column (col 2) which has 1 ticket; then navigate to a
	// column that becomes empty by moving its ticket away.
	// Simplest: navigate to col 0 (To Do, 1 ticket), shift it right, col 0 now empty.
	m = sendKey(m, "shift+right")
	// col 0 is now empty and still focused (colIdx==0 before the move lands on col1).
	// Actually shift+right moves cursor to col 1. Let's navigate back to col 0.
	m = sendKey(m, "left")
	if m.ColIdx() != 0 {
		t.Fatalf("expected col 0, got %d", m.ColIdx())
	}
	v := m.View()
	if !contains(v, "TO DO (0)") {
		t.Errorf("empty focused column header not visible in view")
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

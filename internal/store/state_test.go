package store_test

import (
	"testing"

	"github.com/frankcruz/tasklin/internal/model"
	"github.com/frankcruz/tasklin/internal/store"
)

func TestSetAndGetBranchOverride(t *testing.T) {
	gs := model.GlobalState{}
	store.SetBranchOverride(&gs, "/proj", "feature/x", 1, "In Progress")
	store.SetBranchOverride(&gs, "/proj", "feature/x", 2, "Done")

	overrides := store.GetBranchOverrides(gs, "/proj", "feature/x")
	if len(overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(overrides))
	}

	// Update existing override.
	store.SetBranchOverride(&gs, "/proj", "feature/x", 1, "Done")
	overrides = store.GetBranchOverrides(gs, "/proj", "feature/x")
	for _, bt := range overrides {
		if bt.TicketID == 1 && bt.Status != "Done" {
			t.Errorf("ticket 1 override: expected 'Done', got %q", bt.Status)
		}
	}
}

func TestGetBranchOverrides_NoProject(t *testing.T) {
	gs := model.GlobalState{}
	overrides := store.GetBranchOverrides(gs, "/nonexistent", "main")
	if overrides != nil {
		t.Errorf("expected nil overrides, got %v", overrides)
	}
}

func TestApplyBranchOverrides(t *testing.T) {
	tickets := []model.Ticket{
		{ID: 1, Title: "a", Status: "To Do"},
		{ID: 2, Title: "b", Status: "In Progress"},
		{ID: 3, Title: "c", Status: "To Do"},
	}
	overrides := []model.BranchTicket{
		{TicketID: 1, Status: "In Progress"},
		{TicketID: 3, Status: "Done"},
	}
	result := store.ApplyBranchOverrides(tickets, overrides)

	expected := map[int]string{1: "In Progress", 2: "In Progress", 3: "Done"}
	for _, t2 := range result {
		if t2.Status != expected[t2.ID] {
			t.Errorf("ticket %d: expected status %q, got %q", t2.ID, expected[t2.ID], t2.Status)
		}
	}
}

func TestApplyBranchOverrides_DoesNotMutateOriginal(t *testing.T) {
	tickets := []model.Ticket{
		{ID: 1, Title: "a", Status: "To Do"},
	}
	overrides := []model.BranchTicket{{TicketID: 1, Status: "Done"}}
	store.ApplyBranchOverrides(tickets, overrides)

	if tickets[0].Status != "To Do" {
		t.Error("original tickets slice was mutated")
	}
}

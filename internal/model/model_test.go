package model_test

import (
	"testing"

	"github.com/frankcruz/tasklin/internal/model"
)

func TestDefaultStatuses(t *testing.T) {
	statuses := model.DefaultStatuses()
	if len(statuses) != 3 {
		t.Fatalf("expected 3 default statuses, got %d", len(statuses))
	}
	names := []string{"To Do", "In Progress", "Done"}
	for i, st := range statuses {
		if st.Name != names[i] {
			t.Errorf("status %d: expected name %q, got %q", i, names[i], st.Name)
		}
		if st.ID != i+1 {
			t.Errorf("status %d: expected ID %d, got %d", i, i+1, st.ID)
		}
		if st.Order != i {
			t.Errorf("status %d: expected order %d, got %d", i, i, st.Order)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := model.DefaultConfig()
	if cfg.TitleLimit != 0 {
		t.Errorf("expected TitleLimit 0, got %d", cfg.TitleLimit)
	}
	if cfg.DefaultDoneStatus != "Done" {
		t.Errorf("expected DefaultDoneStatus 'Done', got %q", cfg.DefaultDoneStatus)
	}
	if len(cfg.Statuses) != 3 {
		t.Errorf("expected 3 statuses in default config, got %d", len(cfg.Statuses))
	}
}

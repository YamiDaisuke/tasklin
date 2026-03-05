package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/frankcruz/tasklin/internal/model"
	"gopkg.in/yaml.v3"
)

func globalStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tasklin", "state.yaml"), nil
}

// ReadGlobalState reads ~/.config/tasklin/state.yaml.
func ReadGlobalState() (model.GlobalState, error) {
	var gs model.GlobalState
	p, err := globalStatePath()
	if err != nil {
		return gs, err
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return model.GlobalState{Projects: map[string]map[string][]model.BranchTicket{}}, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return gs, err
	}
	if err := yaml.Unmarshal(data, &gs); err != nil {
		return gs, fmt.Errorf("parse state.yaml: %w", err)
	}
	if gs.Projects == nil {
		gs.Projects = map[string]map[string][]model.BranchTicket{}
	}
	return gs, nil
}

// WriteGlobalState writes ~/.config/tasklin/state.yaml.
func WriteGlobalState(gs model.GlobalState) error {
	p, err := globalStatePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(gs)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// GetBranchOverrides returns branch-level ticket status overrides for a project+branch.
func GetBranchOverrides(gs model.GlobalState, projectPath, branch string) []model.BranchTicket {
	if proj, ok := gs.Projects[projectPath]; ok {
		return proj[branch]
	}
	return nil
}

// SetBranchOverride sets a ticket status override for a project+branch.
func SetBranchOverride(gs *model.GlobalState, projectPath, branch string, ticketID int, status string) {
	if gs.Projects == nil {
		gs.Projects = map[string]map[string][]model.BranchTicket{}
	}
	if gs.Projects[projectPath] == nil {
		gs.Projects[projectPath] = map[string][]model.BranchTicket{}
	}
	overrides := gs.Projects[projectPath][branch]
	for i, bt := range overrides {
		if bt.TicketID == ticketID {
			overrides[i].Status = status
			gs.Projects[projectPath][branch] = overrides
			return
		}
	}
	gs.Projects[projectPath][branch] = append(overrides, model.BranchTicket{TicketID: ticketID, Status: status})
}

// ApplyBranchOverrides returns tickets with branch overrides applied (runtime shadow).
func ApplyBranchOverrides(tickets []model.Ticket, overrides []model.BranchTicket) []model.Ticket {
	overrideMap := map[int]string{}
	for _, bt := range overrides {
		overrideMap[bt.TicketID] = bt.Status
	}
	result := make([]model.Ticket, len(tickets))
	copy(result, tickets)
	for i, t := range result {
		if s, ok := overrideMap[t.ID]; ok {
			result[i].Status = s
		}
	}
	return result
}

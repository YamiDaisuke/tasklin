package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/frankcruz/tasklin/internal/model"
	"github.com/frankcruz/tasklin/internal/store"
	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move <ticket-id> <status>",
	Short: "Move a ticket to a different status",
	Args:  cobra.ExactArgs(2),
	RunE:  runMove,
}

func runMove(cmd *cobra.Command, args []string) error {
	ticketID := args[0]
	targetStatus := args[1]

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	s := store.New(cwd)
	if !s.Initialised() {
		return fmt.Errorf(".todo/ not found — run 'tasklin init' first")
	}

	cfg, err := s.ReadConfig()
	if err != nil {
		return err
	}

	resolved := ""
	for _, st := range cfg.Statuses {
		if strings.EqualFold(st.Name, targetStatus) {
			resolved = st.Name
			break
		}
	}
	if resolved == "" {
		return fmt.Errorf("unknown status %q", targetStatus)
	}

	tickets, err := s.ReadTickets()
	if err != nil {
		return err
	}

	found := false
	for i, t := range tickets {
		if t.ID != ticketID {
			continue
		}
		found = true
		if t.Status == resolved {
			return nil
		}
		tr := model.Transition{From: t.Status, To: resolved, At: time.Now().UTC()}
		tickets[i].Status = resolved
		tickets[i].Transitions = append(tickets[i].Transitions, tr)
		if err := s.WriteTicket(tickets[i]); err != nil {
			return err
		}
		break
	}
	if !found {
		return fmt.Errorf("ticket %s not found", ticketID)
	}

	fmt.Printf("#%s → %s\n", ticketID, resolved)
	return nil
}

func init() {
	rootCmd.AddCommand(moveCmd)
}

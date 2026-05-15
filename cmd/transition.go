package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/frankcruz/tasklin/internal/model"
	"github.com/frankcruz/tasklin/internal/store"
	"github.com/spf13/cobra"
)

// transitionCmd is an internal command used by git hooks only.
var transitionCmd = &cobra.Command{
	Use:    "_transition <ticket-id> <status>",
	Short:  "Internal: transition a ticket status (used by git hooks)",
	Hidden: true,
	Args:   cobra.ExactArgs(2),
	RunE:   runTransition,
}

func runTransition(cmd *cobra.Command, args []string) error {
	ticketID := args[0]
	targetStatus := args[1]

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	s := store.New(cwd)
	if !s.Initialised() {
		return fmt.Errorf(".todo/ not found in %s", cwd)
	}

	tickets, err := s.ReadTickets()
	if err != nil {
		return err
	}

	for i, t := range tickets {
		if t.ID == ticketID {
			if t.Status == targetStatus {
				return nil // No change needed
			}
			tr := model.Transition{From: t.Status, To: targetStatus, At: time.Now().UTC()}
			tickets[i].Status = targetStatus
			tickets[i].Transitions = append(tickets[i].Transitions, tr)
			return s.WriteTicket(tickets[i])
		}
	}
	return fmt.Errorf("ticket %s not found", ticketID)
}

func init() {
	rootCmd.AddCommand(transitionCmd)
}

package cmd

import (
	"fmt"
	"os"
	"strconv"
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
	ticketID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid ticket id: %s", args[0])
	}
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
		break
	}
	if !found {
		return fmt.Errorf("ticket %d not found", ticketID)
	}

	if err := s.WriteTickets(tickets); err != nil {
		return err
	}

	fmt.Printf("#%d → %s\n", ticketID, resolved)
	return nil
}

func init() {
	rootCmd.AddCommand(moveCmd)
}

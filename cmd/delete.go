package cmd

import (
	"fmt"
	"os"

	"github.com/frankcruz/tasklin/internal/store"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <ticket-id>",
	Short: "Delete a ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func runDelete(cmd *cobra.Command, args []string) error {
	ticketID := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	s := store.New(cwd)
	if !s.Initialised() {
		return fmt.Errorf(".todo/ not found — run 'tasklin init' first")
	}

	tickets, err := s.ReadTickets()
	if err != nil {
		return err
	}

	idx := -1
	for i, t := range tickets {
		if t.ID == ticketID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("ticket %s not found", ticketID)
	}

	title := tickets[idx].Title

	if err := s.WriteDeletedTicket(tickets[idx]); err != nil {
		return err
	}
	if err := s.DeleteTicketFile(ticketID); err != nil {
		return err
	}

	fmt.Printf("#%s %s deleted\n", ticketID, title)
	return nil
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

package cmd

import (
	"fmt"
	"os"
	"strconv"

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
	ticketID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid ticket id: %s", args[0])
	}

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
		return fmt.Errorf("ticket %d not found", ticketID)
	}

	deleted, err := s.ReadDeleted()
	if err != nil {
		return err
	}

	title := tickets[idx].Title
	deleted = append(deleted, tickets[idx])
	tickets = append(tickets[:idx], tickets[idx+1:]...)

	if err := s.WriteTickets(tickets); err != nil {
		return err
	}
	if err := s.WriteDeleted(deleted); err != nil {
		return err
	}

	fmt.Printf("#%d %s deleted\n", ticketID, title)
	return nil
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

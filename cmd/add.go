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

var addLabels []string
var addStatus string

var addCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Create a new ticket",
	Long:  "Create a new ticket with an optional status and labels.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	title := strings.Join(args, " ")

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

	statuses := store.SortedStatuses(cfg.Statuses)
	if len(statuses) == 0 {
		return fmt.Errorf("no statuses configured")
	}

	status := statuses[0].Name
	if addStatus != "" {
		found := false
		for _, st := range statuses {
			if strings.EqualFold(st.Name, addStatus) {
				status = st.Name
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown status %q", addStatus)
		}
	}

	id, err := s.NextID()
	if err != nil {
		return err
	}

	ticket := model.Ticket{
		ID:        id,
		Title:     title,
		Status:    status,
		Labels:    addLabels,
		CreatedAt: time.Now().UTC(),
	}

	tickets, err := s.ReadTickets()
	if err != nil {
		return err
	}
	tickets = append(tickets, ticket)
	if err := s.WriteTickets(tickets); err != nil {
		return err
	}

	if len(addLabels) > 0 {
		known, err := s.ReadLabels()
		if err != nil {
			return err
		}
		existing := make(map[string]bool, len(known))
		for _, l := range known {
			existing[l] = true
		}
		for _, l := range addLabels {
			if !existing[l] {
				known = append(known, l)
				existing[l] = true
			}
		}
		if err := s.WriteLabels(known); err != nil {
			return err
		}
	}

	fmt.Printf("#%d %s\n", id, title)
	return nil
}

func init() {
	addCmd.Flags().StringArrayVarP(&addLabels, "label", "l", nil, "label to attach (repeatable)")
	addCmd.Flags().StringVarP(&addStatus, "status", "s", "", "initial status (defaults to first configured status)")
	rootCmd.AddCommand(addCmd)
}

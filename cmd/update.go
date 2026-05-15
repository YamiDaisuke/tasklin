package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/frankcruz/tasklin/internal/store"
	"github.com/spf13/cobra"
)

var updateTitle string
var updateAddLabels []string
var updateRemoveLabels []string

var updateCmd = &cobra.Command{
	Use:   "update <ticket-id>",
	Short: "Update a ticket's title or labels",
	Args:  cobra.ExactArgs(1),
	RunE:  runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ticketID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid ticket id: %s", args[0])
	}

	if !cmd.Flags().Changed("title") && !cmd.Flags().Changed("add-label") && !cmd.Flags().Changed("remove-label") {
		return fmt.Errorf("nothing to update: specify --title, --add-label, or --remove-label")
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

	t := &tickets[idx]
	var changes []string

	if cmd.Flags().Changed("title") && updateTitle != t.Title {
		changes = append(changes, fmt.Sprintf("  title: %q → %q", t.Title, updateTitle))
		t.Title = updateTitle
	}

	if cmd.Flags().Changed("add-label") || cmd.Flags().Changed("remove-label") {
		existing := make(map[string]bool, len(t.Labels))
		for _, l := range t.Labels {
			existing[l] = true
		}

		var added, removed []string

		for _, l := range updateAddLabels {
			if !existing[l] {
				t.Labels = append(t.Labels, l)
				existing[l] = true
				added = append(added, l)
			}
		}

		remove := make(map[string]bool, len(updateRemoveLabels))
		for _, l := range updateRemoveLabels {
			remove[strings.ToLower(l)] = true
		}
		if len(remove) > 0 {
			kept := t.Labels[:0]
			for _, l := range t.Labels {
				if remove[strings.ToLower(l)] {
					removed = append(removed, l)
				} else {
					kept = append(kept, l)
				}
			}
			t.Labels = kept
		}

		if len(added) > 0 || len(removed) > 0 {
			var parts []string
			for _, l := range added {
				parts = append(parts, "+"+l)
			}
			for _, l := range removed {
				parts = append(parts, "-"+l)
			}
			changes = append(changes, "  labels: "+strings.Join(parts, ", "))
		}
	}

	if len(changes) == 0 {
		fmt.Printf("#%d no changes\n", ticketID)
		return nil
	}

	if err := s.WriteTickets(tickets); err != nil {
		return err
	}

	if cmd.Flags().Changed("add-label") {
		known, err := s.ReadLabels()
		if err != nil {
			return err
		}
		existing := make(map[string]bool, len(known))
		for _, l := range known {
			existing[l] = true
		}
		for _, l := range updateAddLabels {
			if !existing[l] {
				known = append(known, l)
				existing[l] = true
			}
		}
		if err := s.WriteLabels(known); err != nil {
			return err
		}
	}

	fmt.Printf("#%d %s\n", t.ID, t.Title)
	for _, c := range changes {
		fmt.Println(c)
	}
	return nil
}

func init() {
	updateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "new title")
	updateCmd.Flags().StringArrayVarP(&updateAddLabels, "add-label", "l", nil, "label to add (repeatable)")
	updateCmd.Flags().StringArrayVarP(&updateRemoveLabels, "remove-label", "r", nil, "label to remove (repeatable)")
	rootCmd.AddCommand(updateCmd)
}

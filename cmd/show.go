package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/frankcruz/tasklin/internal/store"
	"github.com/spf13/cobra"
)

var showVerbose bool

var showCmd = &cobra.Command{
	Use:   "show <ticket-id>",
	Short: "Show ticket details",
	Args:  cobra.ExactArgs(1),
	RunE:  runShow,
}

var (
	showIDStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	showTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	showKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(10)
	showChipStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	showDimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	showBoldStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	showSepStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	showArrowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

func statusColor(name string) lipgloss.Color {
	switch strings.ToLower(name) {
	case "red":
		return lipgloss.Color("1")
	case "green":
		return lipgloss.Color("2")
	case "yellow":
		return lipgloss.Color("3")
	case "blue":
		return lipgloss.Color("4")
	case "magenta":
		return lipgloss.Color("5")
	case "cyan":
		return lipgloss.Color("6")
	case "white":
		return lipgloss.Color("7")
	default:
		return lipgloss.Color(name)
	}
}

func runShow(cmd *cobra.Command, args []string) error {
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

	cfg, err := s.ReadConfig()
	if err != nil {
		return err
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
	t := tickets[idx]

	// Resolve status color from config.
	dotColor := lipgloss.Color("252")
	for _, st := range cfg.Statuses {
		if st.Name == t.Status {
			dotColor = statusColor(st.Color)
			break
		}
	}

	const sep = "──────────────────────────────────────────────"

	fmt.Println()
	fmt.Printf("  %s  %s\n",
		showIDStyle.Render(fmt.Sprintf("#%d", t.ID)),
		showTitleStyle.Render(t.Title),
	)
	fmt.Println("  " + showSepStyle.Render(sep))

	dot := lipgloss.NewStyle().Foreground(dotColor).Render("●")
	fmt.Printf("  %s%s %s\n",
		showKeyStyle.Render("Status"),
		dot,
		showBoldStyle.Render(t.Status),
	)

	if len(t.Labels) > 0 {
		chips := make([]string, len(t.Labels))
		for i, l := range t.Labels {
			chips[i] = showChipStyle.Render("[" + l + "]")
		}
		fmt.Printf("  %s%s\n", showKeyStyle.Render("Labels"), strings.Join(chips, " "))
	} else {
		fmt.Printf("  %s%s\n", showKeyStyle.Render("Labels"), showDimStyle.Render("none"))
	}

	fmt.Printf("  %s%s\n",
		showKeyStyle.Render("Created"),
		showDimStyle.Render(t.CreatedAt.Format("02 Jan 2006")),
	)

	if showVerbose {
		fmt.Println("  " + showSepStyle.Render(sep))
		fmt.Printf("  %s\n", showBoldStyle.Render("Transitions"))
		if len(t.Transitions) == 0 {
			fmt.Printf("  %s\n", showDimStyle.Render("  none"))
		} else {
			fmt.Println()
			for _, tr := range t.Transitions {
				when := showDimStyle.Render(tr.At.Format("02 Jan 2006  15:04"))
				from := showDimStyle.Render(tr.From)
				arrow := showArrowStyle.Render("→")
				to := showBoldStyle.Render(tr.To)
				fmt.Printf("    %s    %s %s %s\n", when, from, arrow, to)
			}
		}
	}

	fmt.Println()
	return nil
}

func init() {
	showCmd.Flags().BoolVarP(&showVerbose, "verbose", "v", false, "show full transition history")
	rootCmd.AddCommand(showCmd)
}

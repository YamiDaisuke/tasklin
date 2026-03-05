package cmd

import (
	"fmt"
	"os"

	"github.com/frankcruz/tasklin/internal/store"
	"github.com/frankcruz/tasklin/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tasklin",
	Short: "Personal project backlog CLI",
	Long:  "A lightweight CLI for managing personal project backlogs with a TUI kanban board.",
	RunE:  runRoot,
}

func runRoot(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	s := store.New(cwd)

	if !s.Initialised() {
		fmt.Println(".todo/ not found — running init first...")
		if err := runInit(cmd, args); err != nil {
			return err
		}
	}

	return tui.Run(s, cwd)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/frankcruz/tasklin/internal/hooks"
	internalgit "github.com/frankcruz/tasklin/internal/git"
	"github.com/frankcruz/tasklin/internal/model"
	"github.com/frankcruz/tasklin/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise .todo/ in the current directory",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	s := store.New(cwd)
	scanner := bufio.NewScanner(os.Stdin)

	// 1. Already initialised?
	if s.Initialised() {
		if !prompt(scanner, "Already initialised. Re-initialise? [y/N]", false) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// 2. Choose statuses.
	defaults := model.DefaultStatuses()
	fmt.Println("Default statuses:")
	for _, st := range defaults {
		fmt.Printf("  %s — %s — order %d\n", st.Name, st.Color, st.Order)
	}

	var statuses []model.Status
	if prompt(scanner, "Use default statuses? [Y/n]", true) {
		statuses = defaults
	} else {
		statuses = collectStatuses(scanner)
	}

	cfg := model.Config{
		TitleLimit:        0,
		DefaultDoneStatus: "Done",
		Statuses:          statuses,
	}

	// 3. Git hook prompt.
	gitRoot := internalgit.RepoRoot(cwd)
	if gitRoot != "" {
		if prompt(scanner, "Git repository detected. Add git hooks for automatic ticket transitions? [y/N]", false) {
			doneStatus := promptString(scanner, fmt.Sprintf("Transition to which status on merge? [%s]", cfg.DefaultDoneStatus))
			if doneStatus == "" {
				doneStatus = cfg.DefaultDoneStatus
			}
			gitDir := internalgit.GitDir(gitRoot)
			if err := hooks.InstallCommitMsg(gitDir, doneStatus); err != nil {
				return err
			}
			if err := hooks.InstallPostMerge(gitDir, doneStatus); err != nil {
				return err
			}
			fmt.Println("Installed commit-msg and post-merge hooks.")

			if prompt(scanner, "Also stage .todo/ folder in all commits? [Y/n]", true) {
				if err := hooks.InstallPreCommit(gitDir); err != nil {
					return err
				}
				fmt.Println("Installed pre-commit hook.")
			}
		}
	}

	// 4. Write config and tickets.yaml.
	if err := s.Init(cfg); err != nil {
		return err
	}

	todoPath := filepath.Join(cwd, store.TodoDir)
	fmt.Printf("Initialised .todo/ at %s\n", todoPath)
	return nil
}

func prompt(scanner *bufio.Scanner, question string, defaultYes bool) bool {
	fmt.Print(question + " ")
	if !scanner.Scan() {
		return defaultYes
	}
	ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if ans == "" {
		return defaultYes
	}
	return ans == "y" || ans == "yes"
}

func promptString(scanner *bufio.Scanner, question string) string {
	fmt.Print(question + " ")
	if !scanner.Scan() {
		return ""
	}
	return strings.TrimSpace(scanner.Text())
}

func collectStatuses(scanner *bufio.Scanner) []model.Status {
	var statuses []model.Status
	id := 1
	for {
		fmt.Printf("Status %d name (or empty to finish): ", id)
		if !scanner.Scan() {
			break
		}
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			if len(statuses) < 2 {
				fmt.Println("Minimum 2 statuses required.")
				continue
			}
			break
		}
		color := promptString(scanner, "  Color (ANSI name or code):")
		orderStr := promptString(scanner, fmt.Sprintf("  Order [%d]:", id-1))
		order := id - 1
		if o, err := strconv.Atoi(orderStr); err == nil {
			order = o
		}
		statuses = append(statuses, model.Status{ID: id, Name: name, Color: color, Order: order})
		id++
	}
	return statuses
}

func init() {
	rootCmd.AddCommand(initCmd)
}

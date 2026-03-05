package git

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// RepoRoot returns the git repo root from cwd, or "" if not in a repo.
func RepoRoot(cwd string) string {
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// CurrentBranch returns the current branch name, or "" on error.
func CurrentBranch(cwd string) string {
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GitDir returns the path to the .git directory for the given root.
func GitDir(root string) string {
	return filepath.Join(root, ".git")
}

// IsMainBranch returns true if branch is main or master.
func IsMainBranch(branch string) bool {
	return branch == "main" || branch == "master"
}

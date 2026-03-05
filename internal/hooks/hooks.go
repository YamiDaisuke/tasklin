package hooks

import (
	"fmt"
	"os"
	"path/filepath"
)

// findTasklin is a shell snippet that locates the tasklin binary.
// It checks PATH first, then common Go install locations, so the hook works
// even when git runs with a minimal environment that excludes ~/go/bin.
const findTasklin = `
TASKLIN=$(command -v tasklin 2>/dev/null)
if [ -z "$TASKLIN" ]; then
  for _dir in "$HOME/go/bin" "$HOME/.local/bin" "/usr/local/bin"; do
    if [ -x "$_dir/tasklin" ]; then
      TASKLIN="$_dir/tasklin"
      break
    fi
  done
fi
if [ -z "$TASKLIN" ]; then
  echo "tasklin not found, skipping ticket transition" >&2
  exit 0
fi
`

// InstallCommitMsg writes the commit-msg hook.
// Using commit-msg (not post-commit) means the ticket update is staged before
// git finalises the commit object, so the change lands in the same commit.
func InstallCommitMsg(gitDir, doneStatus string) error {
	script := fmt.Sprintf(`#!/bin/sh
MSG=$(cat "$1")
if echo "$MSG" | grep -qE '^\[([0-9]+)\]'; then
  TICKET_ID=$(echo "$MSG" | grep -oE '^\[([0-9]+)\]' | tr -d '[]')
%s
  "$TASKLIN" _transition "$TICKET_ID" "%s" && git add .todo/
fi
`, findTasklin, doneStatus)
	return writeHook(gitDir, "commit-msg", script)
}

// InstallPostMerge writes the post-merge hook.
// After transitioning the ticket it amends the merge commit so the .todo/
// change is recorded in the same commit rather than requiring a follow-up one.
func InstallPostMerge(gitDir, doneStatus string) error {
	script := fmt.Sprintf(`#!/bin/sh
BRANCH=$(git reflog | awk 'NR==1{print $6}' | sed 's/.*\///')
if echo "$BRANCH" | grep -qE '\[([0-9]+)\]'; then
  TICKET_ID=$(echo "$BRANCH" | grep -oE '\[([0-9]+)\]' | tr -d '[]')
%s
  "$TASKLIN" _transition "$TICKET_ID" "%s" && git add .todo/ && git commit --amend --no-edit --no-verify
fi
`, findTasklin, doneStatus)
	return writeHook(gitDir, "post-merge", script)
}

// InstallPreCommit writes the optional pre-commit hook that stages .todo/.
func InstallPreCommit(gitDir string) error {
	script := `#!/bin/sh
git add .todo/
`
	return writeHook(gitDir, "pre-commit", script)
}

func writeHook(gitDir, name, content string) error {
	p := filepath.Join(gitDir, "hooks", name)
	if err := os.WriteFile(p, []byte(content), 0755); err != nil {
		return fmt.Errorf("write hook %s: %w", name, err)
	}
	return nil
}

package hooks

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallPostCommit writes the post-commit hook.
func InstallPostCommit(gitDir, doneStatus string) error {
	script := fmt.Sprintf(`#!/bin/sh
MSG=$(git log -1 --pretty=%%B)
if echo "$MSG" | grep -qE '^\[([0-9]+)\]'; then
  TICKET_ID=$(echo "$MSG" | grep -oE '^\[([0-9]+)\]' | tr -d '[]')
  tasklin _transition "$TICKET_ID" "%s"
fi
`, doneStatus)
	return writeHook(gitDir, "post-commit", script)
}

// InstallPostMerge writes the post-merge hook.
func InstallPostMerge(gitDir, doneStatus string) error {
	script := fmt.Sprintf(`#!/bin/sh
BRANCH=$(git reflog | awk 'NR==1{print $6}' | sed 's/.*\///')
if echo "$BRANCH" | grep -qE '\[([0-9]+)\]'; then
  TICKET_ID=$(echo "$BRANCH" | grep -oE '\[([0-9]+)\]' | tr -d '[]')
  tasklin _transition "$TICKET_ID" "%s"
fi
`, doneStatus)
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

package hooks_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frankcruz/tasklin/internal/hooks"
)

func setupGitDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestInstallCommitMsg(t *testing.T) {
	gitDir := setupGitDir(t)
	if err := hooks.InstallCommitMsg(gitDir, "Done"); err != nil {
		t.Fatalf("InstallCommitMsg: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(gitDir, "hooks", "commit-msg"))
	if err != nil {
		t.Fatalf("commit-msg hook not found: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Done") {
		t.Error("commit-msg hook missing 'Done' status")
	}
	if !strings.Contains(content, "_transition") {
		t.Error("commit-msg hook missing '_transition' call")
	}
	if !strings.Contains(content, "git add .todo/") {
		t.Error("commit-msg hook missing 'git add .todo/' to stage the change")
	}
	if !strings.Contains(content, "command -v tasklin") {
		t.Error("commit-msg hook missing PATH fallback for tasklin")
	}
	// Check executable bit
	info, _ := os.Stat(filepath.Join(gitDir, "hooks", "commit-msg"))
	if info.Mode()&0111 == 0 {
		t.Error("commit-msg hook is not executable")
	}
}

func TestInstallPostMerge(t *testing.T) {
	gitDir := setupGitDir(t)
	if err := hooks.InstallPostMerge(gitDir, "Done"); err != nil {
		t.Fatalf("InstallPostMerge: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(gitDir, "hooks", "post-merge"))
	if err != nil {
		t.Fatalf("post-merge hook not found: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Done") {
		t.Error("post-merge hook missing 'Done' status")
	}
	if !strings.Contains(content, "git commit --amend --no-edit --no-verify") {
		t.Error("post-merge hook missing amend step to include change in merge commit")
	}
	if !strings.Contains(content, "command -v tasklin") {
		t.Error("post-merge hook missing PATH fallback for tasklin")
	}
}

func TestInstallPreCommit(t *testing.T) {
	gitDir := setupGitDir(t)
	if err := hooks.InstallPreCommit(gitDir); err != nil {
		t.Fatalf("InstallPreCommit: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(gitDir, "hooks", "pre-commit"))
	if err != nil {
		t.Fatalf("pre-commit hook not found: %v", err)
	}
	if !strings.Contains(string(data), "git add .todo/") {
		t.Error("pre-commit hook missing 'git add .todo/'")
	}
}

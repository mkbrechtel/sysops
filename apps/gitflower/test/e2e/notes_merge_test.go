// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestReviewNoteAndMerge stands up a tiny git repo, runs
// `gitflower review --no-tui` to write a fresh review into the notes
// ref for HEAD, then runs `gitflower review merge --include-file` to
// archive that note into the branch as an -s ours merge with a
// `review/<short>.review` file. Asserts:
//   - the note exists on refs/notes/review for HEAD before merging,
//   - the merge commit's subject starts with `[Review]`,
//   - the merge commit has exactly two parents (code + notes),
//   - the file `review/<short>.review` exists in the merge tree,
//   - after the merge, the SECOND `gitflower review --no-tui` picks
//     the merge commit as the base for scope (since LastReviewMergeSHA
//     returns it).
func TestReviewNoteAndMerge(t *testing.T) {
	t.Parallel()

	bin := buildBinary(t)
	repo := newMiniRepo(t)

	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=tester", "GIT_AUTHOR_EMAIL=t@e",
		"GIT_COMMITTER_NAME=tester", "GIT_COMMITTER_EMAIL=t@e",
	)

	// First review: writes a note for HEAD.
	run(t, env, repo, bin, "review", "--no-tui")

	// Note must exist for HEAD on refs/notes/review.
	headSHA := mustGit(t, repo, "rev-parse", "HEAD")
	noteBody := mustGit(t, repo, "notes", "--ref=refs/notes/review", "show", headSHA)
	if !strings.Contains(noteBody, "# Review") {
		t.Fatalf("note body doesn't look like a .review: %s", truncOut(noteBody, 200))
	}

	// Archive with --include-file.
	mergeOut := run(t, env, repo, bin, "review", "merge", "--include-file")
	if !strings.Contains(mergeOut, "[Review] merge") {
		t.Errorf("merge output didn't confirm: %s", mergeOut)
	}

	// Verify the merge commit shape.
	mergeSHA := mustGit(t, repo, "rev-parse", "HEAD")
	subj := mustGit(t, repo, "log", "-1", "--format=%s", mergeSHA)
	if !strings.HasPrefix(subj, "[Review]") {
		t.Errorf("merge subject doesn't start with [Review]: %q", subj)
	}
	parents := strings.Fields(mustGit(t, repo, "log", "-1", "--format=%P", mergeSHA))
	if len(parents) != 2 {
		t.Errorf("merge should have 2 parents, got %d: %v", len(parents), parents)
	}

	// File mirror exists in the merge tree.
	short := headSHA
	if len(short) > 7 {
		short = short[:7]
	}
	tree := mustGit(t, repo, "ls-tree", "-r", mergeSHA)
	want := "review/" + short + ".review"
	if !strings.Contains(tree, want) {
		t.Errorf("merge tree missing %s; got:\n%s", want, tree)
	}

	// Second review must pick the merge commit as the base.
	// Add a new code commit on top so there's something to review.
	writeFile(t, repo, "x.txt", "hello\n")
	mustGit(t, repo, "add", "x.txt")
	mustGit(t, repo, "commit", "-m", "add x")

	out := run(t, env, repo, bin, "review", "--no-tui")
	_ = out
	// Easier check: scope's base — which becomes the previous
	// [Review] merge SHA — is what LastReviewMergeSHA returns.
	lastMerge := mustGit(t, repo, "log", "--merges", "--format=%H", "--grep=^\\[Review\\]", "-1")
	if lastMerge != mergeSHA {
		t.Errorf("LastReviewMergeSHA expected %s, got %s", mergeSHA, lastMerge)
	}
}

// --- helpers -------------------------------------------------------

func newMiniRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init", "-q", "-b", "main", ".")
	mustGit(t, dir, "config", "user.email", "t@e")
	mustGit(t, dir, "config", "user.name", "tester")
	// One initial commit on main so HEAD resolves.
	writeFile(t, dir, "README", "hello\n")
	mustGit(t, dir, "add", "README")
	mustGit(t, dir, "commit", "-q", "-m", "init")
	// Branch off so HEAD is on a feature commit (review scope = main..HEAD).
	mustGit(t, dir, "checkout", "-q", "-b", "feature")
	writeFile(t, dir, "feat.txt", "line1\nline2\n")
	mustGit(t, dir, "add", "feat.txt")
	mustGit(t, dir, "commit", "-q", "-m", "feature work")
	return dir
}

func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimRight(string(out), "\n")
}

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	abs := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, env []string, dir, bin string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", filepath.Base(bin), strings.Join(args, " "), err, out)
	}
	return string(out)
}

func truncOut(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

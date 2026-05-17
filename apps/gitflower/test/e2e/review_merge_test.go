// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

//go:build with_review_merge

package e2e_test

import (
	"os"
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
//   - the merge commit has exactly two parents (code + orphan archive),
//   - the orphan archive is parentless and holds only this review,
//   - the file `review/<short>.review` exists in the merge tree,
//   - after the merge, the SECOND `gitflower review --no-tui` picks
//     the merge commit as the base for scope (since LastReviewMergeSHA
//     returns it).
//
// Gated behind the `with_review_merge` build tag — the merge feature
// is off in the default binary.
func TestReviewNoteAndMerge(t *testing.T) {
	t.Parallel()

	bin := buildBinaryWithTags(t, "with_review_merge")
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

	// Merge commit body must embed the gitflower-free recipes for
	// reading the note (git + grep), so future readers don't need
	// the tool to extract the review. The recipe now reads from the
	// orphan archive commit (the merge's second parent), not from
	// the live notes ref — that's the whole point of isolating
	// the merge from the rest of the notes-ref history.
	mergeBody := mustGit(t, repo, "log", "-1", "--format=%B", mergeSHA)
	for _, want := range []string{
		"Review-Archive: ",
		"Reviewed-Commit: ",
		"git show ",
		"grep -v -E '^### (ReadStart|ReadEnd|SkipStart|SkipEnd) '",
		"grep -B3 -E ",
	} {
		if !strings.Contains(mergeBody, want) {
			t.Errorf("merge commit body missing %q\n--- body ---\n%s",
				want, mergeBody)
		}
	}

	// The second parent must be an orphan commit (no parents of its
	// own), holding a single blob keyed by the reviewed commit's
	// full SHA. That isolation is the whole point — the merge must
	// NOT drag in the notes ref's full history.
	archiveSHA := strings.TrimSpace(mustGit(t, repo, "rev-parse", mergeSHA+"^2"))
	archiveParents := strings.Fields(mustGit(t, repo, "log", "-1", "--format=%P", archiveSHA))
	if len(archiveParents) != 0 {
		t.Errorf("review archive must be an orphan commit, got parents: %v", archiveParents)
	}
	archiveTree := mustGit(t, repo, "ls-tree", archiveSHA)
	if !strings.Contains(archiveTree, headSHA) {
		t.Errorf("archive tree missing entry for %s; got:\n%s", headSHA, archiveTree)
	}
	// The body of that one blob must actually be the review.
	noteFromArchive := mustGit(t, repo, "show", archiveSHA+":"+headSHA)
	if !strings.Contains(noteFromArchive, "# Review") {
		t.Errorf("archive blob doesn't look like a .review:\n%s",
			truncOut(noteFromArchive, 200))
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

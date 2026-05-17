// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

//go:build with_review_merge

package app

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"gitflower/internal/git"
	"gitflower/review"
)

// cmdReviewMerge creates a merge commit on the current branch whose
// second parent is an orphan archive commit holding only the
// reviewed commit's note as a blob. Optional --include-file also
// writes the rendered review body to review/<tip-short>.review in
// the merge commit's tree.
//
// All git operations go through internal/git (go-git). No external
// git binary needed.
func cmdReviewMerge(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("review merge", flag.ContinueOnError)
	fs.SetOutput(stderr)
	includeFile := fs.Bool("include-file", false, "also write the review body to review/<tip-short>.review in the merge commit's tree")
	notesRef := fs.String("notes-ref", review.DefaultNotesRef, "git notes ref holding the review body")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: gitflower review merge [--include-file] [--notes-ref ref]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Archive the active review note into HEAD as a two-parent merge")
		fmt.Fprintln(stderr, "commit subjected `[Review] <tip-short>`. The branch's tree is")
		fmt.Fprintln(stderr, "unchanged unless --include-file is passed.")
		fmt.Fprintln(stderr)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	repo, err := git.Open("")
	if err != nil {
		fmt.Fprintf(stderr, "review merge: %v\n", err)
		return 1
	}
	tip, err := repo.HeadHash()
	if err != nil {
		fmt.Fprintf(stderr, "review merge: %v\n", err)
		return 1
	}
	tipSHA := tip.String()

	// Confirm a note exists for HEAD; if not, refuse — there's
	// nothing to archive.
	body, err := review.ReadNote(*notesRef, tipSHA)
	if err != nil {
		fmt.Fprintf(stderr, "review merge: %v\n", err)
		return 1
	}
	if body == "" {
		fmt.Fprintf(stderr, "review merge: no note for HEAD (%s) on %s\n",
			tipSHA[:12], *notesRef)
		return 1
	}

	short := tipSHA
	if len(short) > 7 {
		short = short[:7]
	}
	subject := fmt.Sprintf("[Review] %s", short)

	// Start with HEAD's tree as the merge tree. --include-file
	// splices a single new file into it.
	mergeTree, err := repo.CommitTreeHash(tip)
	if err != nil {
		fmt.Fprintf(stderr, "review merge: %v\n", err)
		return 1
	}
	if *includeFile {
		relPath := filepath.Join("review", short+".review")
		blob, err := repo.WriteBlob([]byte(body))
		if err != nil {
			fmt.Fprintf(stderr, "review merge: write blob: %v\n", err)
			return 1
		}
		mergeTree, err = repo.AddFileToTree(mergeTree, relPath, blob)
		if err != nil {
			fmt.Fprintf(stderr, "review merge: splice %s: %v\n", relPath, err)
			return 1
		}
	}

	// Build a single-purpose orphan commit that holds ONLY this
	// review's note. The merge's second parent points here so the
	// [Review] merge doesn't drag in the full notes-ref history.
	archive, err := buildReviewArchive(repo, body, tipSHA, short)
	if err != nil {
		fmt.Fprintf(stderr, "review merge: archive: %v\n", err)
		return 1
	}
	archiveSHA := archive.String()

	// commit-tree with two parents: HEAD (first) and the orphan
	// archive (second). First-parent stays on the code line. The
	// commit body embeds three shell recipes that read the archived
	// note straight out of the merge's reachable objects — no `git
	// notes` ref required.
	viewCmds := review.ArchiveViewCommands(archiveSHA, tipSHA)
	commitBody := fmt.Sprintf(
		"%s\n\nReview-Archive: %s\nReviewed-Commit: %s\n\n"+
			"View the full review:\n  %s\n\n"+
			"Drop reading/skip bookkeeping:\n  %s\n\n"+
			"Just the reactions, with surrounding context:\n  %s\n",
		subject, archiveSHA, tipSHA,
		viewCmds[0], viewCmds[1], viewCmds[2],
	)
	merge, err := repo.WriteCommit(mergeTree, []git.Hash{tip, archive}, commitBody)
	if err != nil {
		fmt.Fprintf(stderr, "review merge: commit: %v\n", err)
		return 1
	}

	// Advance current branch to the new merge commit.
	if err := repo.UpdateRef("HEAD", merge); err != nil {
		fmt.Fprintf(stderr, "review merge: update-ref HEAD: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "[Review] merge %s created (tree=%s, archive=%s)\n",
		merge.String()[:12], mergeTree.String()[:12], archiveSHA[:12])
	return 0
}

// buildReviewArchive writes the review body as a blob, wraps it in
// a one-entry tree keyed by the reviewed commit's SHA (matching
// git's notes layout), and creates an orphan commit referencing
// that tree. Returns the orphan commit's OID.
func buildReviewArchive(repo *git.Repo, body, sha, short string) (git.Hash, error) {
	blob, err := repo.WriteBlob([]byte(body))
	if err != nil {
		return git.ZeroHash, fmt.Errorf("hash-object: %w", err)
	}
	tree, err := repo.WriteTree([]git.TreeEntry{{Name: sha, Hash: blob}})
	if err != nil {
		return git.ZeroHash, fmt.Errorf("mktree: %w", err)
	}
	commit, err := repo.WriteCommit(tree, nil, fmt.Sprintf("[Review archive] %s", short))
	if err != nil {
		return git.ZeroHash, fmt.Errorf("commit-tree: %w", err)
	}
	return commit, nil
}

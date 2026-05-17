// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

// Notes-backed review storage. A reviewer's session for one commit
// lives as the note body for that commit in refs/notes/review (or
// whichever ref the caller specifies). Same .review format as the
// in-tree file path; just stored content-addressed in the object
// store and reachable by commit SHA.
//
// Backed by internal/git (go-git) — no external git binary needed.

package review

import (
	"errors"
	"fmt"
	"os"

	"gitflower/internal/git"
)

// DefaultNotesRef is the conventional refs/notes/* ref used for
// review bodies.
const DefaultNotesRef = "refs/notes/review"

// ReadNote returns the note body for `sha` from the named notes ref,
// or "" + nil if no note exists. Any other git failure surfaces as
// an error.
func ReadNote(ref, sha string) (string, error) {
	if ref == "" {
		ref = DefaultNotesRef
	}
	repo, err := git.Open("")
	if err != nil {
		return "", err
	}
	body, err := repo.NoteShow(ref, sha)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("git notes show: %w", err)
	}
	return string(body), nil
}

// WriteNote stores `body` as the note for `sha` on the named notes
// ref, overwriting any previous note.
func WriteNote(ref, sha, body string) error {
	if ref == "" {
		ref = DefaultNotesRef
	}
	repo, err := git.Open("")
	if err != nil {
		return err
	}
	if err := repo.NoteAdd(ref, sha, body); err != nil {
		return fmt.Errorf("git notes add: %w", err)
	}
	return nil
}

// LoadFromNote reads the note for `sha` on `ref` and parses it as a
// .review session. Returns (nil, nil) if there's no note for the
// commit. Any parse failure or git failure surfaces as an error.
func LoadFromNote(ref, sha string) (*ReviewSession, error) {
	body, err := ReadNote(ref, sha)
	if err != nil {
		return nil, err
	}
	if body == "" {
		return nil, nil
	}
	sess, err := Parse(body)
	if err != nil {
		return nil, err
	}
	sess.NotesRef = ref
	sess.NotesSHA = sha
	return sess, nil
}

// LastReviewMergeSHA returns the SHA of the most recent merge commit
// reachable from `branch` whose subject starts with "[Review]". Used
// as the default base for a new review: everything since the last
// archived review is the new scope. Returns "" with nil error if no
// such merge exists.
func LastReviewMergeSHA(branch string) (string, error) {
	if branch == "" {
		branch = "HEAD"
	}
	repo, err := git.Open("")
	if err != nil {
		return "", err
	}
	tip, err := repo.Resolve(branch)
	if err != nil {
		return "", fmt.Errorf("git resolve %s: %w", branch, err)
	}
	if tip == git.ZeroHash {
		return "", nil
	}
	return repo.FindMergeWithSubjectPrefix(tip, "[Review]")
}

// ViewCommands returns shell snippets a reader can copy-paste to
// read the review for `sha` on the live notes ref.
func ViewCommands(ref, sha string) []string {
	short := sha
	if len(short) > 12 {
		short = short[:12]
	}
	return viewCommandsFor(fmt.Sprintf("git notes --ref=%s show %s", ref, short))
}

// ArchiveViewCommands is the merge-commit-resident counterpart of
// ViewCommands. It reads the note body straight out of the orphan
// archive commit's tree (`git show <archive>:<sha>`), so the recipe
// works long after the live notes ref has been pruned and on every
// clone that has the merge commit — no `git notes` dependency.
func ArchiveViewCommands(archiveSHA, sha string) []string {
	return viewCommandsFor(fmt.Sprintf("git show %s:%s", archiveSHA, sha))
}

func viewCommandsFor(prefix string) []string {
	return []string{
		prefix,
		prefix + ` | grep -v -E '^### (ReadStart|ReadEnd|SkipStart|SkipEnd) '`,
		prefix + ` | grep -B3 -E '^(# |## (Sources|Verdicts|Changes in |Issue |Commit |File )|### (Comment|Question|Like|Dislike|Verdict))'`,
	}
}

// NoteBlobSHA returns the OID of the blob currently holding the
// note for `sha` on `ref` — i.e. the immutable object that, right
// now, represents this review version. Pin recipes to it (with
// `git show <blob>`) when you want a snapshot that won't drift as
// further edits move the notes ref forward.
func NoteBlobSHA(ref, sha string) (string, error) {
	if ref == "" {
		ref = DefaultNotesRef
	}
	repo, err := git.Open("")
	if err != nil {
		return "", err
	}
	blob, err := repo.NoteBlob(ref, sha)
	if err != nil || blob == git.ZeroHash {
		return "", nil
	}
	return blob.String(), nil
}

// NotesRefTip returns the OID that refs/notes/<ref> currently points
// at, or "" if the ref doesn't exist yet.
func NotesRefTip(ref string) (string, error) {
	if ref == "" {
		ref = DefaultNotesRef
	}
	repo, err := git.Open("")
	if err != nil {
		return "", err
	}
	tip, err := repo.NotesTip(ref)
	if err != nil || tip == git.ZeroHash {
		return "", nil
	}
	return tip.String(), nil
}

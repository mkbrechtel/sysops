// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

package git

import (
	"fmt"
	"os"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Notes layout primer: refs/notes/<ref> → commit → tree. The tree's
// entries are blobs keyed by the annotated commit's SHA. Git uses
// a flat layout for small note sets and may fan the names out to
// `xx/<rest>` (then `xx/yy/<rest>`, etc.) as the set grows. We
// honour the fan-out on read and stay flat on write — git's own
// `git notes` handles both shapes transparently, and `gc` can
// pack/repack at will.

// NoteShow returns the note body for `sha` on the notes ref
// `notesRef` (e.g. "refs/notes/review"), or os.ErrNotExist when no
// note is recorded. Matches `git notes --ref=<ref> show <sha>`.
func (r *Repo) NoteShow(notesRef, sha string) ([]byte, error) {
	rootTree, err := r.notesRootTree(notesRef)
	if err != nil {
		return nil, err
	}
	if rootTree == nil {
		return nil, os.ErrNotExist
	}
	blob, err := findNoteBlob(rootTree, sha)
	if err != nil {
		return nil, err
	}
	if blob == ZeroHash {
		return nil, os.ErrNotExist
	}
	return r.ReadBlob(blob)
}

// NoteBlob returns the blob OID currently holding the note for
// `sha` on `notesRef`, or ZeroHash if there's no note. Used by
// recipes that want a content-addressed pin instead of a ref name.
func (r *Repo) NoteBlob(notesRef, sha string) (Hash, error) {
	rootTree, err := r.notesRootTree(notesRef)
	if err != nil {
		return ZeroHash, err
	}
	if rootTree == nil {
		return ZeroHash, nil
	}
	return findNoteBlob(rootTree, sha)
}

// NoteAdd writes `body` as the note for `sha` on `notesRef`,
// replacing any existing entry. Equivalent to `git notes --ref=<ref>
// add -f -F - <sha>`. Builds a new commit on top of the previous
// notes-ref tip and advances the ref atomically (well: as atomic as
// any non-locking write — fine for single-process use).
func (r *Repo) NoteAdd(notesRef, sha, body string) error {
	blob, err := r.WriteBlob([]byte(body))
	if err != nil {
		return fmt.Errorf("git notes add: blob: %w", err)
	}

	// Existing notes tree (if any) becomes the base for the new
	// tree — preserves every other commit's notes.
	var baseTree Hash
	var prevCommit Hash
	if t, err := r.notesRootTree(notesRef); err != nil {
		return err
	} else if t != nil {
		baseTree = t.Hash
		// Find the commit so we can chain the new note commit
		// onto it (preserves notes-ref history, same as `git
		// notes`).
		if ref, err := r.r.Reference(plumbing.ReferenceName(notesRef), true); err == nil {
			prevCommit = ref.Hash()
		}
	}

	// Write the entry flat — git's `gc` may later pack into a
	// fan-out layout; our reader handles both.
	newTree, err := r.AddFileToTree(baseTree, sha, blob)
	if err != nil {
		return fmt.Errorf("git notes add: tree: %w", err)
	}

	var parents []Hash
	if prevCommit != ZeroHash {
		parents = []Hash{prevCommit}
	}
	msg := "Notes added by 'gitflower'\n"
	commit, err := r.WriteCommit(newTree, parents, msg)
	if err != nil {
		return fmt.Errorf("git notes add: commit: %w", err)
	}
	if err := r.UpdateRef(notesRef, commit); err != nil {
		return fmt.Errorf("git notes add: update-ref %s: %w", notesRef, err)
	}
	return nil
}

// NotesTip returns the OID that `notesRef` points at, or ZeroHash
// if the ref doesn't exist yet.
func (r *Repo) NotesTip(notesRef string) (Hash, error) {
	ref, err := r.r.Reference(plumbing.ReferenceName(notesRef), true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return ZeroHash, nil
		}
		return ZeroHash, fmt.Errorf("git rev-parse %s: %w", notesRef, err)
	}
	return ref.Hash(), nil
}

// notesRootTree returns the tree object at the tip of `notesRef`,
// or (nil, nil) when the ref doesn't exist yet (a clean miss, no
// notes recorded).
func (r *Repo) notesRootTree(notesRef string) (*object.Tree, error) {
	ref, err := r.r.Reference(plumbing.ReferenceName(notesRef), true)
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("notes: read ref %s: %w", notesRef, err)
	}
	c, err := r.r.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("notes: commit %s: %w", ref.Hash(), err)
	}
	t, err := c.Tree()
	if err != nil {
		return nil, fmt.Errorf("notes: tree: %w", err)
	}
	return t, nil
}

// findNoteBlob looks up `sha` inside a notes tree, trying both the
// flat layout (full SHA as the entry name) and the fan-out layout
// (`xx/<rest>`, `xx/yy/<rest>`, etc.). Returns ZeroHash when no
// entry matches.
func findNoteBlob(root *object.Tree, sha string) (Hash, error) {
	// Try flat first — this is the layout `git notes add` produces
	// for small note sets and the one we always write.
	if e := lookupEntry(root, sha); e != nil {
		return e.Hash, nil
	}
	// Fan-out layout: walk every "<n-chars>/..." split git might
	// use. In practice git only fans on 2-char boundaries, but
	// any prefix-split is valid so we just probe a couple.
	for n := 1; n <= 4 && n < len(sha); n++ {
		prefix := sha[:n*2]
		rest := sha[n*2:]
		t := root
		ok := true
		for i := 0; i < n; i++ {
			subName := prefix[i*2 : i*2+2]
			e := lookupEntry(t, subName)
			if e == nil {
				ok = false
				break
			}
			subTree, err := loadSubtree(t, e.Hash)
			if err != nil {
				return ZeroHash, err
			}
			t = subTree
		}
		if !ok {
			continue
		}
		if e := lookupEntry(t, rest); e != nil {
			return e.Hash, nil
		}
	}
	return ZeroHash, nil
}

func lookupEntry(t *object.Tree, name string) *object.TreeEntry {
	for i := range t.Entries {
		if t.Entries[i].Name == name {
			return &t.Entries[i]
		}
	}
	return nil
}

func loadSubtree(parent *object.Tree, hash Hash) (*object.Tree, error) {
	// object.Tree has a Tree method that uses the parent's storer
	// to fetch nested trees; use it so we don't need the *Repo.
	for i := range parent.Entries {
		if parent.Entries[i].Hash == hash {
			t, err := parent.Tree(parent.Entries[i].Name)
			if err != nil {
				return nil, fmt.Errorf("notes: subtree %s: %w", hash, err)
			}
			return t, nil
		}
	}
	return nil, fmt.Errorf("notes: subtree %s not found in parent", hash)
}

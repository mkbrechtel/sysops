// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

package git

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// AddFileToTree returns the OID of a tree that is `base` plus
// `path` → `blob`. Intermediate directories are created if they
// don't exist; an existing entry at `path` is replaced. Used for
// `gitflower review merge --include-file` to splice a single new
// file into HEAD's tree without touching the index.
//
// Each nested level is rewritten recursively (git's CoW tree
// model), so this is O(depth) — fine for our review/<sha>.review
// case.
func (r *Repo) AddFileToTree(base Hash, path string, blob Hash) (Hash, error) {
	parts := splitPath(path)
	if len(parts) == 0 {
		return ZeroHash, fmt.Errorf("AddFileToTree: empty path")
	}
	return r.addRecursive(base, parts, blob)
}

func (r *Repo) addRecursive(treeHash Hash, parts []string, blob Hash) (Hash, error) {
	// Load existing entries (empty when base is ZeroHash — e.g.
	// the directory doesn't exist yet).
	var entries []object.TreeEntry
	if treeHash != ZeroHash {
		t, err := r.r.TreeObject(treeHash)
		if err != nil {
			return ZeroHash, fmt.Errorf("AddFileToTree: load %s: %w", treeHash, err)
		}
		entries = append(entries, t.Entries...)
	}

	name := parts[0]
	if len(parts) == 1 {
		// Leaf: insert or replace the blob entry.
		entries = replaceOrInsert(entries, object.TreeEntry{
			Name: name,
			Mode: filemode.Regular,
			Hash: blob,
		})
		return r.encodeTree(entries)
	}

	// Nested: recurse into the named subtree (or build a fresh
	// one if there's no matching entry yet).
	var childHash Hash
	for _, e := range entries {
		if e.Name == name && e.Mode == filemode.Dir {
			childHash = e.Hash
			break
		}
	}
	newChild, err := r.addRecursive(childHash, parts[1:], blob)
	if err != nil {
		return ZeroHash, err
	}
	entries = replaceOrInsert(entries, object.TreeEntry{
		Name: name,
		Mode: filemode.Dir,
		Hash: newChild,
	})
	return r.encodeTree(entries)
}

func (r *Repo) encodeTree(entries []object.TreeEntry) (Hash, error) {
	// Tree entries are sorted by name (git's own ordering rule).
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	t := &object.Tree{Entries: entries}
	obj := r.r.Storer.NewEncodedObject()
	obj.SetType(plumbing.TreeObject)
	if err := t.Encode(obj); err != nil {
		return ZeroHash, fmt.Errorf("encode tree: %w", err)
	}
	return r.r.Storer.SetEncodedObject(obj)
}

func replaceOrInsert(entries []object.TreeEntry, e object.TreeEntry) []object.TreeEntry {
	for i, cur := range entries {
		if cur.Name == e.Name {
			entries[i] = e
			return entries
		}
	}
	return append(entries, e)
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

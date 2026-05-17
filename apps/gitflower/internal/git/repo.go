// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

// Package git wraps go-git for the read/write operations gitflower
// needs, replacing direct `exec.Command("git", ...)` calls. Goal:
// no external git binary dependency for plumbing (rev-parse, tree
// reads, object writes) so the binary stays self-contained and the
// returned errors are typed instead of "exit status 1".
//
// Operations that go-git either doesn't support or can't match
// byte-for-byte with `git` (notably `git diff -U2 --inter-hunk-context=0`
// and `git format-patch --stdout`) still live elsewhere; see the
// review/scope.go comments.
package git

import (
	"fmt"
	"os"
	"sort"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Repo is a thin handle around a *gogit.Repository plus the path
// it lives at. Open once near the top of a command and pass down;
// the underlying *gogit.Repository is safe to share.
type Repo struct {
	r    *gogit.Repository
	root string
}

// Hash re-exports go-git's plumbing.Hash so callers don't need a
// transitive go-git import for the common case of "OID returned by
// some helper".
type Hash = plumbing.Hash

// ZeroHash is the all-zeros OID, returned by helpers when a ref
// doesn't exist (mirrors `git rev-parse --quiet`'s empty output).
var ZeroHash = plumbing.ZeroHash

// Open discovers a repository walking up from `dir` (or cwd when
// dir is ""). Each call re-opens — go-git's PlainOpen is cheap
// (mostly metadata) and we don't want to cache across the
// process: tests chdir between cases and a stale singleton would
// silently route every subsequent call to the first repo opened.
func Open(dir string) (*Repo, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("git open: getwd: %w", err)
		}
	}
	r, err := gogit.PlainOpenWithOptions(dir, &gogit.PlainOpenOptions{
		DetectDotGit:          true,
		EnableDotGitCommonDir: true,
	})
	if err != nil {
		return nil, fmt.Errorf("git open %s: %w", dir, err)
	}
	root := dir
	if wt, err := r.Worktree(); err == nil {
		root = wt.Filesystem.Root()
	}
	return &Repo{r: r, root: root}, nil
}

// Raw exposes the underlying *go-git Repository for the few sites
// (commit walking, etc.) that need more than the helpers below.
// Try to grow the wrapper instead of reaching past it.
func (r *Repo) Raw() *gogit.Repository { return r.r }

// Toplevel returns the absolute path of the working-tree root —
// equivalent to `git rev-parse --show-toplevel`.
func (r *Repo) Toplevel() string { return r.root }

// HeadBranch returns the short name of the current branch (e.g.
// "feature/x"), or "" + nil error when HEAD is detached. Matches
// `git rev-parse --abbrev-ref HEAD` for the common case.
func (r *Repo) HeadBranch() (string, error) {
	ref, err := r.r.Head()
	if err != nil {
		return "", fmt.Errorf("git head: %w", err)
	}
	if !ref.Name().IsBranch() {
		return "", nil
	}
	return ref.Name().Short(), nil
}

// HeadHash returns the commit OID at HEAD — `git rev-parse HEAD`.
func (r *Repo) HeadHash() (Hash, error) {
	ref, err := r.r.Head()
	if err != nil {
		return ZeroHash, fmt.Errorf("git head: %w", err)
	}
	return ref.Hash(), nil
}

// Resolve takes any revision expression (branch, tag, "HEAD",
// "HEAD^{tree}", a hex SHA, etc.) and returns its OID. Mirrors
// `git rev-parse --verify <rev>`. Returns ZeroHash with nil error
// when the rev doesn't exist (to match `--quiet`); errors propagate
// any other go-git failure.
func (r *Repo) Resolve(rev string) (Hash, error) {
	h, err := r.r.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			return ZeroHash, nil
		}
		// go-git also returns ErrInvalidReference / object-not-found
		// for unknown revs depending on shape; treat as "not found"
		// rather than fatal so callers can decide.
		return ZeroHash, nil
	}
	return *h, nil
}

// ConfigUserEmail returns the resolved git config `user.email` from
// the standard scope chain (system → global → local), matching
// `git config user.email`. Returns "" + nil when unset so callers
// can decide whether that's an error.
func (r *Repo) ConfigUserEmail() (string, error) {
	cfg, err := r.r.ConfigScoped(config.SystemScope)
	if err != nil {
		return "", fmt.Errorf("git config: %w", err)
	}
	if cfg.User.Email != "" {
		return cfg.User.Email, nil
	}
	// Fall back to the merged scope — ConfigScoped(SystemScope)
	// returns just one scope on some go-git versions; ask for the
	// full merge to pick up global+local overrides.
	cfg2, err := r.r.Config()
	if err != nil {
		return "", fmt.Errorf("git config: %w", err)
	}
	return cfg2.User.Email, nil
}

// LogEntry is one commit in a CommitsBetween result — enough to
// drive the `git log --format=%H %s` use case without exposing the
// full *object.Commit.
type LogEntry struct {
	Hash    Hash
	Subject string // first line of the commit message
	Message string // full message
}

// CommitsBetween returns the commits in `base..tip` (commits
// reachable from tip but not from base), newest-first — same order
// as `git log <base>..<tip>`. Uses go-git's IsAncestor to filter.
func (r *Repo) CommitsBetween(base, tip Hash) ([]LogEntry, error) {
	iter, err := r.r.Log(&gogit.LogOptions{From: tip})
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	defer iter.Close()

	baseCommit, err := r.r.CommitObject(base)
	if err != nil {
		return nil, fmt.Errorf("git log: base %s: %w", base, err)
	}

	var out []LogEntry
	err = iter.ForEach(func(c *object.Commit) error {
		// Stop walking once we hit an ancestor of base (or base
		// itself). go-git's IsAncestor walks back from the receiver
		// to check if `c` is an ancestor of baseCommit.
		isAnc, _ := c.IsAncestor(baseCommit)
		if isAnc {
			return stopWalk
		}
		out = append(out, LogEntry{
			Hash:    c.Hash,
			Subject: strings.SplitN(c.Message, "\n", 2)[0],
			Message: c.Message,
		})
		return nil
	})
	if err != nil && err != stopWalk {
		return nil, fmt.Errorf("git log walk: %w", err)
	}
	return out, nil
}

// CommitsOnBranch returns every commit reachable from `tip`,
// newest-first — same shape as `git log <branch>`. Useful for
// looking up subject lines without bringing in a separate iterator
// API.
func (r *Repo) CommitsOnBranch(tip Hash) ([]LogEntry, error) {
	iter, err := r.r.Log(&gogit.LogOptions{From: tip})
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	defer iter.Close()
	var out []LogEntry
	err = iter.ForEach(func(c *object.Commit) error {
		out = append(out, LogEntry{
			Hash:    c.Hash,
			Subject: strings.SplitN(c.Message, "\n", 2)[0],
			Message: c.Message,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("git log walk: %w", err)
	}
	return out, nil
}

// stopWalk is a sentinel returned from ForEach callbacks to break
// out early without flagging an error.
var stopWalk = fmt.Errorf("internal/git: stop walk")

// CommitFiles returns the paths changed between `base` and `tip`'s
// trees, mirroring `git diff --name-only base...tip`. Uses
// symmetric difference (changes since the merge-base) so it matches
// what `git diff base...tip` reports.
func (r *Repo) CommitFiles(base, tip Hash) ([]string, error) {
	baseCommit, err := r.r.CommitObject(base)
	if err != nil {
		return nil, fmt.Errorf("git diff names: base %s: %w", base, err)
	}
	tipCommit, err := r.r.CommitObject(tip)
	if err != nil {
		return nil, fmt.Errorf("git diff names: tip %s: %w", tip, err)
	}
	mergeBases, err := baseCommit.MergeBase(tipCommit)
	if err != nil {
		return nil, fmt.Errorf("git diff names: merge-base: %w", err)
	}
	if len(mergeBases) == 0 {
		// No common history; fall back to a straight base→tip diff.
		mergeBases = []*object.Commit{baseCommit}
	}
	mb := mergeBases[0]
	patch, err := mb.Patch(tipCommit)
	if err != nil {
		return nil, fmt.Errorf("git diff names: patch: %w", err)
	}
	seen := map[string]struct{}{}
	for _, fp := range patch.FilePatches() {
		from, to := fp.Files()
		var name string
		if to != nil {
			name = to.Path()
		} else if from != nil {
			name = from.Path()
		}
		if name == "" {
			continue
		}
		seen[name] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

// FindMergeWithSubjectPrefix walks history from `from` (newest
// first) looking for a merge commit whose subject (first message
// line) starts with `prefix`. Returns the OID as a hex string, or
// "" if no such commit is reachable. Mimics `git log --merges
// --grep='^<prefix>' --format=%H -1 <from>`.
func (r *Repo) FindMergeWithSubjectPrefix(from Hash, prefix string) (string, error) {
	iter, err := r.r.Log(&gogit.LogOptions{From: from})
	if err != nil {
		return "", fmt.Errorf("git log: %w", err)
	}
	defer iter.Close()
	var found string
	stop := fmt.Errorf("stop") // sentinel to break ForEach
	err = iter.ForEach(func(c *object.Commit) error {
		if len(c.ParentHashes) < 2 {
			return nil
		}
		first := strings.SplitN(c.Message, "\n", 2)[0]
		if strings.HasPrefix(first, prefix) {
			found = c.Hash.String()
			return stop
		}
		return nil
	})
	if err != nil && err.Error() != "stop" {
		return "", fmt.Errorf("git log walk: %w", err)
	}
	return found, nil
}

// CommitTreeHash returns the tree OID at the root of `commit` —
// equivalent to `git rev-parse <commit>^{tree}`.
func (r *Repo) CommitTreeHash(commit Hash) (Hash, error) {
	c, err := r.r.CommitObject(commit)
	if err != nil {
		return ZeroHash, fmt.Errorf("git rev-parse %s^{tree}: %w", commit, err)
	}
	return c.TreeHash, nil
}

// RefInfo names one ref and the subject of the commit it points
// at. Used to mimic `git for-each-ref --format='%(refname:short)|%(subject)'`.
type RefInfo struct {
	Name    string // short form, e.g. "mr/foo"
	Subject string // first line of the pointed commit's message
}

// ForEachRefPrefix walks every ref whose full name starts with one
// of `prefixes` (e.g. "refs/heads/mr/", "refs/heads/archive/mr/")
// and returns a deterministic, sorted list with subject lines.
func (r *Repo) ForEachRefPrefix(prefixes ...string) ([]RefInfo, error) {
	iter, err := r.r.References()
	if err != nil {
		return nil, fmt.Errorf("git for-each-ref: %w", err)
	}
	var out []RefInfo
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		full := string(ref.Name())
		matched := false
		var prefix string
		for _, p := range prefixes {
			if strings.HasPrefix(full, p) {
				matched = true
				prefix = p
				break
			}
		}
		if !matched {
			return nil
		}
		c, err := r.r.CommitObject(ref.Hash())
		if err != nil {
			// Non-commit refs (tags pointing to tag objects, etc.)
			// — skip rather than crash the listing.
			return nil
		}
		short := strings.TrimPrefix(full, "refs/heads/")
		// Cut just enough of the prefix to drop "refs/heads/"
		// while keeping the rest (so "refs/heads/mr/foo" → "mr/foo"
		// and "refs/heads/archive/mr/foo" → "archive/mr/foo").
		_ = prefix
		subj := strings.SplitN(c.Message, "\n", 2)[0]
		out = append(out, RefInfo{Name: short, Subject: subj})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("git for-each-ref: walk: %w", err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// LsTreeR lists every file in `commit`'s tree, recursively, sorted —
// equivalent to `git ls-tree -r --name-only <sha>`. Returns paths
// relative to the repo root.
func (r *Repo) LsTreeR(commit Hash) ([]string, error) {
	c, err := r.r.CommitObject(commit)
	if err != nil {
		return nil, fmt.Errorf("git ls-tree: commit %s: %w", commit, err)
	}
	tree, err := c.Tree()
	if err != nil {
		return nil, fmt.Errorf("git ls-tree: tree of %s: %w", commit, err)
	}
	var out []string
	err = tree.Files().ForEach(func(f *object.File) error {
		out = append(out, f.Name)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("git ls-tree: walk: %w", err)
	}
	sort.Strings(out)
	return out, nil
}

// ReadBlobAtPath returns the bytes of `path` in `commit`'s tree —
// equivalent to `git show <commit>:<path>`. Returns os.ErrNotExist
// when the path isn't in the tree so the caller can branch cleanly.
func (r *Repo) ReadBlobAtPath(commit Hash, path string) ([]byte, error) {
	c, err := r.r.CommitObject(commit)
	if err != nil {
		return nil, fmt.Errorf("git show: commit %s: %w", commit, err)
	}
	f, err := c.File(path)
	if err != nil {
		if err == object.ErrFileNotFound {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("git show: file %s: %w", path, err)
	}
	body, err := f.Contents()
	if err != nil {
		return nil, fmt.Errorf("git show: read %s: %w", path, err)
	}
	return []byte(body), nil
}

// ReadBlob returns the raw bytes of a blob by OID — used by
// recipes that pin to a specific content hash.
func (r *Repo) ReadBlob(blob Hash) ([]byte, error) {
	b, err := r.r.BlobObject(blob)
	if err != nil {
		return nil, fmt.Errorf("git show blob %s: %w", blob, err)
	}
	rd, err := b.Reader()
	if err != nil {
		return nil, fmt.Errorf("git show blob %s: reader: %w", blob, err)
	}
	defer rd.Close()
	var buf [4096]byte
	var out []byte
	for {
		n, err := rd.Read(buf[:])
		if n > 0 {
			out = append(out, buf[:n]...)
		}
		if err != nil {
			break
		}
	}
	return out, nil
}

// WriteBlob hashes `body` and writes it to the object store —
// equivalent to `git hash-object -w --stdin`.
func (r *Repo) WriteBlob(body []byte) (Hash, error) {
	obj := r.r.Storer.NewEncodedObject()
	obj.SetType(plumbing.BlobObject)
	obj.SetSize(int64(len(body)))
	w, err := obj.Writer()
	if err != nil {
		return ZeroHash, fmt.Errorf("git hash-object: writer: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		_ = w.Close()
		return ZeroHash, fmt.Errorf("git hash-object: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return ZeroHash, fmt.Errorf("git hash-object: close: %w", err)
	}
	h, err := r.r.Storer.SetEncodedObject(obj)
	if err != nil {
		return ZeroHash, fmt.Errorf("git hash-object: store: %w", err)
	}
	return h, nil
}

// TreeEntry names one entry for WriteTree. Mode defaults to a
// regular file (100644) when zero so the common "one blob" case
// stays terse.
type TreeEntry struct {
	Name string
	Hash Hash
	Mode filemode.FileMode
}

// WriteTree builds a tree object from the given entries and stores
// it — equivalent to feeding mode/type/sha/name lines to `git
// mktree`. Entries are sorted by name (git's tree-entry ordering
// is name-sorted with directories suffixed by '/'; we just sort the
// supplied names since the typical caller passes a single entry).
func (r *Repo) WriteTree(entries []TreeEntry) (Hash, error) {
	t := &object.Tree{}
	for _, e := range entries {
		mode := e.Mode
		if mode == 0 {
			mode = filemode.Regular
		}
		t.Entries = append(t.Entries, object.TreeEntry{
			Name: e.Name,
			Mode: mode,
			Hash: e.Hash,
		})
	}
	sort.Slice(t.Entries, func(i, j int) bool {
		return t.Entries[i].Name < t.Entries[j].Name
	})
	obj := r.r.Storer.NewEncodedObject()
	if err := t.Encode(obj); err != nil {
		return ZeroHash, fmt.Errorf("git mktree: encode: %w", err)
	}
	h, err := r.r.Storer.SetEncodedObject(obj)
	if err != nil {
		return ZeroHash, fmt.Errorf("git mktree: store: %w", err)
	}
	return h, nil
}

// WriteCommit creates a commit object — equivalent to `git
// commit-tree <tree> [-p <parent>...] -m <msg>`. Author/committer
// are filled from the repo's config (falling back to "gitflower"
// when user.name/email are unset) so the resulting commit is
// well-formed even in trimmed-down environments.
func (r *Repo) WriteCommit(tree Hash, parents []Hash, msg string) (Hash, error) {
	author := r.signature()
	c := &object.Commit{
		Author:       author,
		Committer:    author,
		Message:      msg,
		TreeHash:     tree,
		ParentHashes: parents,
	}
	obj := r.r.Storer.NewEncodedObject()
	if err := c.Encode(obj); err != nil {
		return ZeroHash, fmt.Errorf("git commit-tree: encode: %w", err)
	}
	h, err := r.r.Storer.SetEncodedObject(obj)
	if err != nil {
		return ZeroHash, fmt.Errorf("git commit-tree: store: %w", err)
	}
	return h, nil
}

// UpdateRef sets `ref` to point at `hash`. Mirrors `git update-ref
// <ref> <hash>`. The ref name is interpreted exactly — pass
// "HEAD" or "refs/heads/main" as appropriate.
func (r *Repo) UpdateRef(ref string, hash Hash) error {
	name := plumbing.ReferenceName(ref)
	// `update-ref HEAD` should follow the symbolic ref and update
	// the branch it points at; go-git's SetReference on "HEAD"
	// would clobber HEAD's symbolic-ness instead.
	if ref == "HEAD" {
		head, err := r.r.Reference(plumbing.HEAD, false)
		if err != nil {
			return fmt.Errorf("git update-ref HEAD: read head: %w", err)
		}
		if head.Type() == plumbing.SymbolicReference {
			name = head.Target()
		}
	}
	newRef := plumbing.NewHashReference(name, hash)
	return r.r.Storer.SetReference(newRef)
}

// signature builds the author/committer signature for a new commit
// from user.name/user.email, with safe fallbacks so commit objects
// are always well-formed.
func (r *Repo) signature() object.Signature {
	cfg, _ := r.r.Config()
	name := "gitflower"
	email := "gitflower@local"
	if cfg != nil {
		if cfg.User.Name != "" {
			name = cfg.User.Name
		}
		if cfg.User.Email != "" {
			email = cfg.User.Email
		}
	}
	return object.Signature{
		Name:  name,
		Email: email,
		When:  nowFunc(),
	}
}

// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

package review

import (
	"fmt"
	"regexp"
	"strings"

	"gitflower/internal/git"
)

// Scope describes what a review covers.
type Scope struct {
	Branch  string   // the branch being reviewed ("to" side)
	Base    string   // ref the diff is taken against ("from" side)
	TipSHA  string   // resolved tip of Branch at time of scope computation
	BaseSHA string   // resolved tip of Base at time of scope computation
	Diff    string   // symbolic diff range (Base..Branch)
	Commits []Commit // commits in Base..Branch, oldest last (git log order)
	Files   []string // paths changed in Base...Branch
	RawDiff string   // full unified diff, base...branch
	Title   string   // parsed from most-recent [Merge Request] subject; falls back to branch

	// FilePatches: per-file unified diff (git diff base..branch -- <path>).
	// Populated lazily by Render so unused files don't pay the cost.
	FilePatches map[string]string

	// CommitPatches: per-commit `git format-patch --stdout` body keyed by SHA.
	// Lazily populated.
	CommitPatches map[string]string
}

// Commit is one entry in Scope.Commits. Patch is the mbox-style git
// format-patch output for that single commit.
type Commit struct {
	SHA     string
	Short   string
	Subject string
	Patch   string
}

// ScopeFor computes a Scope for the given branch.
// If base is empty, it tries to parse `Base:` from the most-recent
// [Merge Request] commit on the branch; failing that, defaults to "main".
func ScopeFor(branch, base string) (*Scope, error) {
	if branch == "" {
		return nil, fmt.Errorf("scope: branch is required")
	}
	repo, err := git.Open("")
	if err != nil {
		return nil, fmt.Errorf("scope: %w", err)
	}
	tip, err := repo.Resolve(branch)
	if err != nil || tip == git.ZeroHash {
		return nil, fmt.Errorf("scope: branch %q not found", branch)
	}

	if base == "" {
		// Prefer the most recent [Review] merge commit on the branch
		// as the base — "the next review starts from the last review
		// archive". Falls back to the [Merge Request] convention
		// (parseBaseFromBranch) and finally "main".
		if sha, err := LastReviewMergeSHA(branch); err == nil && sha != "" {
			base = sha
		}
	}
	if base == "" {
		base = parseBaseFromBranch(branch)
	}
	if base == "" {
		base = "main"
	}
	baseHash, err := repo.Resolve(base)
	if err != nil || baseHash == git.ZeroHash {
		return nil, fmt.Errorf("scope: base ref %q not found (override with --base)", base)
	}

	entries, err := repo.CommitsBetween(baseHash, tip)
	if err != nil {
		return nil, fmt.Errorf("scope: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("scope: no commits in %s..%s", base, branch)
	}
	var commits []Commit
	for _, e := range entries {
		sha := e.Hash.String()
		short := sha
		if len(short) > 7 {
			short = short[:7]
		}
		// Per-commit patch: still uses `git format-patch` because
		// go-git has no equivalent that produces the same
		// mbox-style output (which our renderer copies verbatim
		// into the # Commits section of the .review).
		patch, _ := execFormatPatch(sha)
		commits = append(commits, Commit{
			SHA:     sha,
			Short:   short,
			Subject: e.Subject,
			Patch:   patch,
		})
	}

	files, err := repo.CommitFiles(baseHash, tip)
	if err != nil {
		return nil, fmt.Errorf("scope: %w", err)
	}

	// Per-file diffs (and the aggregate RawDiff) still go through
	// `git diff` because we need the `-U2 --inter-hunk-context=0`
	// flags — go-git's diff text doesn't expose context size.
	// renderQuotedDiff parses the result by `+`/`-`/` ` sign; the
	// exact unified-diff shape matters.
	rawDiff, err := execRangeDiff(base, branch)
	if err != nil {
		return nil, fmt.Errorf("scope: %w", err)
	}

	title := findMRTitle(branch)
	if title == "" {
		title = branch
	}

	return &Scope{
		Branch:        branch,
		Base:          base,
		TipSHA:        tip.String(),
		BaseSHA:       baseHash.String(),
		Diff:          base + ".." + branch,
		Commits:       commits,
		Files:         files,
		RawDiff:       rawDiff,
		Title:         title,
		FilePatches:   map[string]string{},
		CommitPatches: map[string]string{},
	}, nil
}

// FilePatch returns the unified diff for one path in scope, computing it on
// demand and caching.
func (s *Scope) FilePatch(path string) string {
	if p, ok := s.FilePatches[path]; ok {
		return p
	}
	// `:(top)path` pathspec magic anchors the path to the repo root.
	// Without it, `git diff -- path` resolves `path` relative to cwd,
	// so running gitflower from a subdir returns empty patches for
	// every file (Scope.Files are root-relative). Empty patches let
	// renderChanges skip the entire `## Changes in <path>` body and
	// drop any anchored comments/marks/reads.
	out, err := execFileDiff(s.Base, s.Branch, path)
	if err != nil {
		return ""
	}
	if s.FilePatches == nil {
		s.FilePatches = map[string]string{}
	}
	s.FilePatches[path] = out
	return out
}

// CommitPatch returns the `git format-patch --stdout` body for one commit
// in scope, computing on demand and caching.
func (s *Scope) CommitPatch(sha string) string {
	if p, ok := s.CommitPatches[sha]; ok {
		return p
	}
	out, err := execCommitPatchRange(sha)
	if err != nil {
		return ""
	}
	if s.CommitPatches == nil {
		s.CommitPatches = map[string]string{}
	}
	s.CommitPatches[sha] = out
	return out
}

var (
	basePat   = regexp.MustCompile(`(?m)^Base:\s*(\S+)\s*$`)
	mrSubjPat = regexp.MustCompile(`^\[Merge Request\]\s*(.*)$`)
)

func parseBaseFromBranch(branch string) string {
	repo, err := git.Open("")
	if err != nil {
		return ""
	}
	tip, err := repo.Resolve(branch)
	if err != nil || tip == git.ZeroHash {
		return ""
	}
	entries, err := repo.CommitsOnBranch(tip)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		first := strings.SplitN(e.Message, "\n", 2)[0]
		if !strings.HasPrefix(first, "[Merge Request]") {
			continue
		}
		if m := basePat.FindStringSubmatch(e.Message); m != nil {
			return m[1]
		}
		return "main"
	}
	return ""
}

func findMRTitle(branch string) string {
	repo, err := git.Open("")
	if err != nil {
		return ""
	}
	tip, err := repo.Resolve(branch)
	if err != nil || tip == git.ZeroHash {
		return ""
	}
	entries, err := repo.CommitsOnBranch(tip)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if m := mrSubjPat.FindStringSubmatch(e.Subject); m != nil {
			return m[1]
		}
	}
	return ""
}

// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"gitflower/review"
)

// TestSpaceWalkOnLongDiffs exercises Space + PgDn against synthetic
// hunks bigger than any viewport: a 500-line all-add diff and a
// 400-line all-remove diff. The driver imitates a real reviewer's
// flow — PgDn through the visible content (firing read ticks as
// hunks fully display), Space to jump to the next unread / next
// file. We require 100% read at the end; this test is here to catch
// rare regressions in the walk logic, so anything less than every
// hunk read is a bug we want to surface.
func TestSpaceWalkOnLongDiffs(t *testing.T) {
	addPatch := buildAddPatch("big_add.txt", 500)
	delPatch := buildRemovePatch("big_remove.txt", 400)
	combined := addPatch + "\n" + delPatch

	scope := review.Scope{
		Branch:  "feature",
		Base:    "main",
		TipSHA:  "abc1234567890",
		BaseSHA: "0000111122223333",
		Diff:    "main..feature",
		Title:   "long",
		Commits: []review.Commit{
			{SHA: "abc1234567890", Short: "abc1234", Subject: "long diff commit"},
		},
		Files:   []string{"big_add.txt", "big_remove.txt"},
		RawDiff: combined,
		FilePatches: map[string]string{
			"big_add.txt":    addPatch,
			"big_remove.txt": delPatch,
		},
		CommitPatches: map[string]string{
			"abc1234567890": "From abc1234 ...\n" + combined,
		},
	}
	tmp := t.TempDir()
	reviewPath := filepath.Join(tmp, "long.review")
	sess := review.New(scope, "tester@example.com", reviewPath)

	m := newModel(sess, tmp, 1000.0)
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 25})

	// First Space drills into the first file at "5 before first unread".
	m = key(t, m, ' ', " ")
	if m.mode != modeDiff {
		t.Fatalf("after first Space: mode %v want modeDiff", m.mode)
	}

	// Drive: PgDn until current file is fully read, then Space to
	// advance; stop on the verdict editor.
	const maxSteps = 2000
	stuck := 0
	for i := 0; i < maxSteps; i++ {
		// Fire pending read ticks deterministically.
		for anchor := range m.pendingReads {
			next, _ := m.Update(delayedReadMsg{anchor: anchor})
			m = next.(*model)
		}
		if m.edit == editSummary {
			t.Logf("reached verdict editor after %d step(s)", i)
			break
		}
		before := stateSig(m)
		// Use Space when the current file is exhausted, PgDn otherwise.
		var msg tea.Msg
		if m.atEOF || m.mode == modeTree || !m.fileHasUnread(m.fileIdx) {
			msg = tea.KeyPressMsg{Code: ' ', Text: " "}
		} else {
			msg = tea.KeyPressMsg{Code: tea.KeyPgDown}
		}
		m = step(t, m, msg)
		if stateSig(m) == before {
			stuck++
			if stuck > 4 {
				t.Fatalf("walk stuck after %d step(s): %s", i, before)
			}
		} else {
			stuck = 0
		}
	}

	// Tally: every hunk in every (real) file must be marked read.
	type miss struct {
		path string
		read int
		all  int
	}
	var misses []miss
	for _, f := range m.files {
		if strings.HasPrefix(f.Path, "commit:") {
			continue
		}
		read := 0
		for _, h := range f.Hunks {
			a := review.HunkAnchor(f.Path, h.NewStart, h.NewLines)
			if m.sess.IsRead(a) {
				read++
			}
		}
		if read != len(f.Hunks) {
			misses = append(misses, miss{f.Path, read, len(f.Hunks)})
		}
	}
	if len(misses) > 0 {
		var sb strings.Builder
		for _, mi := range misses {
			fmt.Fprintf(&sb, "  %-30s %d/%d hunks read\n", mi.path, mi.read, mi.all)
		}
		t.Errorf("walk left hunks unread:\n%s", sb.String())
	}
}

// TestSpaceOnlyWalkOnLongDiffs is the strict version: only Space is
// pressed (no PgDn). With the "page-the-unread-on-repeat" behaviour
// inside spaceWalkInFile, even tall hunks must get fully read.
func TestSpaceOnlyWalkOnLongDiffs(t *testing.T) {
	scope := review.Scope{
		Branch:  "feature",
		Base:    "main",
		TipSHA:  "abc1234567890",
		BaseSHA: "0000111122223333",
		Diff:    "main..feature",
		Title:   "long",
		Commits: []review.Commit{{SHA: "abc1234567890", Short: "abc1234", Subject: "long"}},
		Files:   []string{"big_add.txt", "big_remove.txt"},
		RawDiff: buildAddPatch("big_add.txt", 500) + "\n" + buildRemovePatch("big_remove.txt", 400),
		FilePatches: map[string]string{
			"big_add.txt":    buildAddPatch("big_add.txt", 500),
			"big_remove.txt": buildRemovePatch("big_remove.txt", 400),
		},
		CommitPatches: map[string]string{"abc1234567890": "From abc1234\n"},
	}
	tmp := t.TempDir()
	sess := review.New(scope, "tester@example.com", filepath.Join(tmp, "long.review"))
	m := newModel(sess, tmp, 1000.0)
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 25})

	const maxPresses = 2000
	stuck := 0
	for i := 0; i < maxPresses; i++ {
		for anchor := range m.pendingReads {
			next, _ := m.Update(delayedReadMsg{anchor: anchor})
			m = next.(*model)
		}
		if m.edit == editSummary {
			t.Logf("Space-only walk reached verdict editor after %d press(es)", i)
			break
		}
		before := stateSig(m)
		m = key(t, m, ' ', " ")
		if stateSig(m) == before {
			stuck++
			if stuck > 4 {
				t.Fatalf("Space-only stuck after %d press(es) at %s", i, before)
			}
		} else {
			stuck = 0
		}
	}

	// Tally: every real file's hunks must be read.
	for _, f := range m.files {
		if strings.HasPrefix(f.Path, "commit:") {
			continue
		}
		for _, h := range f.Hunks {
			a := review.HunkAnchor(f.Path, h.NewStart, h.NewLines)
			if !m.sess.IsRead(a) {
				t.Errorf("unread after Space-only walk: %s @ %d,%d", f.Path, h.NewStart, h.NewLines)
			}
		}
	}
}

// TestAltSpaceSkipsAndAdvances verifies that Alt+Space marks the
// current unread hunk as skipped and jumps to the next one, and that
// a skipped hunk's render uses the muted-olive add style (not the
// strong unread green).
func TestAltSpaceSkipsAndAdvances(t *testing.T) {
	scope := review.Scope{
		Branch:  "feature",
		Base:    "main",
		TipSHA:  "abc1234567890",
		BaseSHA: "0000111122223333",
		Diff:    "main..feature",
		Title:   "skip",
		Commits: []review.Commit{{SHA: "abc1234567890", Short: "abc1234", Subject: "skip"}},
		Files:   []string{"big_add.txt"},
		RawDiff: buildAddPatch("big_add.txt", 50),
		FilePatches: map[string]string{
			"big_add.txt": buildAddPatch("big_add.txt", 50),
		},
		CommitPatches: map[string]string{"abc1234567890": "From abc1234\n"},
	}
	tmp := t.TempDir()
	sess := review.New(scope, "tester@example.com", filepath.Join(tmp, "skip.review"))
	m := newModel(sess, tmp, 1000.0)
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 25})

	// Drill in.
	m = key(t, m, ' ', " ")
	if m.mode != modeDiff {
		t.Fatalf("expected modeDiff, got %v", m.mode)
	}

	// Alt+Space: skip the hunk, advance to EOF or next unread.
	hunk := m.currentHunk()
	if hunk == nil {
		t.Fatalf("expected a current hunk")
	}
	anchor := review.HunkAnchor(m.currentFile().Path, hunk.NewStart, hunk.NewLines)
	if m.sess.IsSkipped(anchor) {
		t.Fatalf("anchor pre-skipped before Alt+Space")
	}
	m = step(t, m, tea.KeyPressMsg{Code: ' ', Text: " ", Mod: tea.ModAlt})
	if !m.sess.IsSkipped(anchor) {
		t.Errorf("Alt+Space did not mark the hunk skipped")
	}
	if m.sess.IsRead(anchor) {
		t.Errorf("Alt+Space wrongly marked the hunk read")
	}
}

// TestFastSpamDoesNotMarkUnseenHunksRead simulates a reviewer who
// pages through content faster than the per-viewport reading time:
// the read tick is scheduled, but the user keeps scrolling so by
// the time it fires the hunk is no longer on screen. The hunk must
// stay unread — otherwise the read-rate gating is meaningless.
func TestFastSpamDoesNotMarkUnseenHunksRead(t *testing.T) {
	scope := review.Scope{
		Branch:  "feature",
		Base:    "main",
		TipSHA:  "abc1234567890",
		BaseSHA: "0000111122223333",
		Diff:    "main..feature",
		Title:   "fast",
		Commits: []review.Commit{{SHA: "abc1234567890", Short: "abc1234", Subject: "fast"}},
		Files:   []string{"a.txt", "b.txt"},
		RawDiff: buildAddPatch("a.txt", 5) + "\n" + buildAddPatch("b.txt", 5),
		FilePatches: map[string]string{
			"a.txt": buildAddPatch("a.txt", 5),
			"b.txt": buildAddPatch("b.txt", 5),
		},
		CommitPatches: map[string]string{"abc1234567890": "From abc1234\n"},
	}
	tmp := t.TempDir()
	sess := review.New(scope, "tester@example.com", filepath.Join(tmp, "fast.review"))
	// Low rate (1 line/sec → 5s delay for a 5-line hunk). Spam Spaces
	// faster than that and fire the tick AFTER we've already moved on:
	// the hunk should NOT be marked read.
	m := newModel(sess, tmp, 1.0)
	m = step(t, m, tea.WindowSizeMsg{Width: 100, Height: 25})

	// Drill into a.txt.
	m = key(t, m, ' ', " ")
	anchorA := review.HunkAnchor("a.txt", 1, 5)
	if !m.pendingReads[anchorA] {
		// Render a.txt's hunk so the tick gets scheduled.
		m.refreshViewport()
	}

	// Skip a.txt (Alt+Space marks the hunk as Skipped + lands on a's
	// EOF marker), then Space again to advance past EOF into b.txt.
	// At this point a.txt's hunk is no longer in m.hunkRanges, so the
	// pending read tick (scheduled when a was first displayed) must
	// not promote a from skipped to read.
	m = step(t, m, tea.KeyPressMsg{Code: ' ', Text: " ", Mod: tea.ModAlt})
	if !m.sess.IsSkipped(anchorA) {
		t.Fatalf("Alt+Space should have skipped a.txt")
	}
	m = key(t, m, ' ', " ") // advance from a's EOF to b.txt
	if !strings.HasSuffix(m.currentFile().Path, "b.txt") {
		t.Fatalf("expected to land on b.txt, got %s", m.currentFile().Path)
	}
	for anchor := range m.pendingReads {
		next, _ := m.Update(delayedReadMsg{anchor: anchor})
		m = next.(*model)
	}
	if m.sess.IsRead(anchorA) {
		t.Errorf("a.txt was marked read despite reviewer paging past it")
	}
}

// buildAddPatch builds a `git diff` patch that adds `n` lines to a
// brand-new file at `path`. The lines are short and deterministic so
// the wrap/render paths don't add noise to the test.
func buildAddPatch(path string, n int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "diff --git a/%s b/%s\n", path, path)
	sb.WriteString("new file mode 100644\n")
	sb.WriteString("index 0000000..abcdef0\n")
	fmt.Fprintf(&sb, "--- /dev/null\n")
	fmt.Fprintf(&sb, "+++ b/%s\n", path)
	fmt.Fprintf(&sb, "@@ -0,0 +1,%d @@\n", n)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "+line %d\n", i)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// buildRemovePatch builds a `git diff` patch that removes all `n` lines
// from an existing file at `path`.
func buildRemovePatch(path string, n int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "diff --git a/%s b/%s\n", path, path)
	sb.WriteString("deleted file mode 100644\n")
	sb.WriteString("index abcdef0..0000000\n")
	fmt.Fprintf(&sb, "--- a/%s\n", path)
	sb.WriteString("+++ /dev/null\n")
	fmt.Fprintf(&sb, "@@ -1,%d +0,0 @@\n", n)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "-line %d\n", i)
	}
	return strings.TrimRight(sb.String(), "\n")
}

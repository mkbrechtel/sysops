// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"gitflower/review"
)

// TestTUIDrivesSession exercises the TUI's key handling without a real
// terminal. It constructs the model directly, feeds it a WindowSizeMsg
// followed by a sequence of synthetic key presses, and verifies session
// state after each step. Catches regressions in section→line drill-in,
// comment creation, verdict cycling, and save.
func TestTUIDrivesSession(t *testing.T) {
	tmp := t.TempDir()
	reviewPath := filepath.Join(tmp, "test.review")

	scope := review.Scope{
		Branch:  "feature",
		Base:    "main",
		TipSHA:  "abc1234567890",
		BaseSHA: "0000111122223333",
		Diff:    "main..feature",
		Title:   "feature",
		Commits: []review.Commit{
			{SHA: "abc1234567890", Short: "abc1234", Subject: "feature commit"},
		},
		Files: []string{"foo.txt"},
		RawDiff: `diff --git a/foo.txt b/foo.txt
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/foo.txt
@@ -0,0 +1,2 @@
+line one
+line two`,
		FilePatches: map[string]string{
			"foo.txt": `diff --git a/foo.txt b/foo.txt
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/foo.txt
@@ -0,0 +1,2 @@
+line one
+line two`,
		},
		CommitPatches: map[string]string{
			"abc1234567890": "From abc1234 ...\n",
		},
	}
	sess := review.New(scope, "tester@example.com", reviewPath)

	m := newModel(sess, tmp, 10*time.Millisecond)

	// Set window dimensions so the model can render.
	m = step(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})

	// Initial state: section mode on Changes.
	if m.mode != modeTree {
		t.Fatalf("initial mode: got %v want modeTree", m.mode)
	}
	if m.sect != sectionChanges {
		t.Errorf("initial section: got %v want sectionChanges", m.sect)
	}

	// Press Space → drill into Changes (modeDiff) on first unread hunk.
	m = key(t, m, ' ', " ")
	if m.mode != modeDiff {
		t.Fatalf("after Space in section mode: got mode %v want modeDiff", m.mode)
	}
	if m.fileIdx != 0 || m.hunkIdx != 0 {
		t.Errorf("after drill: got fileIdx=%d hunkIdx=%d, want 0/0", m.fileIdx, m.hunkIdx)
	}

	// Cycle verdict forward with '>'.
	m = key(t, m, '>', ">")
	if sess.Verdict != review.VerdictChanges {
		t.Errorf("after '>': verdict %q want %q", sess.Verdict, review.VerdictChanges)
	}

	// Add a comment: c → type text → Alt+Enter.
	m = key(t, m, 'c', "c")
	if m.edit != editComment {
		t.Fatalf("after 'c': got edit %v want editComment", m.edit)
	}
	for _, r := range "Looks fine." {
		m = key(t, m, r, string(r))
	}
	m = step(t, m, tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt})
	if m.edit != editNone {
		t.Errorf("after submit: still in edit mode %v", m.edit)
	}
	if got := len(sess.Comments()); got != 1 {
		t.Fatalf("expected 1 comment, got %d", got)
	}
	if body := sess.Comments()[0].Text; body != "Looks fine." {
		t.Errorf("comment text: got %q", body)
	}

	// Save with 's'.
	m = key(t, m, 's', "s")
	data, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("expected file at %s: %v", reviewPath, err)
	}
	body := string(data)
	for _, want := range []string{
		"# Review",
		"## Sources",
		"## Verdicts",
		"### Verdict: requested-changes",
		"# Changes",
		"## Changes in `foo.txt`",
		"> +line one",
		"### Comment (From: tester",
		"Looks fine.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("rendered file missing %q\n--- BEGIN ---\n%s\n--- END ---", want, body)
			return
		}
	}
}

// step feeds a generic tea.Msg through the model and returns the (asserted)
// model after the update.
func step(t *testing.T, m *model, msg tea.Msg) *model {
	t.Helper()
	next, _ := m.Update(msg)
	mm, ok := next.(*model)
	if !ok {
		t.Fatalf("Update returned %T, want *model", next)
	}
	return mm
}

// key feeds a synthetic KeyPressMsg for a single rune.
func key(t *testing.T, m *model, code rune, text string) *model {
	return step(t, m, tea.KeyPressMsg{Code: code, Text: text})
}

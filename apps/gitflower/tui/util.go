// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

// Small string / path / arithmetic helpers used across the package.

package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"gitflower/review"
)

func truncate(s string, w int) string {
	if w < 1 {
		return ""
	}
	if len(s) <= w {
		return s
	}
	if w < 4 {
		return s[:w]
	}
	return "…" + s[len(s)-w+1:]
}

func atoi(s string) (int, bool) {
	n := 0
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// pathDir returns the directory portion of `p`, or "" for top-level.
func pathDir(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[:i]
	}
	return ""
}

// pathInDir reports whether path `p` lives under directory `dir` (any
// depth). dir == "" means root.
func pathInDir(p, dir string) bool {
	if dir == "" {
		return true
	}
	return strings.HasPrefix(p, dir+"/")
}

// wrapDiffText hard-wraps a single diff line's payload (no sign, no gutter)
// at `width`. ansi.Hardwrap preserves any escape codes embedded in `s`.
// Tabs are expanded to spaces first because Hardwrap counts a tab as
// width 1 while the terminal expands it to the next 8-column stop —
// without expansion, lines with tabs slip past `width`, the terminal
// re-wraps them itself, and our hanging-indent never gets applied.
func wrapDiffText(s string, width int) []string {
	if width < 1 {
		return []string{s}
	}
	s = expandTabs(s, 8)
	wrapped := ansi.Hardwrap(s, width, false)
	return strings.Split(wrapped, "\n")
}

// expandTabs replaces each tab with enough spaces to advance to the next
// `tabSize`-column tab stop. It only inspects ASCII so it stays cheap;
// any embedded ANSI escape sequences are passed through unchanged but
// counted as visible — diff payload doesn't normally contain escapes.
func expandTabs(s string, tabSize int) string {
	if !strings.ContainsRune(s, '\t') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	col := 0
	for _, r := range s {
		if r == '\t' {
			pad := tabSize - (col % tabSize)
			for i := 0; i < pad; i++ {
				b.WriteByte(' ')
			}
			col += pad
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

// commitMessageHeader is the sentinel hunk header used for the
// synthetic commit-message hunk. renderFileDiff special-cases this
// header to render its (LineAdd) lines as plain prose: no "+ " sign,
// no green background. The lines are kept as LineAdd so the walk
// and per-line read tracking still apply.
const commitMessageHeader = "@@ commit message @@"

// commitMessageHunk extracts the header + commit-message portion of
// a `git format-patch --stdout` body and returns it as a synthetic
// review.Hunk. Lines are emitted as LineAdd (so the walk treats them
// as reviewable), but the hunk header is the sentinel that tells the
// renderer to draw them as prose, not a diff. Returns an empty hunk
// if the patch has no extractable preamble.
func commitMessageHunk(patch string) review.Hunk {
	lines := strings.Split(patch, "\n")
	var preamble []string
	for _, ln := range lines {
		if strings.HasPrefix(ln, "diff --git ") {
			break
		}
		preamble = append(preamble, ln)
	}
	for len(preamble) > 0 {
		last := preamble[len(preamble)-1]
		if last == "" || last == "---" ||
			strings.HasPrefix(last, " ") ||
			strings.HasPrefix(last, "---") {
			preamble = preamble[:len(preamble)-1]
			continue
		}
		break
	}
	h := review.Hunk{
		Header:   commitMessageHeader,
		OldStart: 0, OldLines: 0,
		NewStart: 1, NewLines: len(preamble),
	}
	for _, ln := range preamble {
		h.Lines = append(h.Lines, review.Line{Kind: review.LineAdd, Text: ln})
	}
	return h
}

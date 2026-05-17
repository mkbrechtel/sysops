// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

// The last two `git` shellings-out in the codebase. Both produce
// text whose exact shape gitflower's format pipeline depends on:
//
//   - `git diff -U2 --inter-hunk-context=0 base...branch` — fed
//     into renderQuotedDiff which parses unified-diff lines by
//     their leading sign character. go-git's Patch.String() doesn't
//     expose context size or inter-hunk merging, and writing a
//     custom formatter on top of its diff chunks would re-do work
//     git already does well.
//
//   - `git format-patch --stdout` — mbox-style commit dump
//     (`From <SHA> Mon Sep 17 …`, `From:`, `Date:`, `Subject:`,
//     body, `---`, diffstat, diff). The .review's # Commits section
//     embeds this verbatim; replicating git's exact output is
//     unnecessary churn.
//
// Both are read-only and run synchronously, so this is a small
// well-contained exception to the "no shelling out" rule.

package review

import (
	"fmt"
	"os/exec"
	"strings"
)

func execRangeDiff(base, branch string) (string, error) {
	return gitOut("diff", "--no-color",
		"-U2", "--inter-hunk-context=0",
		base+"..."+branch)
}

func execFileDiff(base, branch, path string) (string, error) {
	return gitOut("diff", "--no-color",
		"-U2", "--inter-hunk-context=0",
		base+".."+branch, "--", ":(top)"+path)
}

func execFormatPatch(sha string) (string, error) {
	return gitOut("format-patch", "-1", "--stdout",
		"--no-signature", "--no-color", sha)
}

func execCommitPatchRange(sha string) (string, error) {
	return gitOut("format-patch", "--stdout", sha+"^.."+sha)
}

func gitOut(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s",
				strings.Join(args, " "),
				strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w",
			strings.Join(args, " "), err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

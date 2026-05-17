// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

// Thin wrappers around the repo for the TUI's needs. Backed by
// the internal/git package (go-git) — no external git binary.

package tui

import (
	"strings"

	"gitflower/internal/git"
)

func gitRoot() (string, error) {
	r, err := git.Open("")
	if err != nil {
		return "", err
	}
	return r.Toplevel(), nil
}

func gitTreeFiles(sha string) ([]string, error) {
	r, err := git.Open("")
	if err != nil {
		return nil, err
	}
	h, err := r.Resolve(sha)
	if err != nil || h == git.ZeroHash {
		return nil, err
	}
	return r.LsTreeR(h)
}

func gitFileLines(sha, path string) ([]string, error) {
	r, err := git.Open("")
	if err != nil {
		return nil, err
	}
	h, err := r.Resolve(sha)
	if err != nil || h == git.ZeroHash {
		return nil, err
	}
	body, err := r.ReadBlobAtPath(h, path)
	if err != nil {
		return nil, err
	}
	s := strings.TrimRight(string(body), "\n")
	if s == "" {
		return nil, nil
	}
	return strings.Split(s, "\n"), nil
}

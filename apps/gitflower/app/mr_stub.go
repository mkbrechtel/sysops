// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

//go:build !with_mrs

// Default build: `gitflower mr ...` is disabled. Rebuild with
// `go build -tags with_mrs` to enable the merge-request subcommands.
// Stub keeps the top-level dispatcher in App() compilable without
// pulling the MR machinery into a stripped-down binary.

package app

import (
	"fmt"
	"io"
)

func cmdMR(_ []string, _ io.Writer, stderr io.Writer) int {
	fmt.Fprintln(stderr, "mr: feature not enabled in this build")
	fmt.Fprintln(stderr, "rebuild with: go build -tags with_mrs")
	return 2
}

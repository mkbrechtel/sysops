// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

//go:build !with_review_merge

// Default build: `gitflower review merge` is disabled. Rebuild with
// `go build -tags with_review_merge` (or `make build BUILDTAGS=...`)
// to enable it. The stub here keeps cmdReview compilable without
// pulling the merge-only dependencies into a stripped-down binary.

package app

import (
	"fmt"
	"io"
)

func cmdReviewMerge(_ []string, _ io.Writer, stderr io.Writer) int {
	fmt.Fprintln(stderr, "review merge: feature not enabled in this build")
	fmt.Fprintln(stderr, "rebuild with: go build -tags with_review_merge")
	return 2
}

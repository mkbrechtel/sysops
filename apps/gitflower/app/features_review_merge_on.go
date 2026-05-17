// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

//go:build with_review_merge

package app

// WithReviewMerge is true when the binary was built with
// `-tags with_review_merge`. Used by the usage printer (and any
// other branch that wants to behave differently under the flag).
// Dead-code-eliminated together with its `if` branch in builds
// where the flag is off.
const WithReviewMerge = true

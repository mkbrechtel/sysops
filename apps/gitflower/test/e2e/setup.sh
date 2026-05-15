#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
# SPDX-License-Identifier: EUPL-1.2
#
# Builds a deterministic test repo at $1 (default /tmp/gitflower-e2e-repo).
# A handful of small commits on a feature branch off `main`, so gitflower's
# `review` command has something realistic but tiny to chew on.
#
# Idempotent: blows away the destination and rebuilds.
#
# Output: prints the path to the rebuilt repo on stdout.

set -euo pipefail

dest="${1:-/tmp/gitflower-e2e-repo}"
rm -rf "$dest"
mkdir -p "$dest"
cd "$dest"

# Fixed identity + dates so the resulting commit SHAs and metadata are
# byte-for-byte stable across runs.
export GIT_AUTHOR_NAME="Test Author"      GIT_AUTHOR_EMAIL="test@example.com"
export GIT_COMMITTER_NAME="Test Author"   GIT_COMMITTER_EMAIL="test@example.com"
export GIT_AUTHOR_DATE="2026-01-01T00:00:00Z"
export GIT_COMMITTER_DATE="$GIT_AUTHOR_DATE"

git init -q -b main
git config commit.gpgsign false

# ---- main: initial state ----------------------------------------------------
cat > README.md <<'EOF'
# Test project

This is a small fixture used by gitflower's end-to-end review tests.
EOF
cat > greet.go <<'EOF'
package greet

func Hello(name string) string {
	return "Hello, " + name + "!"
}
EOF
git add README.md greet.go
git commit -q -m "Initial commit"

# ---- feature branch: three commits ------------------------------------------
git checkout -q -b feature

# Commit 1: add a test
GIT_AUTHOR_DATE="2026-01-02T10:00:00Z" GIT_COMMITTER_DATE="2026-01-02T10:00:00Z" \
cat > greet_test.go <<'EOF'
package greet

import "testing"

func TestHello(t *testing.T) {
	got := Hello("world")
	if got != "Hello, world!" {
		t.Fatalf("got %q", got)
	}
}
EOF
git add greet_test.go
GIT_AUTHOR_DATE="2026-01-02T10:00:00Z" GIT_COMMITTER_DATE="2026-01-02T10:00:00Z" \
git commit -q -m "Add basic greet test"

# Commit 2: handle empty input
GIT_AUTHOR_DATE="2026-01-02T11:00:00Z" GIT_COMMITTER_DATE="2026-01-02T11:00:00Z" \
cat > greet.go <<'EOF'
package greet

func Hello(name string) string {
	if name == "" {
		name = "world"
	}
	return "Hello, " + name + "!"
}
EOF
git add greet.go
GIT_AUTHOR_DATE="2026-01-02T11:00:00Z" GIT_COMMITTER_DATE="2026-01-02T11:00:00Z" \
git commit -q -m "Default to 'world' when name is empty"

# Commit 3: doc tweak (the [Merge Request] tip)
GIT_AUTHOR_DATE="2026-01-02T12:00:00Z" GIT_COMMITTER_DATE="2026-01-02T12:00:00Z" \
git commit -q --allow-empty -m "[Merge Request] greet: default empty name to \"world\"

Two small changes plus a test:

- a basic test covering the happy path
- the empty-name default

Base: main"

git checkout -q main >/dev/null

echo "$dest"

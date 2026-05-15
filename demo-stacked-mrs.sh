#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
# SPDX-License-Identifier: EUPL-1.2
#
# Walk through a stacked-MR lifecycle (A → B → C) using the event-log
# convention. Self-contained: creates /tmp/mr-demo, throws it away on
# rerun. Reads like a screenplay; each `act` prints what's about to
# happen before doing it.

set -euo pipefail

REPO=/tmp/mr-demo
rm -rf "$REPO" && mkdir -p "$REPO" && cd "$REPO"

# Local git identity (no global config needed).
export GIT_AUTHOR_NAME="Author"      GIT_AUTHOR_EMAIL="author@demo"
export GIT_COMMITTER_NAME="Author"   GIT_COMMITTER_EMAIL="author@demo"

review() { GIT_AUTHOR_NAME="Reviewer" GIT_AUTHOR_EMAIL="reviewer@demo" \
           GIT_COMMITTER_NAME="Reviewer" GIT_COMMITTER_EMAIL="reviewer@demo" \
           git commit --allow-empty -m "$1"; }
merger() { GIT_AUTHOR_NAME="Merger" GIT_AUTHOR_EMAIL="merger@demo" \
           GIT_COMMITTER_NAME="Merger" GIT_COMMITTER_EMAIL="merger@demo" \
           "$@"; }

act() { printf '\n\033[1;36m── %s ──\033[0m\n' "$*"; }
show() { printf '\033[2m$ %s\033[0m\n' "$*"; eval "$*"; }

act "0. init repo, initial commit on main"
git init -q -b main
echo "# demo" > README.md
git add README.md
git commit -q -m "Initial commit"

act "1. open MR A (base: main)"
git checkout -q -b mr/a
echo "feature A" > a.txt && git add a.txt && git commit -q -m "Implement feature A"
git commit -q --allow-empty -m "[Merge Request] Add feature A

Adds a.txt with the A implementation.

Base: main"

act "2. open MR B stacked on A (base: mr/a)"
git checkout -q -b mr/b
echo "feature B" > b.txt && git add b.txt && git commit -q -m "Implement feature B (on top of A)"
git commit -q --allow-empty -m "[Merge Request] Add feature B

Adds b.txt; depends on A.

Base: mr/a"

act "3. open MR C stacked on B (base: mr/b)"
git checkout -q -b mr/c
echo "feature C" > c.txt && git add c.txt && git commit -q -m "Implement feature C (on top of B)"
git commit -q --allow-empty -m "[Merge Request] Add feature C

Adds c.txt; depends on B.

Base: mr/b"

act "4. dashboard view: active MRs"
show "git for-each-ref --format='%(refname:short) | %(subject)' refs/heads/mr/"

act "5. stack discovery (read Base: line from each MR tip body)"
for ref in $(git for-each-ref --format='%(refname:short)' refs/heads/mr/); do
  base=$(git log -1 --format='%B' "$ref" | awk -F': ' '/^Base:/ {print $2; exit}')
  printf '  %-6s  ←  %s\n' "$ref" "${base:-main}"
done

act "6. reviewer leaves a [Comment MR] on B"
git checkout -q mr/b
review "[Comment MR] please add a test for b.txt"
show "git log --oneline -3 mr/b"

act "7. author addresses comment: new work + fresh [Merge Request] tip"
echo "test for B" > b.test && git add b.test && git commit -q -m "Add test for feature B"
git commit -q --allow-empty -m "[Merge Request] Add feature B (revised)

Added b.test in response to reviewer comment.

Base: mr/a"
show "git log --oneline -6 mr/b"

act "8. reviewer approves all three, bottom-up"
git checkout -q mr/a && review "[Approved MR] looks good"
git checkout -q mr/b && review "[Approved MR] test resolves comment"
git checkout -q mr/c && review "[Approved MR] LGTM"

act "9. merger lands MR A → main (merge commit), then archives the branch"
git checkout -q main
merger git merge -q --no-ff mr/a -m "Merge MR: Add feature A"
git checkout -q mr/a
merger git commit -q --allow-empty -m "[Merged MR] sha=$(git rev-parse main)"
# "rename" mr/a → archive/mr/a
git branch -q archive/mr/a mr/a
git checkout -q main
git branch -q -D mr/a

act "10. merger lands MR B (B's history still contains [MR A] — no rebase)"
merger git merge -q --no-ff mr/b -m "Merge MR: Add feature B"
git checkout -q mr/b
merger git commit -q --allow-empty -m "[Merged MR] sha=$(git rev-parse main)"
git branch -q archive/mr/b mr/b
git checkout -q main
git branch -q -D mr/b

act "11. merger lands MR C"
merger git merge -q --no-ff mr/c -m "Merge MR: Add feature C"
git checkout -q mr/c
merger git commit -q --allow-empty -m "[Merged MR] sha=$(git rev-parse main)"
git branch -q archive/mr/c mr/c
git checkout -q main
git branch -q -D mr/c

act "12. final state: no active MRs, all three archived"
show "git for-each-ref --format='%(refname:short) | %(subject)' refs/heads/mr/ refs/heads/archive/mr/"

act "13. main's first-parent log reads as a merge ledger"
show "git log --first-parent --oneline main"

act "14. zoom in: full event log of archive/mr/b"
show "git log --oneline archive/mr/b"

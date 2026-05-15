#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
# SPDX-License-Identifier: EUPL-1.2
#
# Minimal end-to-end check that doesn't need VHS or a TTY:
# rebuild the test repo, run `gitflower review --no-tui`, compare the
# rendered .review against expected/smoke.review after normalising
# dates and SHAs.
#
# Use `--update` to refresh the golden.

set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
gitflower_dir="$(cd "$here/../.." && pwd)"

update=false
for arg in "$@"; do
  case "$arg" in
    --update) update=true ;;
    -h|--help) sed -n '4,15p' "$0" | sed 's/^# \{0,1\}//' ; exit 0 ;;
    *) echo "unknown flag: $arg" >&2; exit 2 ;;
  esac
done

# Build + setup.
bindir=$(mktemp -d)
trap 'rm -rf "$bindir" /tmp/gitflower-e2e-repo' EXIT
( cd "$gitflower_dir" && go build -o "$bindir/gitflower" . )

"$here/setup.sh" /tmp/gitflower-e2e-repo >/dev/null
cd /tmp/gitflower-e2e-repo
git config user.email reviewer@example.com

# Scaffold a .review without launching the TUI.
"$bindir/gitflower" review --no-tui --base main feature >/dev/null
produced=$(ls -t reviews/*.review | head -n1)

normalise() {
  sed -E '
    s/[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:]+Z/<DATE>/g
    s/\b[0-9a-f]{40}\b/<SHA40>/g
    s/\b[0-9a-f]{12}\b/<SHA12>/g
    s/\b[0-9a-f]{7}\b/<SHA7>/g
  ' "$1"
}

golden="$here/expected/smoke.review"
if [[ "$update" == "true" ]]; then
  mkdir -p "$here/expected"
  cp "$produced" "$golden"
  echo "updated $golden"
  exit 0
fi
if [[ ! -e "$golden" ]]; then
  echo "no golden at $golden — run smoke.sh --update to create one" >&2
  exit 1
fi
if diff -u <(normalise "$golden") <(normalise "$produced"); then
  echo "smoke: ok"
else
  echo "smoke: FAIL (golden differs)" >&2
  exit 1
fi

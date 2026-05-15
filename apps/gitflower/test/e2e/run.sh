#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
# SPDX-License-Identifier: EUPL-1.2
#
# E2E runner: builds the gitflower binary, runs every VHS scenario in
# scenarios/, then diffs the produced .review file against the golden in
# expected/ (after normalising volatile fields).
#
# Requires `vhs` (https://github.com/charmbracelet/vhs) on PATH. Install
# notes: `go install github.com/charmbracelet/vhs@latest` plus `ffmpeg` and
# either `ttyd` or `wezterm` per VHS's docs.
#
# Flags:
#   --update   regenerate goldens from this run's output
#   --keep     don't clean up /tmp/gitflower-e2e-repo on exit

set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
gitflower_dir="$(cd "$here/../.." && pwd)"

update=false
keep=false
for arg in "$@"; do
  case "$arg" in
    --update) update=true ;;
    --keep)   keep=true ;;
    -h|--help)
      sed -n '4,$p' "$0" | sed -n '/^#/!q; s/^# \{0,1\}//p'
      exit 0
      ;;
    *) echo "unknown flag: $arg" >&2; exit 2 ;;
  esac
done

if ! command -v vhs >/dev/null; then
  echo "vhs not on PATH (install: go install github.com/charmbracelet/vhs@latest)" >&2
  exit 1
fi

# Build the gitflower binary once into a temp path so the test doesn't
# fight with whatever you have under apps/gitflower/gitflower.
bindir=$(mktemp -d)
trap '$keep || rm -rf "$bindir" /tmp/gitflower-e2e-repo' EXIT
( cd "$gitflower_dir" && go build -o "$bindir/gitflower" . )

export GITFLOWER_BIN="$bindir/gitflower"
export GITFLOWER_E2E="$here"

# Normaliser used for golden diffs.
normalise() {
  sed -E '
    s/[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:]+Z/<DATE>/g
    s/\b[0-9a-f]{40}\b/<SHA40>/g
    s/\b[0-9a-f]{12}\b/<SHA12>/g
    s/\b[0-9a-f]{7}\b/<SHA7>/g
  ' "$1"
}

fail=0
for tape in "$here"/scenarios/*.tape; do
  name=$(basename "$tape" .tape)
  echo "=== $name ==="
  (
    cd "$here/scenarios"
    rm -f "$name.gif" "$name.cast"
    vhs "$tape"
  )
  # The TUI wrote one or more .review files; pick the freshest under the
  # rebuilt repo's reviews/ directory.
  produced=$(ls -t /tmp/gitflower-e2e-repo/reviews/*.review 2>/dev/null | head -n1 || true)
  if [[ -z "$produced" ]]; then
    echo "  FAIL: no .review file produced"
    fail=$((fail + 1))
    continue
  fi
  golden="$here/expected/$name.review"
  if [[ "$update" == "true" ]]; then
    mkdir -p "$here/expected"
    cp "$produced" "$golden"
    echo "  updated $golden"
    continue
  fi
  if [[ ! -e "$golden" ]]; then
    echo "  FAIL: missing $golden (run with --update to create)"
    fail=$((fail + 1))
    continue
  fi
  if diff -u <(normalise "$golden") <(normalise "$produced"); then
    echo "  ok"
  else
    echo "  FAIL: $name mismatched golden"
    fail=$((fail + 1))
  fi
done

if [[ $fail -gt 0 ]]; then
  echo "FAILED: $fail scenario(s)" >&2
  exit 1
fi
echo "All scenarios passed."

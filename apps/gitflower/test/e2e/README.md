<!--
SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
SPDX-License-Identifier: EUPL-1.2
-->

# gitflower e2e tests

Three runners covering different layers.

| Runner | Needs | What it does |
|---|---|---|
| `go test ./tui/` | Go only | drives the bubbletea model in-process via synthetic key events. Verifies section→line drill-in, comment creation, verdict cycling, and save. Sub-second. **Canonical** TUI behaviour check. |
| `test/e2e/smoke.sh` | Go + diff | rebuilds the fixture repo, runs `gitflower review --no-tui`, diffs the output `.review` against the golden after normalising dates/SHAs. Covers the format pipeline against real git. |
| `test/e2e/run.sh` | [VHS](https://github.com/charmbracelet/vhs) + `ttyd` + `ffmpeg` | drives each `scenarios/*.tape` through a real PTY/TUI session, then runs the same golden check. Produces `.gif` and `.cast` recordings. Slow but visual. |

The top-level `Makefile` wires these as `make test` / `make e2e` / `make e2e-vhs`.

The fixture repo is deterministic: `setup.sh` reconstructs it from scratch every run, with fixed identities and author dates so commit SHAs are byte-stable.

## Layout

```
test/e2e/
├── setup.sh            # rebuilds the fixture repo at /tmp/gitflower-e2e-repo
├── smoke.sh            # non-TUI golden check
├── run.sh              # VHS-driven golden check
├── scenarios/
│   └── walk-and-comment.tape  # VHS script: enter Changes, walk, comment, approve
└── expected/
    └── smoke.review    # normalised golden for smoke.sh
```

## Adding a scenario

1. Drop a new `scenarios/<name>.tape` in. VHS's reference: <https://github.com/charmbracelet/vhs>.
2. Run `./run.sh --update` once to generate `expected/<name>.review`.
3. Review the golden by hand. Commit when happy.

## Updating the golden after an intentional change

```bash
./smoke.sh --update
./run.sh   --update
```

Both rewrite the goldens from the current run's output. `git diff expected/` to sanity-check, then commit.

## Why VHS, not asciinema

VHS is bubbletea's native demo tool — one `.tape` file describes keys + timing declaratively, and VHS handles the PTY plumbing. It does also emit an asciinema `.cast` alongside the GIF if you `Output …cast` in the tape, so you don't lose the asciinema replay path. The `.gif` is for humans, the golden `.review` is for CI.

Generated `.gif` / `.cast` / `.mp4` / `.webm` files in `scenarios/` are gitignored by default — promote individual files by adding them explicitly with `git add -f`.

## Installing VHS

```bash
go install github.com/charmbracelet/vhs@latest
# also: ttyd (or wezterm) + ffmpeg per the VHS README
```

`smoke.sh` is the CI-friendly path — it covers the format round-trip and most of the workflow surface without needing a PTY. VHS scenarios stay as on-demand visual checks.

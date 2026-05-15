---
# SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
# SPDX-License-Identifier: EUPL-1.2
status: reviewed
---

# `.review` — patch-quoting file format for in-tree code reviews

## Goal

A single-file format for a code review. The design is based on Markdown with a fixed chapter structure — H1 = section, H2 = item, H3 = reviewer event. Everything quoted with `> ` is a real `git format-patch` body; everything else is reviewer prose. The on-disk file is a 1:1 image of the gitflower review TUI.

Replaces the current `*.review.md` markdown-with-fenced-diff format.

## Why

- **LLM-native.** Quoted-patch + interleaved prose is the kernel-review shape. Any model has seen thousands of these; no schema hints needed.
- **Patch-recoverable.** A `.review` file is not itself an applicable patch; recovery means *splitting it into per-commit `.patch` files*. For each `## Commit <sha>: …` subsection, strip the `> ` prefix from its quoted body and write out a standalone `.patch`. `git am *.patch` (or `b4`) then re-creates the commits. The per-file `## Changes in <path>` subsections recover to per-file diffs the same way (`git apply <path>.patch`). Recovery is a tooling step, not "just `sed` once".
- **1:1 UI ↔ file.** Tree mode sections in the TUI map directly to H1 headings. Events sit between the patch lines they anchor to.

## Fixed heading structure

| Level | Use | Examples |
|---|---|---|
| `#` (H1) | Top-level section | `# Review` / `# General Issues` / `# Changes` / `# Commits` |
| `##` (H2) | Item inside a section | `## Issue 1: <title>` / `## Changes in `<path>`` / `## Commit <sha>: <subject>` |
| `###` (H3) | **Reviewer event** | `### Comment (From: Alice <alice@x>, 2026-05-15T18:00:00Z)` |

H3 is reserved for events. Any H3 heading is a structural marker, never prose. This is the rule that lets the parser disambiguate events from arbitrary text.

### Heading shift inside user content

H1–H3 are reserved for the file's chapter structure. User-authored markdown (comment bodies, question bodies, issue bodies, verdict summaries) is shifted **+3 levels** before writing and **-3 levels** when read back into the TUI:

| User-typed | On disk |
|---|---|
| `# Heading` | `#### Heading` (H4) |
| `## Heading` | `##### Heading` (H5) |
| `### Heading` | `###### Heading` (H6) |
| `#### Heading` | `#### Heading` (clamped — H6 is the deepest standard markdown level; in practice users won't go this deep) |

Reading is the inverse: any H4+ heading inside a body is shifted up by 3 before display. H1–H3 inside a body are never produced by the writer and would indicate a hand-edit attempting to break out of a body — the parser treats them as section/item/event boundaries and closes the body accordingly.

This keeps the parser's heading-level invariant absolute (H1=section, H2=item, H3=event) while still letting users write headings in their own prose.

### Blockquote shift inside user content

`> ` (single greater-than + space) at the start of a line is reserved for quoted patch / file-content lines. To let user prose still contain markdown blockquotes, we shift by **+1 level**:

| User-typed | On disk |
|---|---|
| `> quote` | `>> quote` |
| `>> quote` | `>>> quote` |
| ... | ... |

Reading inverts the shift. This is what makes threaded replies work: a reviewer quoting an earlier comment writes `> earlier text` in their event body, which is stored as `>> earlier text` on disk — never colliding with the patch-line convention.

```
### Comment (From: Alice <alice@example.com>, 2026-05-15T18:00:00Z)

Needs a test.

### Comment (From: Bob <bob@example.com>, 2026-05-15T18:10:00Z)

>> Needs a test.

Added in commit X. The edge case is covered by `TestEmptyInput`.
```

Bob's body, in the TUI, reads as a single-level markdown blockquote (`> Needs a test.`) followed by his reply.

## File layout

Default filename: `reviews/<to-branch>-<to-sha>-from-<from-branch>-<from-sha>.review`. Extension `.review` (no `.md` — this is its own type even though the body is markdown-ish). The filename is keyed by *what is being reviewed*, not *who reviewed* — one file per (from, to) pair, shared across reviewers.

Top-level section order (always):

1. `# Review`
   - `## Sources` — metadata about what's being reviewed: from-branch / from-sha, to-branch / to-sha, diff range, commit count. Self-contained so the file is understandable without git access.
   - `## Verdicts` — verdict audit log (`### Verdict: …` events; latest = current state, possibly from multiple reviewers)
2. `# General Issues` — issues raised about the change or the repo as a whole
3. `# Changes` — one `## Changes in <path>` subsection per changed file, with events anchored inline
4. `# Commits` — one `## Commit <sha>: <subject>` subsection per commit, with events anchored inline
5. `# File Review` — one `## File <path> @ <short-sha>` subsection per file the reviewer entered in file-review mode, with events anchored inline. Only present when the reviewer actually opened a file in this mode.

   **Opening a file counts as reading it.** On entry to file-review mode for a file, the TUI auto-emits a `### ReadStart` at the first quoted line and a `### ReadEnd` at the last, bracketing whatever portion of the file is captured in the `## File …` subsection. Further scrolling extends the captured range (and the `ReadEnd` moves accordingly); the reviewer doesn't have to scroll through every line for the file to count as reviewed.

### Path encoding in headings

Filenames in `## Changes in `<path>`` are wrapped in single backticks. Inside the backticks:

- ASCII alphanumerics, `_`, `.`, `/`, `-` appear literally.
- Every other byte is escaped as `\uXXXX` (4-hex Unicode codepoint), or `\\` for a literal backslash, or `` \` `` for a literal backtick.

That keeps the heading line ASCII-only, single-line, and round-trippable for any filename git permits (including non-UTF-8 names, which git itself quotes with C-style escapes).

Parser regex (informative):

```
^## Changes in `((?:[A-Za-z0-9_./\\-]|\\u[0-9a-fA-F]{4}|\\\\|\\`)+)`$
^## File `((?:[A-Za-z0-9_./\\-]|\\u[0-9a-fA-F]{4}|\\\\|\\`)+)` @ ([0-9a-f]{7,40})$
```

### Quoted line shapes

Inside a `> ` quoted block, the *parent section* decides how to interpret the content:

| Parent section | `> ` line carries | Parser hint |
|---|---|---|
| `## Changes in <path>` | unified diff / `git diff` output | leading `diff --git`, `@@`, `+`, `-`, ` ` |
| `## Commit <sha>: …` | `git format-patch --stdout` body | mail-style headers then unified diff |
| `## File <path> @ <sha>` | line-numbered file content, format `<linenum>: <line>` | leading decimal then `: ` |

`## File` quote format:

```
> 1: package main
> 2:
> 3: import "fmt"
> 4:
> 5: func main() {
> 6:     fmt.Println("hello")
> 7: }
```

- `> <N>: <content>` — line N of the file at the tip SHA. `<content>` is the line verbatim (UTF-8). Empty content is `> <N>:` or `> <N>: ` (either accepted).
- Non-contiguous ranges: just emit only the lines the reviewer actually viewed. Gaps in the `<N>:` sequence are implicit (e.g. jumping from `> 7:` straight to `> 50:` means lines 8–49 were not visited).

```
# Review

## Sources

- From: `mr/a` at `abc1234567890abcdef`
- To:   `mr/b` at `def567890abc1234567`
- Diff: `mr/a..mr/b`
- Commits: 3 (oldest first)

## Verdicts

### Verdict: requested-changes (From: Alice <alice@example.com>, 2026-05-15T18:00:00Z)

Implementation is sound but needs a test before merge.

# General Issues

## Issue 1: change does not follow project coding style

Several names break the existing convention (single-letter identifiers
in production paths, lower_snake instead of camelCase for exported
names). Worth a pass before merge.

## Issue 2: extract auth into a separate role

(body)

# Changes

## Changes in `b.txt`

> diff --git a/b.txt b/b.txt
> @@ -0,0 +1,3 @@
> +feature B

### Comment (From: Alice <alice@example.com>, 2026-05-15T18:00:00Z)

Needs a test.

### ReadStart (From: Alice <alice@example.com>, 2026-05-15T18:01:00Z)

> +initial implementation

### Like (From: Alice <alice@example.com>, 2026-05-15T18:02:00Z)

> +line 3

### Dislike (From: Alice <alice@example.com>, 2026-05-15T18:02:30Z)

### Comment (From: Alice <alice@example.com>, 2026-05-15T18:02:30Z)

Single-letter name. See Issue 1.

### ReadEnd (From: Alice <alice@example.com>, 2026-05-15T18:02:30Z)

## Changes in `b.test`

> diff --git a/b.test b/b.test
> @@ -0,0 +1 @@
> +test for B

### ReadStart (From: Alice <alice@example.com>, 2026-05-15T18:03:00Z)

### ReadEnd (From: Alice <alice@example.com>, 2026-05-15T18:03:00Z)

# Commits

## Commit dd56c2ea01a7: Add feature B (revised)

> From dd56c2ea01a7… Mon Sep 17 00:00:00 2001
> From: Author <author@demo>
> Date: …
> Subject: [PATCH] Add feature B (revised)
>
> Added b.test in response to reviewer comment.
>
> ---
>  b.txt | 1 +
>
> diff --git a/b.txt b/b.txt
> @@ -0,0 +1 @@
> +feature B

### Question (From: Alice <alice@example.com>, 2026-05-15T18:05:00Z)

Why "revised"? Was there an earlier version that got squashed?

## Commit b855271e1cad: Implement feature B (on top of A)

> From b855271e1cad… …
> …

# File Review

## File `auth/handler.go` @ dd56c2ea01a7

> 42: func authenticate(req *Request) error {
> 43:     token := req.Header.Get("X-Token")
> 44:     if token == "" {

### Comment (From: Alice <alice@example.com>, 2026-05-15T18:10:00Z)

The empty-token check below should also reject whitespace-only tokens.

> 45:         return ErrMissingToken
> 46:     }
> 47:     return verify(token)
> 48: }

### ReadStart (From: Alice <alice@example.com>, 2026-05-15T18:10:00Z)

### ReadEnd (From: Alice <alice@example.com>, 2026-05-15T18:11:00Z)
```

## Reviewer-event syntax

Every reviewer action is an H3 heading followed by an optional markdown body:

```
### <Kind>[: <param>] (From: <Name> <<email>>, <RFC3339 date>)

<body markdown — paragraphs, lists, code blocks, anything>
```

- The heading line ends the previous event (if any).
- The body runs until the next H1, H2, H3, or `> ` patch line.
- Body may be empty (for markers like `ReadStart` / `Like` with no reasoning).
- `<Kind>` is **case-insensitive on read** (`comment`, `Comment`, `COMMENT` all open a Comment event). The writer always emits the canonical capitalised form.

| `<Kind>` | Where it lives | Body | Param |
|---|---|---|---|
| `Comment` | `# Changes` / `# Commits` (anchored to a patch line) | required, prose | — |
| `Question` | same | required, prose | — |
| `ReadStart` | same | **empty** (marker only — opens a read range) | — |
| `ReadEnd` | same | **empty** (marker only — closes a read range) | — |
| `Like` | same | **empty** (marker only — 👍) | — |
| `Dislike` | same | **empty** (marker only — 👎) | — |
| `Verdict` | `# Review` → `## Verdicts` only | optional summary | `open` \| `requested-changes` \| `approved` \| `denied` |

`Like`, `Dislike`, `ReadStart`, `ReadEnd` are pure single-point markers — no reasoning attached. `Comment` / `Question` are also single-point (anchored to one preceding `> ` line, never a range). The reviewed *area* is expressed only via the `ReadStart` … `ReadEnd` pair: everything between them in the same (sub)section was read. If the reviewer wants to explain a reaction, they add a separate `### Comment` immediately after the marker.

**Pairing `ReadStart` with `ReadEnd`.** Pairs are matched by **timestamp**, not lexical position. For each `(reviewer, section)` partition, sort all `ReadStart` and `ReadEnd` events by their `(From: …, <date>)` timestamp, then walk in order and pair each `ReadStart` with the next `ReadEnd` from the same reviewer in the same (sub)section. An unmatched `ReadStart` closes at the section end. This works correctly across hand-edits and multi-reviewer files, where positional pairing would be ambiguous.

Multiple `### Verdict:` events form an audit log; the latest is the current state.

```
### Verdict: requested-changes (From: Alice <alice@example.com>, 2026-05-15T18:00:00Z)

Needs a test before merge.

### Verdict: approved (From: Alice <alice@example.com>, 2026-05-15T19:00:00Z)

Test was added, ready to merge.
```

## Anchoring

An event under `# Changes` or `# Commits` attaches to the **immediately preceding `> ` patch line** at parse time. The TUI's anchor format (`<path>:<line>`, etc.) is *derived* from that patch line's position in its hunk + the file the hunk belongs to. The anchor is **not stored explicitly** in the event — moving an event in the file moves it to a different anchor by definition.

Events in `# Review` and `# General Issues` (no preceding `> ` line) anchor to nothing — they apply to the section.

Multiple events at the same anchor are supported: the parser appends them in chronological (file) order, so consecutive `### Comment (…)` paragraphs on the same anchor form a thread.

**Commit-message comments.** To comment on a commit's *intent* (the commit message, not any diff line), anchor to the `Subject:` line of the `git format-patch` body — that line is itself a `> ` line, so a `### Comment` placed right after it falls out of the existing rules with no special-casing.

## Parser

Line-oriented state machine. Track: `section` (one of Review / GeneralIssues / Changes / Commits / FileReview), `subsection` (issue / commit / file), `lastPatchLine`, `currentEvent`.

| Line shape | Action |
|---|---|
| `# <title>` | Close any open event; switch section; reset subsection and `lastPatchLine`. |
| `## <title>` | Close any open event; switch subsection (parse `Issue N: title` or `Commit sha: subject`). |
| `### <Kind>[: <param>] (From: …, <date>)` | Close any open event; open a new event of the given kind, anchored to `lastPatchLine` (or to the (sub)section if none). |
| `> <text>` | Close any open event; record `lastPatchLine` (append to current section/subsection's quoted-content buffer). Under `## Changes in …` / `## Commit …` the text is unified-diff; under `## File … @ …` the text is `<N>: <line>`. |
| `>` (no trailing space) | Empty quoted line; same as `> ` with empty content. |
| Blank line | If inside an event body: noted (kept as paragraph separator). Otherwise: separator only. |
| Any other line | If inside an event body: append as body line. If inside an issue body (under `## Issue N:`): append as body line. Otherwise: free prose (`# Review` summary or stray text). |

Header regex (informative):

```
^### (Comment|Question|ReadStart|ReadEnd|Like|Dislike|Verdict)(: (open|requested-changes|approved|denied))? \(From: (.+) <(.+)>, (.+)\)$
```

## Writer

Generation per save:

1. Emit `# Review`.
   - Emit `## Sources` subsection with from/to branch + sha, diff range, commit count.
   - Emit `## Verdicts` subsection containing one or more `### Verdict: <state> (From: …)` events (audit log; latest = current state; reviewer field may differ across entries).
2. Emit `# General Issues`. For each issue: `## Issue <N>: <title>` + body.
3. Emit `# Changes`. For each changed file (sorted, oldest-merge-order if available):
   - `## Changes in `<encoded-path>`` heading.
   - `git diff <base>..<branch> -- <path>` → prefix every line with `> ` → emit.
   - Interleave events at their anchor positions inside this subsection.
4. Emit `# Commits`. For each commit (oldest first):
   - `## Commit <short>: <subject>` heading.
   - `git format-patch --stdout <commit>^..<commit>` → prefix `> ` → emit.
   - Interleave events.
5. If the reviewer entered file-review mode on at least one file, emit `# File Review`. For each such file (in the order first opened):
   - `## File `<encoded-path>` @ <short-sha>` heading.
   - For each contiguous range of lines the reviewer viewed: `git show <tip>:<path>` → number lines → emit only the visited range as `> <N>: <line>`.
   - On the first open of a file, auto-emit `### ReadStart (From: …)` immediately after the first quoted line and `### ReadEnd (From: …)` immediately after the last quoted line. Subsequent extensions of the visited range move `ReadEnd` (and possibly `ReadStart`, if the reviewer scrolled backwards from the entry point).
   - Interleave other events (Comment / Question / Like / Dislike) at their anchor positions.

## UI ↔ file mapping

| TUI tree section | File section |
|---|---|
| Tree mode → Diffs | `# Changes` → per-`## Changes in <path>` |
| Tree mode → Tree (files at tip) | Computed from `git ls-tree <tip>` — not stored in file unless the reviewer opens one (then it lands in `# File Review`) |
| Tree mode → Commits | `# Commits` → per-`## Commit` |
| Tree mode → Issues (general) | `# General Issues` → per-`## Issue` |
| File-review mode → open file | `# File Review` → `## File <path> @ <sha>` with visited line ranges as `> <N>: <line>` |
| Diff/File mode comment on a line | `### Comment (…)` paragraph between the relevant `> ` lines under `## Changes in <path>`, `## Commit …`, or `## File …` |
| Question (same, `### Question (…)`) | Likewise |
| Reviewed range | `### ReadStart (…)` + `### ReadEnd (…)` pair, anchored at the first and last `> ` lines of the range |
| Like / dislike marker | `### Like (…)` / `### Dislike (…)` likewise (body-less) |
| Verdict | `# Review` → `## Verdicts` → `### Verdict: <state> (…)` event (latest wins) |

## Recovering patches

```bash
# whole-review changes (all per-file diffs concatenated)
sed -nE '/^# Changes$/,/^# Commits$/p' file.review \
  | sed -E '/^> ?/!d; s/^> ?//'

# one file's changes
awk '/^## Changes in `b\.txt`$/{p=1; next} /^## /{p=0} p' file.review \
  | sed -E '/^> ?/!d; s/^> ?//'

# a single commit
awk '/^## Commit dd56c2ea01a7/{p=1; next} /^## /{p=0} p' file.review \
  | sed -E '/^> ?/!d; s/^> ?//'
```

`git apply` of the recovered stream succeeds. `git am` works on the per-commit recovery.

## Workflow

1. `gitflower review <branch>` — generates a fresh `.review` from the branch's diff + commits (or loads an existing one).
2. TUI opens; reviewer reads, comments, marks. Every action auto-saves.
3. `.review` lives on the MR branch under `reviews/<to-branch>-<to-sha>-from-<from-branch>-<from-sha>.review`. Subsequent reviewers open the same file and append their events.
4. On approval, the merger lands the MR; the `.review` lives in `archive/mr/<slug>` as part of the archived branch history (transient — does not land on `main` per the in-tree-review pattern's dissolution rule).
5. Issues from `# General Issues` can be promoted to standalone `issues/*.md` files before dissolution.

## Why no frontmatter

1. **One representation, not two.** All state lives in the body. A reader (human or LLM) processes the document linearly; nothing is hidden in a top metadata block.
2. **Frontmatter encourages divergence.** With YAML at the top *and* events inline, "what's the verdict?" has two answers. Single source of truth: the body events.

## Why not mbox

mbox would force one envelope per event with `From`/`Subject`/`In-Reply-To` headers. Heavier per-event overhead, and the threading is already clear from "events-between-patch-lines" placement. Mbox is the right choice when reviews travel by email; for in-tree storage we don't need it.

## Multi-reviewer support

One `.review` file per (from, to) pair, shared across reviewers. Each event carries `(From: <Name> <<email>>, <date>)` attribution, so comments / questions / verdicts from different people interleave naturally in the same file. Concurrent editing is *not* solved by the format — coordinate by branch + lock conventions externally if needed.

## Binary diffs

`git diff` emits `Binary files a/foo and b/foo differ` for binary content (or, with `--binary`, a base64 literal). Either way the line is prefixed with `> ` like any other diff line. Events can still anchor to a binary file's `Binary files` line for a file-level comment, but per-line review is not possible inside a binary blob — that's a property of binary content, not of this format.

## Non-goals

- **Not a patching surface.** The patches inside ARE git patches and they recover cleanly when split per-commit, but the `.review` file is for *tracking a single review accurately in one file*. If you want to apply changes from it, recover the per-commit `.patch` files first and run `git am` on those.
- **Not a binary format.**
- **Not a forge replacement.** No notifications, no permissions model, no rate limiting. Just the on-disk artefact of one review.

## References

- Linux kernel patch review on `lkml` (the mental model this format borrows from).
- The in-tree-review pattern (`patterns/approaches/in-tree-review.md` — not yet written; see prior discussion).
- The current `*.review.md` implementation in `apps/gitflower/review/` — the thing this replaces.

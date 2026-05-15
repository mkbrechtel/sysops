---
title: Merge Request as Tip Commit 🔖
---

<!--
SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>

SPDX-License-Identifier: EUPL-1.2
-->

## Overview 📋

A branch is a Merge Request iff its history ends with an *event-log* of
empty commits whose subjects encode the MR's lifecycle —
`[Merge Request]` opens it, `[Approved MR] / [Denied MR] / [Comment MR]`
record review actions, `[Merged MR] / [Closed MR]` close it. The
**tip** is whichever event happened most recently; the MR's current
state is the tip's subject.

The merge request lives in `git` itself, not in a forge database.
Forges (Forgejo / GitLab / GitHub) — and the project's own website —
become *renderers* of the branch, not its home.

## Goals 🎯

- Make a branch's intent and current state visible to anyone running
  `git log -1`.
- Keep MR descriptions, reviews, and comments inside the repo, so they
  survive forge migrations and are searchable with `grep`.
- Reduce "open MR" to a single, greppable convention — no API, no
  database, no labels system.
- Let agents and scripts act on MRs by reading commit messages and
  appending empty commits; no auth tokens.
- Make "what's everyone working on?" answerable from refs alone.

## Pattern Structure 📑

### The MR commit

```
[Merge Request] Add stacked MR helper script

We need a way to list MR branches and their stacking
relationships from the CLI without talking to the forge.

This adds tests/stacked-mrs, which walks each MR branch's
history and prints its parent MR.

Base: main
Closes: issues/stacked-mrs.feature.md
```

- **Subject:** `[Merge Request] ` + a human title.
- **Body:** what a reviewer would read on the forge — motivation,
  scope, follow-ups, references to issue files. An optional
  `Base: <branch>` line declares the stacking parent (see
  [Stacked Merge Requests 🪜](./stacked-merge-requests.md)).
- **Empty commit:** no file changes. Implementation lives in the
  commits *underneath* it.

### Review events (the event log)

Reviews are also empty commits, appended to the branch tip.

| Subject prefix         | Meaning                  | Body                | Posted by               |
|------------------------|--------------------------|---------------------|-------------------------|
| `[Merge Request] …`    | MR opened (or re-opened) | description         | author / agent          |
| `[Approved MR] …`      | Reviewer approved        | optional reasoning  | website / maintainer    |
| `[Denied MR] …`        | Reviewer denied          | reasoning           | website / maintainer    |
| `[Comment MR] …`       | Reviewer wants changes   | comment             | website / maintainer    |
| `[Merged MR] sha=…`    | Merger landed it         | merge SHA, notes    | merger agent            |
| `[Closed MR] …`        | MR abandoned             | reason              | merger / operator       |

A `[Comment MR]` is handled by the author adding new work commits and
then appending a *fresh* `[Merge Request]` tip — effectively
re-opening at the new tip. The full review thread is preserved in
history; the state machine just resets to "open".

### Branch namespace = lifecycle

```
refs/heads/mr/<slug>          ← active (open / commented / approved / denied)
refs/heads/archive/mr/<slug>  ← terminal (merged or closed)
```

Closing or merging doesn't delete the branch — it moves the ref into
`archive/mr/`. The full thread stays greppable forever. Branches
don't rename atomically across a remote; do it in two pushes:

```bash
git push origin refs/heads/mr/foo:refs/heads/archive/mr/foo
git push origin --delete mr/foo
```

### Listing MRs

```bash
# Active
git for-each-ref --format='%(refname:short)|%(subject)' refs/heads/mr/

# Archived
git for-each-ref --format='%(refname:short)|%(subject)' refs/heads/archive/mr/
```

That output *is* the MR dashboard. Pipe it into anything — `fzf`, the
website build, a status page, an agent's planning context.

### State machine

```
                                  [Approved MR]
                                 ┌────────────► merger merges
                                 │              → [Merged MR]
                                 │              → move to archive/mr/
                                 │
   [Merge Request] ── reviews ───┤[Comment MR]
   (open)                        ├────────────► author appends work
                                 │              + fresh [Merge Request]
                                 │              (reopens at new tip)
                                 │
                                 └────────────► [Denied MR]
                                                → [Closed MR]
                                                → move to archive/mr/
```

State = subject prefix of the most recent commit on the branch.

### Forge / website integration

Forges already render the tip commit's body as part of the MR view —
when the body is the original `[Merge Request]` commit, no extra
wiring is needed. For a project-owned website, expose a tiny API:

```
GET  /api/mrs                   → list (state per ref-walk)
GET  /api/mrs/<slug>            → detail (commits + diff vs Base:)
POST /api/mrs/<slug>/review     → append empty event commit
```

The backend's only mutation is `git commit --allow-empty` on the MR
branch. No database.

### Server-side auto-approval (optional)

Low-risk MRs can be approved at push time by a `post-receive` hook
that path-globs the diff. If every changed path matches an allowed
pattern (e.g. `questions/*.md`) the hook appends `[Approved MR]
auto: <reason>` directly via `commit-tree` + `update-ref` — no
checkout, no LLM round-trip. Operators override by appending a later
`[Comment MR] / [Denied MR]`; the merger respects the most recent
event.

## Security Considerations 🔐

- Commit subjects and bodies are public the moment they're pushed.
  Don't put secrets in any event commit.
- Force-pushing the tip is part of normal use (rebasing onto a new
  base, amending a wording fix); protect `main` with the usual
  non-fast-forward rejection (see
  [Worktree Treehouses 🌳](./worktree-treehouses.md)), but leave MR
  branches force-pushable.
- The auto-approval hook is an authorisation primitive — review its
  path glob like any other policy rule.

## Anti-Patterns ⚠️

- ❌ Writing the MR description only in the forge UI. Forge databases
  churn; the repo doesn't.
- ❌ Putting code changes in any event commit (`[Merge Request]`,
  `[Approved MR]`, …). All event commits are empty by definition.
- ❌ Mutating an existing event commit instead of appending a new one.
  The log is append-only; rewriting hides the thread.
- ❌ Deleting an MR branch on close. Move it to `archive/mr/` instead;
  the conversation belongs in the repo.
- ❌ A subject like `[MR]` / `MR:` / `Merge Request:`. The bracket
  form is the protocol; drift breaks the grep.

## Best Practices 💡

- Open the MR commit *first*, even before any code. The branch starts
  life with an empty `[Merge Request]` tip describing what you intend
  to do — like an in-tree issue, but for a unit of work
  ([In-Tree Issues 🗂️](./in-tree-issues.md) covers the longer-lived
  side).
- Keep the subject under 72 chars *after* `[Merge Request] `; it
  shows up in `git log --oneline`.
- Reference issue files (`issues/foo.feature.md`) in the body so the
  issue and MR are linked without an external tracker.
- After addressing a `[Comment MR]`, **append** a new `[Merge Request]`
  rather than amending the old one. The thread stays linear.
- Treat `archive/mr/` as the project's MR archive — grep it for prior
  art before opening a new MR on the same area.

## Checklist ✅

### Open an MR
- [ ] Branch out from `main` (or from a parent MR — see stacking).
- [ ] `git commit --allow-empty -m '[Merge Request] <title>'`; fill
  the body.
- [ ] Push to `refs/heads/mr/<slug>`.

### Review
- [ ] Append `[Approved MR]`, `[Denied MR]`, or `[Comment MR]` —
  empty commit, body = reasoning.

### Land
- [ ] Merger merges into `main`, appends `[Merged MR] sha=<hash>`,
  and pushes the branch to `archive/mr/<slug>` (then deletes
  `mr/<slug>`).

### Close
- [ ] Append `[Closed MR] <reason>`, push to `archive/mr/<slug>`,
  delete `mr/<slug>`.

## Related Patterns 🔗

- [Stacked Merge Requests 🪜](./stacked-merge-requests.md) — how
  this convention extends to chains of dependent MRs.
- [In-Tree Issues 🗂️](./in-tree-issues.md) — same spirit, longer
  horizon: the issue lives in the tree; the MR lives in the event
  log.
- [Worktree Treehouses 🌳](./worktree-treehouses.md) — one
  treehouse per open MR pairs naturally with one event log per
  branch.

## References 📚

- `git for-each-ref` docs: <https://git-scm.com/docs/git-for-each-ref>
- **Reference implementation:** the `fix-europe` project's
  `docs/spec/merge-request-pattern.md` and its `post-receive` hook
  (`deploy/srv-post-receive`) — a single-operator site that runs
  entirely on this convention, with an Astro/Go website rendering
  the MR view from `refs/heads/mr/*`.

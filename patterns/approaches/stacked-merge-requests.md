---
title: Stacked Merge Requests 🪜
---

<!--
SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>

SPDX-License-Identifier: EUPL-1.2
-->

## Overview 📋

When merge request **B** depends on merge request **A** that isn't
merged yet, B is *stacked on top of* A. Because every MR carries a
[`[Merge Request]` event log](./merge-request-tip-commit.md), the
stacking relationship can be recorded two ways:

1. **Declared** — the MR's body contains a `Base: <branch>` line.
   The diff renderer and merger read it directly.
2. **Inferred** — walking back from the MR's tip, the first ancestor
   whose subject starts with `[Merge Request]` is the parent MR.

Declared is more robust to rebases; inferred works even when no one
remembered to write `Base:`. Tooling can use both — declared as the
truth, inferred as a fallback / consistency check.

Either way: no stacking tool, no Graphite, no dependency database.
The graph falls out of `git log` + one optional commit-body line.

## Goals 🎯

- Split big changes into reviewable units without losing the
  dependency graph between them.
- Let a reviewer answer "what does this MR depend on?" with one
  `git log` invocation.
- Avoid a parallel system (`gh stack`, Graphite, ghstack) whose
  state must be kept in sync with branches.
- Make the stack visible as a normal git artefact — diagrams,
  dashboards, and CI can all read it.

## Pattern Structure 📑

### Stacking convention

```
main ───●───●───●
                 \
                  ●─●─[Merge Request] A      ← branch mr/a
                         \                      (Base: main)
                          ●─●─[Merge Request] B   ← branch mr/b
                                  \                  (Base: mr/a)
                                   ●─[Merge Request] C  ← branch mr/c
                                                          (Base: mr/b)
```

- `mr/b` is branched from `mr/a`'s tip; `mr/c` from `mr/b`'s tip.
- Each MR's commit body declares `Base: <parent>` (or omits it,
  defaulting to `main`).
- B's history contains A's tip; C's contains both.
- When A merges to `main`, the merger rebases B onto `main` and
  rewrites B's `Base:` line to `main`; the same for C.

The diff renderer computes `git merge-base <mr-branch> <Base:>` to
show only the MR's own contribution, not the parent's.

### Finding a single MR's parent

From any MR branch tip, find the nearest ancestor MR:

```bash
git log --format='%H %s' <branch>^ \
  | awk '/^[0-9a-f]+ \[Merge Request\]/ { print; exit }'
```

If output is empty, the MR sits directly on `main`. If non-empty,
the printed hash is the tip of the parent MR — map hash → branch
name with `git for-each-ref`.

### Listing all stacks (`tests/stacked-mrs`)

```bash
#!/usr/bin/env bash
# List every MR branch and its parent MR (if any).
set -euo pipefail

# 1. Collect MR tips: hash → branch.
declare -A MR_OF
while read -r hash ref subject; do
  [[ "$subject" == "[Merge Request]"* ]] || continue
  MR_OF[$hash]="$ref"
done < <(git for-each-ref \
  --format='%(objectname) %(refname:short) %(subject)' \
  refs/heads)

# 2. For each MR, walk parents to the next MR commit.
for hash in "${!MR_OF[@]}"; do
  branch="${MR_OF[$hash]}"
  parent_hash=$(git log --format='%H %s' "$hash^" 2>/dev/null \
    | awk '/^[0-9a-f]+ \[Merge Request\]/ { print $1; exit }')
  if [[ -n "$parent_hash" && -n "${MR_OF[$parent_hash]:-}" ]]; then
    printf '%s  ←  %s\n' "$branch" "${MR_OF[$parent_hash]}"
  else
    printf '%s  ←  main\n' "$branch"
  fi
done | sort
```

Sample output:

```
mr/a  ←  main
mr/b  ←  mr/a
mr/c  ←  mr/b
mr/docs-cleanup  ←  main
```

Pipe into `tred`(1) for the transitive reduction, or into Mermaid
for a rendered dependency graph on the website.

### Reviewing a stack

- Review **bottom-up**: A before B before C. A's review bar is
  unchanged; B is reviewed *assuming A is good*.
- A reviewer can see only B's contribution with
  `git diff mr/a..mr/b`.
- When A merges, the maintainer rebases B onto `main` (or B's
  author does) and the stack shortens by one.

## Security Considerations 🔐

- Same as the underlying convention — no secrets in commit
  subjects/bodies.
- A stacked MR inherits everything its parent introduces. Reviewers
  must be aware of the full chain; the script above makes that
  inheritance explicit.

## Anti-Patterns ⚠️

- ❌ Merging B before A. The stack assumes A lands first; merging
  out of order leaves orphan history.
- ❌ Cherry-picking commits between stacked branches "to keep them
  independent". You lose the dependency signal and double-review
  the same code.
- ❌ Stacks deeper than ~3. Each level multiplies rebase cost when
  the bottom changes. If you need more, split into independent
  branches off `main`.
- ❌ Hiding the stack from the forge UI. Link parent MRs in each
  MR body (or let the script generate that block).

## Best Practices 💡

- Name stacked branches with a shared prefix: `mr/feat-a`,
  `mr/feat-b`, `mr/feat-c`. Sorting `git branch` then shows the
  stack visually.
- Run the discovery script in CI on every push; post the output as
  a comment on each MR in the stack, so reviewers see siblings.
- Rebase the *whole stack* whenever the bottom rebases — a single
  `git rebase --update-refs` (git ≥ 2.38) does it.
- Land the bottom MR fast. Stacks rot quickly; a stale bottom
  blocks everything above.

## Checklist ✅

### Author

- [ ] Branch B from A's tip, not from `main`.
- [ ] Add B's `[Merge Request]` tip commit; mention the parent MR
  in the body.
- [ ] After A merges, rebase B onto `main`
  (`git rebase --onto main mr/a mr/b`).

### Reviewer

- [ ] Run the discovery script (or read the bot comment) before
  reviewing — know the stack.
- [ ] Review bottom-up.
- [ ] Merge bottom-up; don't approve a child while the parent is
  contested.

### Tooling

- [ ] Ship the discovery script in `tests/` so every clone has it.
- [ ] Render the stack on the website's MR dashboard.
- [ ] Optional: a `git stack` alias that runs the script and
  highlights the current branch.

## Related Patterns 🔗

- [Merge Request as Tip Commit 🔖](./merge-request-tip-commit.md) —
  the convention this pattern depends on.
- [Worktree Treehouses 🌳](./worktree-treehouses.md) — one
  treehouse per stacked MR keeps rebases local and cheap.
- [In-Tree Issues 🗂️](./in-tree-issues.md) — when a stack
  implements an issue, link the issue file from every tip body.

## References 📚

- `git rebase --update-refs` (git 2.38+): rebases an entire stack of
  branches in one go.
- **Reference implementation:** the `fix-europe` project's
  `docs/spec/merge-request-pattern.md` § *Stacked MRs*. The
  merger agent auto-rebases stacked children when a parent lands,
  or surfaces the chain to the operator if rebase conflicts.
- [ghstack](https://github.com/ezyang/ghstack),
  [Sapling](https://sapling-scm.com/) — heavier tools solving the
  same problem; this pattern is the smalltown version. (We don't
  recommend commercial stacking SaaS — pick one of the open
  approaches above.)

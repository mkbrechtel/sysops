---
title: Worktree Cottages 🏡
---

<!--
SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>

SPDX-License-Identifier: EUPL-1.2
-->

## Overview 📋

A shared bare repository lives on one host. Each contributor — human
or AI agent — checks out their own private worktree under `work/<branch>`
in the same directory. The filesystem (Unix users, groups, setgid,
sticky bit) decides who can write where, and `git worktree` itself
guarantees that no two cottages share a branch. The result is a
village where everyone can *see* what their neighbours are doing, but
only the resident edits inside their own cottage.

It's the same trick whether you have three operators sharing a server
or one operator with three Claude agents: every actor gets a cottage,
and the bare repo is the village hall they all push to.

## Goals 🎯

- Give every contributor (human or agent) an isolated working tree
  without the cost of a full clone per person.
- Let teammates read each other's in-progress work without races or
  permission gymnastics.
- Push access control down to the kernel — no bespoke "who can touch
  this branch" service.
- Use a *single* worktree-creation path for humans and agents, so
  there's only ever one code path to debug.
- Keep `ls work/` as the honest answer to "what's everyone working
  on right now?"

## Pattern Structure 📑

### Layout on disk

```
/srv/repos/<project>.git              ← bare repo, group-owned
/srv/projects/<project>/
├── bin/new-worktree                  ← helper humans + agents share
├── CLAUDE.md                         ← tells agents to use EnterWorktree
├── .claude/
│   ├── settings.json                 ← wires WorktreeCreate/Remove hooks
│   └── hooks/
│       ├── worktree-create           ← delegates to bin/new-worktree
│       └── worktree-remove
└── work/                             ← 3775 (rwxrwsr-t): setgid + sticky
    ├── main/                         ← long-lived; group-readable
    ├── alice/feature-x/              ← owned by alice
    └── claude/refactor-auth/         ← owned by the user driving Claude
```

`work/` is **group-writable, setgid, sticky** (`chmod 3775`). New
cottages inherit the group; the sticky bit means nobody can delete
a neighbour's leaf. Each cottage itself is created group-writable,
then immediately stripped to `g-w` so others can read but not
modify it.

### One branch ⇄ one cottage

Git already refuses to check out the same branch in two worktrees.
Lean on that: the branch *is* the cottage name, and the existence of
the directory is the lock. No external coordination needed.

```
$ bin/new-worktree feature/cute-thing
worktree ready: /srv/projects/foo/work/feature/cute-thing
                (branch feature/cute-thing, based on main)

$ bin/new-worktree feature/cute-thing      # someone else tries
worktree feature/cute-thing already exists and belongs to alice
                  — pick a different name
```

### The shared helper

A small script (the one humans type, the one the agent hook calls)
does four things:

1. **Sanitise the name** — letters, digits, `._/-`, no `..`.
2. **Refuse to steal** — if the directory exists and isn't yours,
   bail out with the owner's name.
3. **Create branch + worktree** in the bare repo (creating the
   branch from a base ref if it doesn't exist yet).
4. **`chmod -R g-w`** so the cottage is yours alone to write.

Intermediate dirs along nested names (`feature/foo`) get `3775` so
the setgid + sticky combo propagates.

### Agent integration

`.claude/settings.json` wires Claude's `WorktreeCreate` and
`WorktreeRemove` lifecycle hooks to two tiny shell scripts that
parse the JSON on stdin and delegate to the *same* `bin/new-worktree`.
That convergence is the whole point: the agent's `EnterWorktree`
tool and a human typing `bin/new-worktree` end up running identical
code, producing identical filesystem layouts, with identical
ownership rules.

```json
{
  "hooks": {
    "WorktreeCreate": [{"hooks": [{"type": "command",
      "command": "/srv/projects/foo/.claude/hooks/worktree-create"}]}],
    "WorktreeRemove": [{"hooks": [{"type": "command",
      "command": "/srv/projects/foo/.claude/hooks/worktree-remove"}]}]
  }
}
```

The remove hook is paranoid by design: it refuses to act on any
path that isn't under `work/`, then calls `git worktree remove
--force`.

### Lifecycle

```
   idea ──► bin/new-worktree <name>       (or EnterWorktree from Claude)
                       │
                       ▼
              work/<name>/  ← edit, commit, push to bare repo
                       │
                       ▼
              MR / merge into main        (use [[in-tree-issues]] style)
                       │
                       ▼
              git worktree remove          (or ExitWorktree)
```

Pushing to the bare repo is local and instant. If the bare repo has
a `reference-transaction` hook (as in this very project), `main` is
fast-forward-only and merge-commit-only — protecting the village
hall without extra services.

## Security Considerations 🔐

- **Input sanitisation in the helper is load-bearing.** The hook
  passes a name out of an untrusted JSON payload. Reject anything
  outside `[A-Za-z0-9._/-]` and explicitly reject `..`.
- **The remove hook must geofence to `work/`.** A typo (or a
  prompt-injected agent) shouldn't be able to delete arbitrary paths.
- **No network surface added.** The bare repo is reached via local
  filesystem permissions; there's no extra daemon to harden.
- **Ownership = authorship.** Every commit in a cottage is made by
  the Unix user who owns it. `git log` and `stat` agree on who did
  what — useful for audit, useful for blame.
- **Agent isolation is *per cottage*, not per process.** An agent
  running as user `alice` can still read everyone else's cottage.
  That's a feature for collaboration; pair it with a system-level
  sandbox if you need stronger separation.

## Anti-Patterns ⚠️

- ❌ **A clone per user.** Wastes disk, hides in-progress work
  behind `ssh`, makes "what is everyone doing?" a coordination
  problem instead of an `ls`.
- ❌ **One shared worktree with branch switching.** Two humans (or
  a human and an agent) will collide on `git checkout` within the
  first hour. Git designed worktrees specifically to make this
  unnecessary.
- ❌ **Letting an agent share its driver's worktree.** Either you
  race the agent for the file lock, or you serialise edits by hand.
  Give the agent its own cottage with `EnterWorktree` and read its
  diffs in yours.
- ❌ **Tracking who-owns-what in Slack / a wiki / a spreadsheet.**
  `ls -l work/` is the truth; anything else drifts.
- ❌ **Replacing filesystem permissions with a CI policy gate.** The
  kernel already does access control. Don't outsource a primitive
  you already have.
- ❌ **Long-lived cottages.** A branch that lingers past its merge
  turns `ls work/` from a current-work view into archaeology.

## Best Practices 💡

- **Name agent cottages with a prefix** (`claude/<task>`,
  `bot/<task>`) so `ls work/` shows at a glance who's doing what.
- **Keep `work/main/` around as the shared read-only checkout** —
  it's the cheap way to grep the canonical tree without leaving
  your own cottage.
- **Pair this with [[in-tree-issues]]** so the same merge ritual
  governs both code and the issues that describe it.
- **Make the helper script the single source of truth.** Don't let
  the agent hook reimplement worktree creation — call the human
  script.
- **Remove cottages on merge.** If the branch is gone from `main`,
  the cottage should be too. A short cron that prunes stale,
  merged-and-empty cottages keeps the village tidy.
- **Document the convention in `CLAUDE.md`** so agents reach for
  `EnterWorktree` instead of trying to `git checkout -b` inside an
  existing cottage.

## Implementation Checklist ✅

### Set up the village

- [ ] Create the bare repo on a shared volume, owned by the project
  Unix group (e.g. `devops`).
- [ ] Create `work/` with `chmod 3775` (setgid + sticky + group
  write).
- [ ] Add `bin/new-worktree` with name sanitisation, ownership
  check, branch-from-base creation, and a final `chmod -R g-w` on
  the cottage.
- [ ] Add a `work/main/` cottage on the default branch as the
  shared read-only checkout.

### Wire the agent path

- [ ] Add `.claude/settings.json` with `WorktreeCreate` and
  `WorktreeRemove` hooks.
- [ ] Add `.claude/hooks/worktree-create` that reads the JSON from
  stdin and shells out to `bin/new-worktree`.
- [ ] Add `.claude/hooks/worktree-remove` that refuses any path
  outside `work/` and then calls `git worktree remove --force`.
- [ ] Add a top-level `CLAUDE.md` telling agents to use
  `EnterWorktree` instead of editing a shared cottage.

### Protect the village hall

- [ ] In the bare repo's hooks, set a `reference-transaction`
  policy on `main` (fast-forward-only, merge-commit-only) — see
  [[in-tree-issues]] for the shape.
- [ ] Auto-push `main` to your public mirrors from the same hook.

## Related Patterns 🔗

- [Smalltown Infrastructure 🏘️](./smalltown-infrastructure.md) —
  the bigger village this cottage sits in: small, legible, operable
  by the team you actually have.
- [In-Tree Issues 🗂️](./in-tree-issues.md) — pair the worktree
  ritual with the same merge ritual for issues; both flow through
  `main`.
- [Cuteness Pattern 🌸](../meta/cuteness.md) — why a small `ls
  work/` is friendlier than a sprawl of clones.

## References 📚

- `git-worktree(1)` — the primitive everything sits on.
- `chmod(1)` — the setgid + sticky combination (`3775`) is the
  whole permission story.
- Example implementation: this very repository's sister project
  `idmcd-devops-portal` (`bin/new-worktree`,
  `.claude/hooks/worktree-create`, `.claude/hooks/worktree-remove`).

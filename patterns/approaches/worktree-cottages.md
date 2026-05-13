---
title: Worktree Cottages 🏡
---

<!--
SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>

SPDX-License-Identifier: EUPL-1.2
-->

## Overview 📋

A bare git repository lives on one host. Inside the bare repo,
alongside `HEAD` and `refs/`, sit a `README.md`, a `.claude/`
directory, a small helper script, and a `work/` directory of
*cottages* — one git worktree per contributor, human or AI agent.
The bare repo *is* the project: there's no wrapper directory, no
parallel "checkout root", no per-clone setup. You `cd` into the
bare repo and everything you need is already there.

Filesystem permissions decide who can write where, and `git
worktree` itself guarantees that no two cottages share a branch.
The village hall (the bare repo) holds the canonical history;
every cottage is one of its worktrees.

Same trick whether you have three operators sharing a server or
one operator with three Claude agents: every actor gets a cottage,
and the bare repo they all push to lives in the same directory.

## Goals 🎯

- Give every contributor (human or agent) an isolated working tree
  without the cost of a full clone per person.
- Let teammates read each other's in-progress work without races
  or permission gymnastics.
- Push access control down to the kernel — no bespoke "who can
  touch this branch" service.
- Keep CI / linting / agent config inside the bare repo, so a fresh
  cottage works the moment it exists. No `git clone && set up
  hooks && copy .env` ritual.
- Use a single worktree-creation path for humans, the CLI, and
  agents — one code path to debug.
- Make `ls work/` the honest answer to "what's everyone working on
  right now?"

## Pattern Structure 📑

### The bare repo *is* the project

```
/srv/repos/foo.git/                   ← the project; bare repo
├── HEAD, refs/, objects/, info/      ← standard bare-repo internals
├── hooks/
│   ├── pre-receive                   ← CI gate (lint, tests)
│   └── reference-transaction         ← main-line policy
├── README.md                         ← village notice board
├── CLAUDE.md → README.md             ← symlink: same doc, two readers
├── .claude/
│   ├── settings.json                 ← wires WorktreeCreate/Remove hooks
│   └── hooks/
│       ├── worktree-create           ← delegates to bin/new-worktree
│       └── worktree-remove
├── bin/new-worktree                  ← shared helper
└── work/                             ← 3775 (rwxrwsr-t)
    ├── main/                         ← the maintainer's cottage
    ├── alice/feature-x/              ← owned by alice
    └── claude/refactor-auth/         ← owned by the user driving Claude
```

Git reserves only a handful of names inside a bare repo (`HEAD`,
`refs/`, `objects/`, `info/`, `hooks/`, `logs/`, `packed-refs`,
`config`, `description`, `worktrees/`). Everything else is yours.
So the operator config, the agent config, and the cottages all live
*inside* the bare repo itself. The result:

- **`cd foo.git/` is arrival.** Nothing exists "above" the repo.
- **Backup is one directory.** `rsync /srv/repos/foo.git/` carries
  history, hooks, agent config, and active cottages.
- **CI / lint / agent permissions travel with the repo.** A fresh
  clone of the bare repo on a new host is a fully-configured
  workspace, not a starting point.

The `.git` suffix on the directory is convention — keep it. Every
tool and every operator's muscle memory expects it.

### One branch ⇄ one cottage

Git already refuses to check out the same branch in two worktrees.
Lean on that: the branch *is* the cottage name, and the existence
of the directory is the lock. No external coordination needed.

```
$ ls work/
alice/feature-x   claude/refactor-auth   main

$ git worktree add work/feature-x -b feature/x main
fatal: 'feature-x' is already checked out at '.../work/alice/feature-x'
```

### The raw operator path

Inside the bare repo, git treats `.` as the git directory. No
`--git-dir` needed:

```bash
cd /srv/repos/foo.git
git worktree add work/my-feature -b my-feature main
cd work/my-feature
# you're home
```

Two commands. That's the whole onboarding.

### The shared helper

For day-to-day use there's `bin/new-worktree`, which wraps the
two commands above with the polish operators want:

1. **Sanitise the name** — letters, digits, `._/-`, no `..`.
2. **Refuse to steal** — if the directory exists and isn't yours,
   bail out with the owner's name.
3. **Create branch + worktree** in the bare repo (creating the
   branch from a base ref if it doesn't exist yet).
4. **`chmod -R g-w`** so the cottage is yours alone to write.

Intermediate dirs along nested names (`feature/foo`) get `3775` so
the setgid + sticky combo propagates.

### The maintainer's cottage

`work/main/` is **the maintainer's cottage**, not a shared scratch
area. It's where the person responsible for the project decides
which branches get merged into `main` — the only cottage from which
that decision is allowed to land. Everyone else can read it; only
the maintainer pushes from it.

If the bare repo's `reference-transaction` hook enforces
fast-forward-only + merge-commit-only updates to `main` (as in this
project), the rule is mechanical, not social: pushes from anywhere
else will simply be rejected.

### The village notice board

`README.md` at the bare repo's root is written for **humans
walking up to a shell prompt for the first time**. It says:

- where the bare repo lives,
- how the `work/` layout works,
- how to create a cottage with raw `git worktree add`,
- how to create one through the helper,
- how Claude users spawn one with `claude --worktree`,
- who owns `work/main/` and what that means for merging.

`CLAUDE.md` is a **symlink to `README.md`**. Agents and humans
read the same instructions; one file means they can't drift apart.

A minimal README body — the part operators actually need — reads:

````markdown
## Making your own cottage

Pick a branch name (`feature/cute-thing`, `fix/login-redirect`).
Then either:

```bash
# Raw git, for operators who like to see the mechanism:
cd /srv/repos/foo.git
git worktree add work/feature/cute-thing -b feature/cute-thing main
cd work/feature/cute-thing

# Or the helper, which also sanitises + sets g-w:
bin/new-worktree feature/cute-thing

# Or, if you're a Claude user, ask Claude:
claude --worktree feature/cute-thing
```

All three produce the same thing: a fresh worktree at
`work/<branch>` on a fresh branch off `main`, owned by you,
readable by the group.

## Who merges what

`work/main/` is the maintainer's cottage. To get your branch in,
open an MR / PR against `main`; the maintainer reviews from their
cottage and lands the merge there. Don't push to `main` from your
own cottage — the bare repo's hooks will reject it.
````

### Agent integration

`.claude/settings.json` wires Claude's `WorktreeCreate` and
`WorktreeRemove` lifecycle hooks to two tiny shell scripts that
parse the JSON on stdin and delegate to the *same*
`bin/new-worktree`. That convergence is the whole point: the
agent's `EnterWorktree` tool, a CLI invocation, and a human typing
`bin/new-worktree` all run identical code, producing identical
filesystem layouts with identical ownership rules.

```json
{
  "hooks": {
    "WorktreeCreate": [{"hooks": [{"type": "command",
      "command": "/srv/repos/foo.git/.claude/hooks/worktree-create"}]}],
    "WorktreeRemove": [{"hooks": [{"type": "command",
      "command": "/srv/repos/foo.git/.claude/hooks/worktree-remove"}]}]
  }
}
```

The remove hook is paranoid by design: it refuses to act on any
path that isn't under `work/`, then calls `git worktree remove
--force`.

Users have three entry points into a Claude session inside a
cottage:

```bash
# Spawn a fresh session that creates and enters a new cottage:
claude --worktree my-task          # explicit branch name
claude --worktree                  # let Claude pick a random one

# Or, from inside an existing session, ask Claude to call
# EnterWorktree on a branch name.
```

One code path; three front doors.

### Lifecycle

```
   idea ──► git worktree add … / bin/new-worktree / claude --worktree
                       │
                       ▼
              work/<branch>/  ← edit, commit, push to bare repo
                       │
                       ▼
              MR / merge into main      (see [[in-tree-issues]])
                       │
                       ▼
              git worktree remove        (or ExitWorktree)
```

Pushing to the bare repo is local and instant — it's right there.
If the bare repo has a `reference-transaction` hook (as in this
very project), `main` is fast-forward-only and merge-commit-only,
protecting the village hall without extra services.

### Multi-Repo Variant: Village of Villages 🏘️

For organisations that run several related bare repos, wrap them
in a thin **org directory**:

```
/srv/orgs/acme/
├── README.md                     ← onboarding for the whole org
├── CLAUDE.md → README.md
├── .claude/                      ← shared agent skills / settings
├── manifest.yaml                 ← lists member repos + their roles
└── repos/
    ├── frontend.git/             ← each is itself a cottage village
    │   ├── work/{main, alice/x, ...}
    │   └── .claude/, bin/, ...
    ├── backend.git/
    │   └── work/...
    └── infra.git/
        └── work/...
```

Each bare repo still follows the single-repo pattern internally —
the org directory adds nothing to the per-repo workflow. What it
provides:

- **Cross-repo docs.** README for "what is acme, what does each
  repo do, where do issues go?"
- **Shared agent assets.** Skills, slash commands, and prompts that
  apply org-wide live in the org's `.claude/`; per-repo `.claude/`
  inherits them by reference. (Concretely: the per-repo settings
  point at the org's hooks directory.)
- **A manifest.** Plain YAML naming each member repo, who maintains
  it, and any version-pinning rules. The manifest is the *only*
  thing humans need to read to understand the shape of the org.

What it deliberately does *not* do:

- **No org-level worktree.** Cottages live inside the individual
  repos, not at the org root. A contributor working across repos
  has multiple cottages, one per repo — same as a single-repo
  contributor with multiple branches.
- **No shared `work/main/`.** Each repo's maintainer owns their
  own `work/main/`; an org-wide merge is several per-repo merges.

A "composed cottage" — one directory holding coordinated worktrees
of several repos on related branches — is a tempting extension but
needs real tooling (multi-worktree helper, branch-bundle manifest,
cross-repo MR ritual). Don't build it until a real workflow asks
for it; until then, contributors juggle per-repo cottages by hand.

## Security Considerations 🔐

- **Input sanitisation in the helper is load-bearing.** The hook
  passes a name out of an untrusted JSON payload. Reject anything
  outside `[A-Za-z0-9._/-]` and explicitly reject `..`.
- **The remove hook must geofence to `work/`.** A typo (or a
  prompt-injected agent) shouldn't be able to delete arbitrary
  paths.
- **No network surface added.** The bare repo is reached via local
  filesystem permissions; there's no extra daemon to harden.
- **Ownership = authorship.** Every commit in a cottage is made by
  the Unix user who owns it. `git log` and `stat` agree on who did
  what — useful for audit, useful for blame.
- **Agent isolation is per cottage, not per process.** An agent
  running as user `alice` can still read everyone else's cottage.
  That's a feature for collaboration; pair it with a system-level
  sandbox if you need stronger separation.
- **`.claude/settings.json` is policy.** It controls what an agent
  is allowed to do. Treat it like CI config: review changes the
  same way you review code.

## Anti-Patterns ⚠️

- ❌ **A wrapper "project" directory around the bare repo.** The
  bare repo can hold operator config directly; adding a parallel
  directory adds a sync problem and a place to forget files.
- ❌ **A clone per user.** Wastes disk, hides in-progress work
  behind `ssh`, makes "what is everyone doing?" a coordination
  problem instead of an `ls`.
- ❌ **One shared worktree with branch switching.** Two humans (or
  a human and an agent) will collide on `git checkout` within the
  first hour. Git designed worktrees specifically to make this
  unnecessary.
- ❌ **Letting an agent share its driver's cottage.** Either you
  race the agent for the file lock, or you serialise edits by hand.
  Give the agent its own cottage with `EnterWorktree` or
  `claude --worktree`.
- ❌ **Tracking who-owns-what in Slack / a wiki / a spreadsheet.**
  `ls -l work/` is the truth; anything else drifts.
- ❌ **Replacing filesystem permissions with a CI policy gate.**
  The kernel already does access control. Don't outsource a
  primitive you already have.
- ❌ **Long-lived cottages.** A branch that lingers past its merge
  turns `ls work/` from a current-work view into archaeology.
- ❌ **Two CLAUDE.md / README.md files.** They will drift. Symlink
  one to the other and write for both audiences.

## Best Practices 💡

- **Name agent cottages with a prefix** (`claude/<task>`,
  `bot/<task>`) so `ls work/` shows at a glance who's doing what.
- **`work/main/` is the maintainer's cottage**, not a shared
  scratch area. Treat it as read-only from any other cottage;
  merges land there because that's where the merge decision is made.
- **Same README for humans and agents.** Make `CLAUDE.md` a symlink
  to `README.md` so the village notice board and the agent
  instructions can't drift apart.
- **Pair this with [[in-tree-issues]]** so the same merge ritual
  governs both code and the issues that describe it.
- **Make the helper script the single source of truth.** Don't let
  the agent hook reimplement worktree creation — call the human
  script.
- **Remove cottages on merge.** If the branch is gone from `main`,
  the cottage should be too. A short cron that prunes stale,
  merged-and-empty cottages keeps the village tidy.
- **Put CI / lint inside the bare repo's `hooks/`.** A fresh
  cottage inherits them by being a worktree of the bare repo;
  there is nothing to install.

## Implementation Checklist ✅

### Set up the village (single repo)

- [ ] `git init --bare /srv/repos/foo.git`, owned by the project
  Unix group (e.g. `devops`).
- [ ] Create `foo.git/work/` with `chmod 3775` (setgid + sticky +
  group write).
- [ ] Add `foo.git/bin/new-worktree` with name sanitisation,
  ownership check, branch-from-base creation, and a final `chmod
  -R g-w` on the cottage.
- [ ] Add `work/main/` as the maintainer's cottage. Make sure the
  maintainer (not the `devops` group) owns it.
- [ ] Write `foo.git/README.md` with: where the bare repo lives,
  raw `git worktree add` example, the `bin/new-worktree` helper,
  the `claude --worktree` entry point, and the who-merges-what
  rule for `work/main/`.
- [ ] `ln -s README.md foo.git/CLAUDE.md`.

### Wire the agent path

- [ ] Add `foo.git/.claude/settings.json` with `WorktreeCreate`
  and `WorktreeRemove` hooks.
- [ ] Add `foo.git/.claude/hooks/worktree-create` that reads the
  JSON from stdin and shells out to `bin/new-worktree`.
- [ ] Add `foo.git/.claude/hooks/worktree-remove` that refuses any
  path outside `work/` and then calls `git worktree remove
  --force`.
- [ ] Confirm the `CLAUDE.md → README.md` symlink covers the
  agent-side instructions too (use `EnterWorktree` or
  `claude --worktree`, never edit a cottage you don't own).

### Protect the village hall

- [ ] In `foo.git/hooks/`, set a `reference-transaction` policy on
  `main` (fast-forward-only, merge-commit-only) — see
  [[in-tree-issues]] for the shape.
- [ ] Add `pre-receive` / `update` hooks for CI and lint that
  every push must satisfy. They live inside the bare repo and
  travel with it.
- [ ] Auto-push `main` to your public mirrors from the same hook.

### Add a multi-repo wrapper (only if you need it)

- [ ] Create `/srv/orgs/<org>/` with its own `README.md`,
  `CLAUDE.md → README.md` symlink, and `.claude/` for shared agent
  assets.
- [ ] Place each member bare repo under `repos/<name>.git/`. Each
  still follows the single-repo checklist above.
- [ ] Add `manifest.yaml` listing the member repos, their
  maintainers, and any pinned relationships.
- [ ] Resist adding org-level `work/`. Cottages live in the repos.

## Related Patterns 🔗

- [Smalltown Infrastructure 🏘️](./smalltown-infrastructure.md) —
  the bigger village this cottage sits in: small, legible,
  operable by the team you actually have.
- [In-Tree Issues 🗂️](./in-tree-issues.md) — pair the worktree
  ritual with the same merge ritual for issues; both flow through
  `main`.
- [Cuteness Pattern 🌸](../meta/cuteness.md) — why two commands
  and an `ls work/` are friendlier than a sprawl of clones.

## References 📚

- `git-worktree(1)` — the primitive everything sits on.
- `git-init(1)` — `--bare` and the conventional `.git` suffix.
- `chmod(1)` — the setgid + sticky combination (`3775`) is the
  whole permission story.
- Example implementation: the sister project `idmcd-devops-portal`
  (`bin/new-worktree`, `.claude/hooks/worktree-create`,
  `.claude/hooks/worktree-remove`).

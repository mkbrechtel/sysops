---
status: draft
---

# Integrate cute patterns library

## Goal

Integrate the [cute patterns library](https://patterns.how) /
[mkbrechtel/patterns](https://github.com/mkbrechtel/patterns) into this
collection. Patterns are short, approachable guides to "how to work and
get stuff done" — they should inform role design *and* be directly
available to users (and AI assistants) working on managed hosts.

## Scope

Two integration surfaces:

### 1. Patterns shape role design

- Role conventions (feature flags, managed-file headers, systemd-first,
  secrets-on-host) are themselves patterns. Cross-link to the patterns
  library from `CODING.md` and role READMEs where a pattern applies.
- Use the patterns vocabulary consistently in docs.

### 2. Patterns shipped on managed hosts (especially devbox)

- Central Claude Code config on devbox hosts (see
  [devbox.feature.md](devbox.feature.md)) includes the patterns as
  **skills** under `~/.claude/skills/` so Claude picks them up by name.
- A `patterns` role (or a feature of `devbox`) installs the patterns
  content on the host: checkout of the upstream repo to
  `/usr/local/share/patterns/` (or similar), symlinked into each user's
  `~/.claude/skills/`.
- Patterns are versioned — pin to a git SHA or tag, not `main`.
- Updates are explicit (re-run role), not automatic, matching the
  claude-code version-pinning policy.

## Design notes

- Patterns repo is the single source of truth. This collection consumes
  it; it does not fork or vendor content into our tree.
- Integration is read-only from our side: we don't try to edit patterns
  from here. If a role's conventions become a new pattern, contribute
  upstream.
- If/when this collection grows its own docs site, cross-link both ways
  with the patterns site (patterns.how). Until then, role READMEs link
  out to specific patterns by URL.
- A "which patterns does this role embody?" field in role READMEs would
  tie the two together explicitly.

## Open questions

- Delivery mechanism: git clone on the target, tarball download, or
  install as a package? Git clone is simplest and keeps the history;
  tarball is faster and has no git-daemon dependency.
- Path on target — `/usr/local/share/patterns/`, `/opt/patterns/`, or
  something inside `/srv`?
- Do *all* managed hosts get the patterns, or only devbox-class hosts?
- How do individual user overrides work? A user might want their own
  skill in `~/.claude/skills/` that shadows a central one — how do we
  resolve conflicts?
- Is there a subset of patterns relevant to infra work vs general
  productivity, and do we filter on install?
- Do we want a CI check that links `[pattern: foo]` references in role
  READMEs actually resolve against the upstream patterns repo?

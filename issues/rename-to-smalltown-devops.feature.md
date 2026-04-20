---
status: draft
---

# Rename collection to smalltown-devops

## Goal

Rename the collection from `mkbrechtel.sysops` to **smalltown-devops**,
published under **smalltown-devops.patterns.how** as part of the
patterns.how family. The new name reflects the positioning: automating
infrastructure management for **small teams** — the scale where a central
platform like GitLab, Kubernetes, or a dedicated SRE team is overkill, but
you still want it to feel managed.

## Scope

- Project name: `smalltown-devops`.
- Docs / homepage URL: `smalltown-devops.patterns.how`.
- Update `galaxy.yml` (namespace + collection name — see open question on
  Galaxy FQCN).
- Update all FQCN references in roles, playbooks, docs, and tests.
- Update README tagline: "small team infrastructure management" — what a
  handful of admins can run across a few dozen hosts without a platform
  team.
- Update `CLAUDE.md`, `CODING.md`, `RELEASE.md`, `GLOBAL.md`, `REUSE.toml`
  as needed.
- Update managed-file-header convention strings
  (`mkbrechtel.sysops.<role>` → new form).
- Update GitHub Actions / galaxy publishing workflow.
- Git repo rename (separate, manual step).

## Design notes

- Breaking change. Acceptable during 0.x.x.
- Homepage on `smalltown-devops.patterns.how` leads with the smalltown
  framing: what problems it solves, what scale it targets, what it
  deliberately skips. Lives in the same Astro site family as patterns.how
  (or cross-linked from it).
- The rename is a one-pass search/replace but needs review — many files
  hardcode the old FQCN.

## Open questions

- Ansible Galaxy FQCN: what goes in `galaxy.yml`? Options:
  - `mkbrechtel.smalltown_devops`
  - `smalltown.devops` (new namespace)
  - `patterns.smalltown_devops` (patterns-family namespace)
  
  Galaxy requires `namespace.name`, underscores only — no hyphens, no
  extra dots. The URL `smalltown-devops.patterns.how` is branding; the
  Galaxy name is a separate technical choice.
- Docs site integration: is `smalltown-devops.patterns.how` a
  **subdomain** served by its own Astro build, or a **path** under
  patterns.how? Impacts deployment topology.
- Deprecation: do we publish one last `mkbrechtel.sysops` pointing users
  to the new name, or cut clean?
- GitHub repo rename (`mkbrechtel/sysops` → ?) — do it as part of this
  ticket or separately?
- "Small team" definition for the README — 1–5 admins? ~10–50 hosts?
  Spell it out so prospective users can self-select.

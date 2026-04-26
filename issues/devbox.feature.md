---
status: draft
---

# Devbox setup

## Goal

A "devbox" host profile that exposes a browser-based development environment:
ttyd (terminal), code-server (editor), and a managed Claude Code install. Each
user works in their own context; the system provides guardrails.

## Scope

- `ttyd` (exists) — browser terminal, per-user or shared auth TBD.
- `code-server` — browser VS Code per user.
- `claude_code` — system-wide install at a **pinned, configurable version**,
  with user-level auto-update *disabled*. Installed via npm (global) or direct
  download — whichever gives us clean version pinning.
- Per-user authentication for Claude Code (each user runs `claude login`).
- Centrally managed configuration pushed to each user's `~/.claude/`:
  - A system prompt fragment with guardrails appropriate for this host.
  - Central skills (symlinks / copies into user config).
  - Central MCP server list.
- All three services behind the reverse proxy (see `reverse-proxy.pattern.md`).

## Design notes

- Shared config lives at `/etc/claude-code/` (or similar) and is
  linked/rendered into each user's home on login or via role run.
- Users can add their own skills/MCPs; central config merges rather than
  overwrites.
- `devbox` role = orchestrator that composes `ttyd`, `code-server`,
  `claude_code`, plus the `users` role.

## Open questions

- npm global vs direct tarball download? npm gives us the package; direct
  download removes npm as a dependency and lets us pin an exact binary.
- How do we prevent users from `npm i -g @anthropic-ai/claude-code@latest`
  themselves — rely on filesystem perms, or just document it as a policy?
- How are central skills/MCPs delivered — checked in here, or pulled from a
  separate git repo?
- Does the central system prompt apply per-invocation (injected) or is it
  just a `CLAUDE.md` shipped into every user's home?
- Is code-server also version-pinned? Same treatment as claude-code?
- Auth for ttyd / code-server: basic auth behind RPX with TLS, or something
  stronger (OIDC)?

---
status: draft
---

# User management

## Goal

Document what the `users` role should cover for local user management.
Implementation is largely in place; this ticket is the specification so we can
spot gaps.

## Scope

- Local user accounts with:
  - UID/GID control (explicit or auto).
  - Primary and secondary groups.
  - Shell (bash/zsh/fish).
  - Home directory creation with a configurable skeleton.
- SSH key management:
  - Keys from files in the collection.
  - Keys fetched from GitHub / GitLab profiles.
  - (Later) keys from an LDAP/SSO source.
- Sudo policy per user or per group.
- Per-user dotfile deployment (links to a user's dotfiles repo, or central
  skeleton).
- Per-user systemd services enabled (lingering on for service persistence).

## Design notes

- `users` takes a list of user specs, each with the above keys.
- Default groups / sudo are opt-in via `_with_` flags (e.g.
  `users_with_sudo`).
- Removal of users is explicit (state: absent); accidental removal must
  require explicit intent.

## Open questions

- What's currently missing vs this list? (Review `roles/users` and note.)
- SSH key sources — is GitHub fetch already supported? At what cadence do
  we refresh?
- Should dotfile management live in this role, or be its own `dotfiles` role
  that `users` can call per user?
- How do we deprovision — archive home, delete, or leave `deleted_<name>`?

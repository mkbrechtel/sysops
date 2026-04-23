---
status: draft
---

# SSH server role

## Goal

A dedicated `ssh_server` role that owns `sshd` configuration on managed hosts. Today the collection has `ssh_agent` (user-side) but no server-side counterpart — `sshd_config` is whatever the distro ships. Needed now because the `users` role is moving authorized_keys into a central `/etc/ssh/authorized_keys.d/<user>` layout, which requires an `AuthorizedKeysFile` directive that something has to own.

## Scope

- Manage `/etc/ssh/sshd_config` (and/or a drop-in under `/etc/ssh/sshd_config.d/`) with opinionated defaults: no root password login, no password auth by default, modern KEX/cipher set, `AuthorizedKeysFile /etc/ssh/authorized_keys.d/%u` to match the `users` role.
- Host key management: leave host keys to the distro's first-boot generation; do not overwrite.
- Handler restart via `systemctl reload ssh` with a `sshd -t` validation gate so a bad config never gets applied.
- Hook for the `users` role to drop per-user authorized_keys files into `/etc/ssh/authorized_keys.d/` without this role fighting it.

## Open questions

- **Password auth default.** Off entirely, or on-but-per-host-overridable? Leaning off by default — key-only is the expectation.
- **Port / ListenAddress.** Keep defaults and let sites override, or expose a first-class variable?
- **Relationship to `users` role.** Does `ssh_server` own the `/etc/ssh/authorized_keys.d/` directory itself (creation + mode), and `users` only drops files into it? Probably yes — keeps ownership clean.

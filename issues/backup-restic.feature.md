---
status: draft
---

# Backup — restic path

## Goal

Working restic client + server roles as one of two backup paths (alongside borg)
to satisfy 3-2-1. Per-project isolation on the server side.

## Scope

- `restic_client`: installs restic, defines one or more backup *jobs* (paths,
  schedule, retention, repo target).
- `restic_server`: serves restic repos over REST, with per-project isolation
  (separate repos, separate credentials).
- Supported targets:
  - Self-hosted restic-server (this role).
  - Hetzner Storage Box (SFTP/rclone).
- Scheduling via systemd timers.
- Retention policy (`forget` / `prune`) as a separate scheduled unit.
- Restore procedure documented, not necessarily automated.

## Design notes

- A client can push to *multiple* repos (3-2-1: one local/near, one offsite).
- Encryption password handled via the secrets role (see `secrets.feature.md`).
- Server: one repo per project, enforced by restic-server's `--private-repos`
  or separate user accounts.
- Monitoring: check instance that verifies last snapshot age.

## Open questions

- Repo layout on server: `/<project>/<host>` or `/<host>/<tag>`?
- How are client credentials provisioned on the server — pre-seeded htpasswd,
  or generated per-host and pushed back into the secrets store?
- Do we want `restic check` (integrity) as a separate scheduled job, and where
  does it run — client or server?
- Hetzner Storage Box: restic-over-SFTP, or rclone backend? (rclone enables
  more targets but adds a dependency.)

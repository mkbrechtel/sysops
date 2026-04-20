---
status: draft
---

# Backup — borg path

## Goal

Working borgbackup client + server roles as the second backup path (alongside
restic) to satisfy 3-2-1. Per-project isolation on the server side.

## Scope

- `borg_client`: installs borg, defines backup *jobs* (paths, schedule,
  retention, repo target).
- `borg_server`: hosts borg repositories via SSH with forced `borg serve`,
  append-only option per repo, per-project isolation.
- Supported targets:
  - Self-hosted borg server (this role).
  - Hetzner Storage Box (SSH + borg).
- Scheduling via systemd timers.
- Retention (`borg prune`) as a scheduled unit.
- Restore procedure documented.

## Design notes

- Append-only mode is the default for offsite repos; a separate "maintenance"
  key can run prune.
- Each client gets its own SSH key; server uses `command=` in authorized_keys
  to restrict to its own repo path.
- Encryption passphrase via secrets role.
- Monitoring: last-archive age check.

## Open questions

- Repo-per-host vs repo-per-project-per-host — what's the unit of isolation?
- How do we handle the append-only vs prune split? Two keys per client, or a
  separate "maintenance host" that prunes?
- Do we want `borg check` on the server side on a schedule?
- Hetzner Storage Box: they provide native borg — do we lean on that, or
  treat it like any SSH target?

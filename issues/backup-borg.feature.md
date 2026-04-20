---
status: draft
---

# Backup — borg path

## Goal

A working borg backup path as the **second, deliberately minimal** backup solution alongside restic. Where the restic path is our full-fledged self-managed cascade (client / server / custodian, per-project aggregation, offsite via Hetzner Storage Box, all parts specified down to systemd drop-ins — see `backup-restic.feature.md`), the borg path is the opposite stance: **import an upstream community Ansible role, set defaults, add run monitoring, move on.** The point is diversity — a second path with a different codebase, different format, different trust model — not a second fully-engineered system.

Together the two paths give us the "2 media" leg of 3-2-1: if a bug, supply-chain compromise, or format quirk takes out restic, borg is still there, and vice versa.

## Scope

### Strategy: thin wrapper over an upstream role

We do not implement a borg role from scratch. A small `borg_client` role in this collection:

- Pulls in [BorgBase's `ansible-role-borgbackup`](https://github.com/borgbase/ansible-role-borgbackup) as the upstream dependency.
- Sets collection-appropriate defaults (backup paths, retention policy, schedule) so a typical host gets a sensible borg backup by just including the role.
- Drops in the monitoring bits that upstream does not provide (see Monitoring below).

No custom server role. Clients push directly to:

- **Borgbase** (hosted) as the common case — credit-card-and-go, they run the server.
- **Hetzner Storage Box** via their native borg endpoint as an alternative.

Either way the remote is someone else's problem to operate. This is the whole point: restic is the "we run it" path, borg is the "someone else runs it" path. No central custodian, no offsite cascade we operate — borg's remote *is* the offsite.

### What's in scope

- Thin `borg_client` role that wraps the upstream role with our defaults.
- Host-level filesystem backup (equivalent of restic's `fs-` class). Service-level and VM-level backups stay on the restic path; borg is the dumb filesystem second line.
- Encryption passphrase provisioned via the secrets role, same machinery as restic.
- A monitoring check that watches the last borg run and alerts on failure or overdue (hooks into the checker framework, same pipeline as the restic client-side run check).
- Basic restore documentation — the upstream role's restore story plus a short how-to for our defaults.

### What's out of scope

- No self-hosted `borg_server` role. If a site wants one, they run one outside this collection.
- No custodian, no offsite replication we drive, no project-level aggregation. Borg does not need them in this design — the remote is already offsite and already operated by someone else.
- No append-only-vs-prune key split engineered by us. Whatever the upstream role and Borgbase/Hetzner support is what we use.

## Design notes

- **Why a second path at all.** Restic is extremely well-specified in this collection, but one bug or one format-level mistake could still take it out. Borg is a completely independent codebase, format, and crypto stack. Running both in parallel is cheap (borg is one more systemd timer on the client) and gives us genuine redundancy.
- **Why not symmetric with restic.** Symmetry would mean a second custodian, second offsite cascade, second base-repo scheme — doubling the ongoing maintenance burden for a backup we hope never to need. The whole point of the second path is *low effort*. A thin wrapper over a battle-tested upstream role, pointed at a hosted remote, is the minimum that still delivers on "we have a second backup."
- **Why Borgbase / Hetzner native borg.** Both are cheap, well-operated, and built specifically for borg. Running our own borg server reintroduces exactly the ops burden we are trying to avoid by having a second path. If we wanted "our own server", that is already what restic is for.
- **Why no custodian for borg.** The custodian pattern exists in the restic path because restic's topology splits password access across client/server/offsite in a way that benefits from a dedicated observer. Borg's topology is one-shot (client → remote), no split, so there is nothing for a custodian to observe that the client's own run check does not already cover.
- **Scope of the collection's contribution.** The upstream role does the heavy lifting. Our `borg_client` role contributes: opinionated defaults, integration with the secrets role, integration with the checker/alerting framework. Three thin layers, nothing load-bearing that upstream does not already provide.

## Open questions

- **Remote choice by default: Borgbase or Hetzner?** Both work; Borgbase is purpose-built for borg and has a sane UI, Hetzner is cheaper and already used by the restic path. A site can pick either — which one do we document as the default happy path?
- **Run monitoring mechanism.** Does the upstream role already emit something we can consume (exit code file, status dir, stdout we capture), or do we wrap the invocation in a systemd unit of our own that writes a status file? The latter is more work but gives us a uniform pipeline with the restic client-side run check.
- **Where does the passphrase live on disk?** The secrets role's env-typed secret is the natural fit (same pattern as restic's `restic-repo.auth.env`), but the upstream role may expect a different shape. Adaptation layer may be needed.

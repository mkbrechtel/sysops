---
status: draft
---

# Backup — restic path

## Goal

Working `restic_client`, `restic_server`, and `restic_monitor` roles as one of two backup paths (alongside borg) to satisfy 3-2-1. The three roles are the load-bearing triad: the client produces snapshots, the server holds the near repos password-free, the monitor carries read-only keys and does all the work that needs keys (freshness checks, integrity checks, offsite replication).

Per-host isolation on the server side, two backup levels on the client side, monitored end-to-end, single near-target for clients with offsite cascade handled by the monitor.

## Status quo

The collection already ships working `restic_client` and `restic_server` roles inherited from an earlier project. This ticket defines the target shape. Implementation is a rework that keeps what works (systemd timer + oneshot pattern, client-generates-then-delegates htpasswd) and swaps out the parts that do not scale (single-server URL as the only config knob, inline plaintext `key:` in inventory, two flavors of job in one list, no monitoring role at all, no offsite cascade, no integrity checks). The new `restic_monitor` role is net-new and is specified in `backup-monitor-host.feature.md`.

## Scope

### Backup levels

- **host** — filesystem paths on the guest.
- **service** — streamed export from a running service (database dump, Keycloak realm export, Grafana dashboards, virtual-machine export, disk image) via restic's `--stdin` mode. Depends on `service-export-import.feature.md`.

Machine-level backups (VM disk volumes) are not a distinct level at this role — they are produced at the hypervisor layer and appear to this role as service-level streamed exports from the hypervisor's point of view, writing into a per-project `<project>/vms` repo.

### Default repo model

A host belongs primarily to one project. Central shared infrastructure lives in a project conventionally named `<org>-infra`; separate backup infrastructure lives in a project conventionally named `<org>-backup`. The canonical definitions of *project*, *service*, *host* and the conventional naming patterns live in `project-service-terminology.feature.md`; this ticket assumes them.

Near repo path on the server: `/<project>/<host>`. One repo per host — all jobs on that host write snapshots into the same repo, separated by `--tag` and snapshot paths, which is restic's native pattern. Project is an organizational prefix, not an enforced trust boundary. Enforcement is at the host level: one restic-server user per host, one access-allowed path prefix per user, enforced by `--private-repos`. Hosts within a project are isolated from each other's backups.

Offsite repo layout is **per-project, aggregated**: `/<project>` on Hetzner Storage Box. All hosts in a project feed into one offsite repo via `restic copy` from the monitor (details in the cascade section). This unlocks cross-host dedup at the offsite layer — similar VM images, shared OS packages, the same config files across a fleet — because restic's content-defined chunking dedups identical chunks across snapshots within a repo.

Blue-green deployments within a project — and the use of restore-into-green as a DR drill — are tracked separately in `blue-green-deployments.feature.md`.

### Project base repo for efficient deduplication

Restic's content-defined chunking uses a per-repo crypto parameters picked at `restic init`. Two repos with different crypto parameters cannot dedup against each other — `restic copy` still works, but every chunk ends up re-stored on the destination, losing the point of aggregation.

Per-project chunker alignment uses a dedicated **project base repo** as the crypto parameter anchor:

1. Project bootstrap creates `/<project>/.restic-base-repo` on the backup server — an empty repo whose sole purpose is holding the project's chunker polynomial. It never receives real snapshots.
2. Every other repo in the project — near repos on the server, the `<project>/vms` repo, the aggregated offsite repo on Hetzner — is initialized with `restic init --copy-chunker-params --from-repo …/<project>/.restic-base-repo`. Same crypto parameters, dedup works across the whole project.
3. Init is idempotent: on re-run, the monitor (which owns this ritual — see `backup-monitor-host.feature.md`) checks whether a repo already exists; if yes, it verifies the chunker polynomial matches the project base and warns if it does not match.

Using a dedicated base repo rather than "whichever repo was first" removes ambiguity about which polynomial a project uses and makes the bootstrap ritual explicit: create `/<project>/.restic-base-repo`, then every subsequent repo is derived. The base repo is tiny (just metadata) and can be read-shared project-wide so any future init from any host works without project-admin credentials.

This same mechanism resolves the hypervisor-VM dedup case: the project's `<project>/vms` repo shares the project's crypto parameters via the base repo, so identical OS images across VMs dedup into a common chunk pool inside that repo and again inside the offsite aggregation.

**Access to the base repo under `--private-repos`.** restic-server's private-repos mode restricts each user to paths matching their username. For the base repo at `/<project>/.restic-base-repo`, that means the user account itself has to be named `<project>/.restic-base-repo`. The base-repo password is stored only on the monitor (provisioned via the secrets role, same machinery as per-host repo credentials).

The client still owns the init code path. When a near repo needs to be initialized, the controller fetches the base-repo password from the monitor into a transient Ansible fact (`no_log` throughout), runs `restic init --copy-chunker-params --from-repo …/<project>/.restic-base-repo` on the client with both passwords in its environment, and the fact is discarded at the end of the play. The base-repo password never lands on the client's disk; the init logic lives in one place (the client role) and is not duplicated on the monitor.

Whether restic-rest accepts usernames containing `/` under `--private-repos` is tracked as an open question below — the entire per-host / per-project scoping scheme here depends on it.

### Cross-host deduplication

The aggregated per-project offsite repo on Hetzner deduplicates across all hosts in the project. That is the main storage payoff of the cascade — twenty similar hosts do not produce twenty copies of shared content offsite, they produce one.

Whether the server's near repos also share storage on disk across hosts (via a hardlink pass on `/srv/restic`) is an empirical question, not assumed. Tracked below.

Cross-project dedup never happens — different chunker parameters, different repo keys, by design.

### Topology — cascade

Three tiers, three roles:

1. **Client → near.** Clients write directly into their per-host repo on `restic_server` via REST, using their own credentials. The server holds no decryption keys, ever.
2. **Monitor → offsite.** The `restic_monitor` host holds read-only credentials for every near repo in its project (for freshness/integrity checks) and write credentials for the per-project offsite repo on Hetzner. Scheduled `restic copy` pulls from each near repo into the aggregated offsite repo. Cryptographically verified, deduplicated, no passwords on the server.
3. **Admin → airgapped (optional).** Weekly ritual, off-line key escrow, see `airgapped-escrow.feature.md`.

The 3-2-1 rule — 3 copies, 2 media, 1 offsite — is satisfied at the *collection* level when both restic and borg run in parallel:

- 3 copies: source filesystem, restic-server (near), Hetzner Storage Box (offsite); borg adds its own independent copy.
- 2 media: restic's disk-based cascade is one medium; borg's separate repo format on different storage is the second.
- 1 offsite: Hetzner Storage Box.

Restic alone delivers 3 copies and 1 offsite but only 1 medium — borg is what closes the rule. The two paths are deliberately asymmetric: restic favors a self-hosted near target with monitor-driven offsite; borg favors hosted-as-a-service (Borgbase or direct-to-Hetzner), tracked in `backup-borg.feature.md`.

Clients never configure an offsite repo directly. If a client needs a snapshot in a second repo (not a copy of an existing snapshot — a genuinely distinct second snapshot), that is modeled as a second job with a different `repo:`. "Same snapshot in two places" is always the monitor's job via `restic copy`.

### Client — `restic_client`

Three top-level lists, no conditional fields, no fan-out per job. Default ships a single full-filesystem job against the near repo and no stream jobs:

```yaml
restic_repos:
  near:
    url: "rest:https://backup.internal:8000/"
    username: "{{ ansible_hostname }}"
    auth_secret: { service: restic, name: near-auth }       # env-typed secret

restic_file_backup_jobs:
  - name: root                          # the default job
    paths: ["/"]
    excludes:                           # virtual + volatile filesystems
      - /proc
      - /sys
      - /dev
      - /run
      - /tmp
      - /var/tmp
      - /mnt
      - /media
    flags: ["-x"]                       # stay on the root filesystem; other mounts need their own job
    repo: near
    retention: { keep_daily: 365 }
    schedule: daily

# restic_stream_backup_jobs is empty by default. Shape (once the stream contract
# is finalized in service-export-import.feature.md):
# restic_stream_backup_jobs:
#   - name: pg
#     stream_command: "…"                 # how to read the service-published export; TBD
#     stdin_filename: pg-dumpall.sql
#     repo: near
#     retention: { keep_daily: 365 }
#     schedule: daily
```

The default `root` job backs up everything on the root filesystem, with `-x` preventing descent into other mounts (`/var/lib/data`, `/home` on a separate partition, etc. — those go in their own jobs). The excludes list covers kernel virtual filesystems (`/proc`, `/sys`, `/dev`, `/run`), scratch space (`/tmp`, `/var/tmp`), and mountpoint shells (`/mnt`, `/media`). Distribution-specific additions (say `/var/lib/containers/storage` for a podman host) extend the list via role defaults, not inventory.

**Uniqueness assertion**: the role refuses to template if any `name` appears in both lists, or appears twice in either list. One flat namespace for jobs. Caught early, not at systemd-enable time.

### Client filesystem layout

Everything the role owns lives under `/etc/restic/`. Secrets live under `/etc/secrets/` (the secrets role's territory). The auth file crosses the boundary via a symlink, the pattern the secrets role explicitly sanctions:

```
/etc/restic/
  repos/
    <repo>/
      restic-repo.env          # RESTIC_REPOSITORY, RESTIC_CACERT, base flags
      restic-repo.auth.env     # symlink → /etc/secrets/restic/<repo>-auth.env (RESTIC_PASSWORD=…)
  jobs/
    <job>/
      restic-backup-job.env    # job-specific: RESTIC_FLAGS (flags + paths, or --stdin + filename)
      repo                     # symlink → ../../repos/<selected-repo>/
```

A repo's identity is the two files in its directory: `restic-repo.env` carries the public config (URL, cert path), `restic-repo.auth.env` carries the passphrase via symlink. Both are repo-level by design — the passphrase belongs to the repo, not the job, which is why it lives inside `repos/<repo>/` next to the URL.

Jobs reach their repo through one symlink — `jobs/<job>/repo` → `../../repos/<selected-repo>` — so the unit file loads both repo fragments via a stable nested path. Changing a job's repo is a single symlink flip.

One flat `jobs/` dir holds both file and stream jobs (uniqueness assertion makes that safe). The systemd template decides which ExecStart fires based on which list the job came from.

### Backup user isolation

Backup jobs do not run as root by default. One system user — `restic-backup-client` — owns every backup unit on the host. The server-side REST daemon runs as its own separate user (`restic-backup-server`, see the Server section); a host that acts as both client and server keeps the two envelopes distinct.

**File-mode jobs** get `AmbientCapabilities=CAP_DAC_READ_SEARCH` and a matching bounding set. That capability bypasses DAC read-and-traverse checks without granting write or any other privilege — restic gets the "read anything" power it needs for whole-filesystem backups, and nothing else. restic itself is not setuid and gets no extra capabilities.

**Stream-mode jobs** also run as `restic-backup-client`, without `CAP_DAC_READ_SEARCH` — they do not read paths directly; they consume a stream produced by a service-owned export endpoint. What this ticket commits to: `stream_command` stays inventory-configurable, and the backup unit runs as the backup-client user without per-service privilege. How the stream actually reaches restic (socket activation, FIFO, systemd `StandardInput=` plumbing, a helper command) and how the producing service publishes it are out of scope here; that contract is defined in `service-export-import.feature.md` when it lands. Likely over socket communication, but the exact wiring is that ticket's job.

### Run-as-root escape hatch

When `CAP_DAC_READ_SEARCH` is not enough — MAC policies (SELinux, AppArmor) blocking read access, encrypted-at-rest filesystems with keys outside the `restic-backup-client` session keyring, niche kernel interfaces that ignore the capability — a per-job `run_as_root: true` override switches that job's unit to `User=root`.

Implementation: the role drops `/etc/systemd/system/restic-backup@<job>.service.d/run-as-root.conf` with `[Service]\nUser=root\nAmbientCapabilities=\nCapabilityBoundingSet=CAP_SYS_ADMIN CAP_DAC_READ_SEARCH ...` (reset to systemd's normal root defaults). Single template unit stays; the per-job drop-in is the only place `root` appears.

This is an escape hatch, not a default. Each `run_as_root: true` job should carry a comment in inventory explaining *why* the capability path does not suffice. The docs-site page for this role calls out the pattern with a visible warning so new operators do not reach for it as a convenience.

### Unit model

One template, `restic-backup@.service`, covers every job. ExecStart is env-driven; the per-job `.env` fills `RESTIC_FLAGS` with whatever the job needs.

```ini
# /etc/systemd/system/restic-backup@.service
[Unit]
Description=restic backup job %i

[Service]
Type=oneshot
User=restic-backup-client
AmbientCapabilities=CAP_DAC_READ_SEARCH
CapabilityBoundingSet=CAP_DAC_READ_SEARCH
NoNewPrivileges=yes
EnvironmentFile=-/etc/restic/jobs/%i/repo/restic-repo.env
EnvironmentFile=-/etc/restic/jobs/%i/repo/restic-repo.auth.env
EnvironmentFile=-/etc/restic/jobs/%i/restic-backup-job.env
ExecStart=/usr/bin/restic backup $RESTIC_FLAGS
```

Per-job `.env` examples:

- File-mode job: `RESTIC_FLAGS="-x /etc /var/lib"` (flags and paths).
- Stream-mode job: `RESTIC_FLAGS="--stdin --stdin-filename pg-dumpall.sql"`.

Stream-mode jobs share the same capability envelope as file-mode — a deliberate simplicity trade. The mitigation is on the stream interface: once `service-export-import.feature.md` defines the import/export socket protocol, what stream-mode jobs actually exchange is bounded by that protocol, not by the capability bit. One template keeps the timer battery, the monitoring checks, and the operator mental model all keyed to a single unit name.

`EnvironmentFile=-` tolerates missing files (belt-and-braces for the auth chain on first run). Per-job customization — the `run_as_root: true` escape hatch, stream-mode stdin wiring (mechanism deferred to `service-export-import.feature.md`) — lives in systemd drop-ins under `/etc/systemd/system/restic-backup@<job>.service.d/*.conf`. The template stays canonical; overrides are additive files the role manages.

### Scheduling — timer battery

The role ships a fixed set of named cadence timers the site can opt into — the "timer battery" pattern already used by the checker and deploy roles. Cadences cover the range most backups ever want: `hourly`, `quarter-daily`, `daily`, `weekly`, `monthly`. Each timer fires a per-cadence target that pulls in the jobs opted into that cadence via `Wants=`.

Per job the admin sets `schedule:` to one of those names. The role wires `restic-backup-<cadence>.target` ⊂ `Wants=` into `restic-backup@<job>.service`. Jobs within a cadence run sequentially (explicit `After=` between them) so failure is attributable per job and total bandwidth stays bounded.

Retention (`restic forget --prune`) runs per repo, not per job — snapshots share the repo, and `forget` applies policy across the whole repo's tags. Default is `keep_daily: 365`, and the same policy applies to the near repo and the aggregated offsite repo. Generous cutoffs plus this symmetry mean replication timing drift cannot strand a snapshot: anything `forget` would remove has long since been copied by the monitor. Scheduled via the same timer battery.

Integrity checks do **not** run on the client. They run on the monitor — see `backup-monitor-host.feature.md`.

### Server — `restic_server`

Serves repos over REST with `--private-repos`, running as a dedicated `restic-backup-server` system user (distinct from the client-side `restic-backup-client` so hosts that run both roles keep separate privilege envelopes). Per-host user accounts authenticate via htpasswd; access scoped to `/<project>/<host>/*`. Restricted-deletion directory layout, rooted at `/srv/restic/`.

TLS via the certificates role (see `secrets.feature.md`), replacing the self-signed cert + `GODEBUG=x509ignoreCN=0` workaround. The same certificates role also provides the SSH host-key material used to pin the Hetzner Storage Box endpoint on the monitor side.

**The server does not hold client repo passwords.** It is a dumb REST endpoint that authenticates and serves pack files. All password-bearing work (replication, integrity check, freshness check, repo init with chunker-param coordination) happens on the monitor. This is a deliberate split — the largest and longest-lived host in the cascade is the one that never gets to decrypt anything.

Offsite replication is not the server's job — it is the monitor's. See `backup-monitor-host.feature.md`.

### Credential provisioning

For each `(host, repo)` pair the client talks to, the controller orchestrates a three-way handshake. Keeps the existing `perserver.yaml` pattern's shape (client-generates, controller-carries, server-ingests) but swaps the source and transport:

1. On the client, the secrets module provisions `restic/<repo>-auth.env` (env-typed secret with `RESTIC_PASSWORD=…`) with source `random`, and returns the value to the controller via `fetch: true`. No plaintext lives in inventory.
2. On the controller, the value is hashed with `password_hash('bcrypt')` (Jinja filter, no new primitive).
3. A delegated task on the matching `restic_server` ensures the `(username, bcrypt)` line exists in `/srv/restic/.htpasswd`. Idempotent on re-run — the line is keyed by username so rotation replaces in place.
4. Every task touching the password has `no_log: true`.

The monitor receives its read-only copy of the same password through the same mechanism: the secrets module fetches the client's `restic/<repo>-auth.env` to the controller, and a delegated task installs it as the monitor's read-only credential for that repo. Monitor-side write credentials for the offsite Hetzner repo are provisioned independently.

Repo init runs on the client as a one-time bootstrap task. The controller fetches the project's base-repo password from the monitor (transient fact, `no_log`), then runs `restic init --copy-chunker-params --from-repo /<project>/.restic-base-repo` on the client with both the base-repo and the new repo's passwords in environment. The base-repo password is never written to disk on the client. See the project-base-repo section for the rationale and access model.

Rotation goes through the secrets-role rotation machinery. A post-rotate hook on the client re-keys the restic repo (`restic key add` new, verify, then `restic key remove` old) and triggers the controller-side htpasswd and monitor-side-credential updates via deferred plays. The bcrypt htpasswd entry is treated as a public-key derivative of the stored secret — computed on demand, not stored separately under `/etc/secrets/`.

The controller needs SSH access to client, server, and monitor during provisioning — the normal deploy assumption. The value transits via Ansible facts with `no_log`, never written to disk on the controller.

### Monitoring

Client-side:

- **Run check** — reads systemd unit state and exit code of the last `restic-backup@<job>.service` run. Alerts on failed exit or overdue timer.

Everything else (freshness check, integrity check, on-demand integrity trigger, offsite replication, on-line key escrow) lives on the monitor host — see `backup-monitor-host.feature.md`.

### Restore

Two paths, both admin-initiated, no systemd unit and no `.env` per job.

**1. File / raw restore via `restic-with-repo`.** A thin shell wrapper at `/usr/local/bin/restic-with-repo` that sources the repo's env fragments and execs restic with whatever args come next:

```sh
#!/bin/sh
# restic-with-repo <repo> <restic args...>
repo="$1"; shift
. /etc/restic/repos/"$repo"/restic-repo.env
. /etc/restic/repos/"$repo"/restic-repo.auth.env
exec restic "$@"
```

Typical use in an emergency:

```
restic-with-repo near snapshots
restic-with-repo near restore <snapshot-id> --target /tmp/recover
```

No surprise behaviour, no "restore to state X" declaration — just the env composition so admins do not hand-assemble `RESTIC_REPOSITORY` and `RESTIC_PASSWORD` at 3am.

**2. Service-level restore via the bidirectional stream contract.** For services that were backed up in stream mode, restore is the inversion of export: the service role publishes an *import* socket alongside its export one, a backup-side helper streams the chosen snapshot from the repo into that socket, the service ingests. Both directions are defined by `service-export-import.feature.md`. This is also the mechanism the blue-green DR drill uses — stream the latest snapshot from the offsite (or near) repo into a fresh green environment's import socket, smoke-test, promote or discard.

Continuous restore exercise via blue-green is **service-level only** (path 2). A streamed service export restores cleanly into a fresh blue-green slot and can be smoke-tested there. Host-level (filesystem) restore targets a live system and does not map cleanly to blue-green; machine-level (VM) restore needs hypervisor cooperation. The nightly drill described in `blue-green-deployments.feature.md` operates on stream-mode jobs via path 2.


## Design notes

- The three-role triad — client, server, monitor — splits duties along a password-access boundary. Server never decrypts, client only reads its own repo, monitor holds read-only project-wide keys plus offsite write credentials. This makes the server the simplest component despite being the longest-lived.
- The three-list client config shape (`restic_repos`, `restic_file_backup_jobs`, `restic_stream_backup_jobs`) replaces the single `restic_client_backup_directives` list with its `database_dump_command`-vs-`directories` discriminator. Two job lists at the inventory layer (clearer schemas, no union typing) collapse to one systemd template at runtime (one timer battery, one monitoring key, one place to edit).
- A job has exactly one repo. "Two repos means two snapshots" is a restic property: `restic backup` produces a new snapshot in each target repo, and those are semantically different snapshots, so they belong in different jobs. Multi-place-same-snapshot exists only via `restic copy`, which is the monitor's job.
- Project-level chunker-parameter alignment (a dedicated `/<project>/.restic-base-repo` repo; every other repo init'd with `--copy-chunker-params --from-repo /<project>/.restic-base-repo`) is what makes the aggregated per-project offsite repo actually dedup. Skipping this step silently turns the offsite repo into a fan-in of unrelated chunk pools — correct but wasteful.
- One template, no wrapper script, argv-only ExecStart (`restic backup $RESTIC_FLAGS`). Both job classes share the privilege envelope (`CAP_DAC_READ_SEARCH`). The stream-mode mitigation is on the protocol side — `service-export-import.feature.md` bounds what stream jobs actually do, not the capability bit.
- Composition via layered `EnvironmentFile=` + a single per-job `repo/` symlink directory lets each fragment have a single owner. The job's repo choice is one symlink; changing the repo is one `ansible.builtin.file` task.
- Backup does not run as root. `CAP_DAC_READ_SEARCH` covers file-mode and (since the template is shared) also applies to stream-mode. The backup user's privilege envelope stays "read any file" — not "run any command as any user".
- Stream producers and consumers (import for restore) are not the backup role's concern. Service roles publish bidirectional stream endpoints; this role consumes the export direction and drives the import direction during restore. The interface — how publication and consumption wire together — lives in `service-export-import.feature.md`.
- The per-project aggregated offsite repo enables cross-host dedup at the offsite layer without violating the per-host isolation at the near layer. `restic copy` respects chunking, so the wire cost is proportional to *new* content, not to the near repos' sizes.
- Cross-host dedup at the offsite step works because `restic copy` identifies chunks by a hash of the content itself, so identical content from different hosts collapses into a single stored copy on the destination. Every repo in the cascade also does its own within-repo dedup automatically.
- Cross-host dedup at the server's filesystem layer — a hardlink pass over `/srv/restic` that collapses identical files across different hosts' near repos — depends on whether restic writes byte-identical files to disk for identical inputs. We have not empirically confirmed which way this goes. If it works, it is a pure storage win on top of what the offsite cascade already provides; if it does not, the pass finds no matches and wastes cycles. An opt-in server flag that schedules the pass only makes sense once this is measured.
- Sharing disk blocks across near repos (if hardlink dedup turns out to work) supposedly does not let one host's operator read another host's backups. The per-repo index that maps stored blobs back to snapshots stays gated by each repo's password; the files on disk without that index are opaque content-addressed storage. Still worth confirming during the same empirical check before relying on the expected disk space saving properties.
- The bcrypt htpasswd entry is a public-key-shaped derivative of the stored password — computed, not stored. Matches the SSH authorized-keys pattern.
- Retention symmetry (same policy on near and offsite) plus a generous `keep_daily: 365` default sidesteps the cascade-timing-drift problem without explicit `After=`/`Before=` ordering between replication and prune.

## Open questions

- **restic-rest user scoping for path-like usernames**. The whole per-host / per-project isolation scheme assumes restic-rest under `--private-repos` accepts usernames that contain `/` and matches them against multi-segment paths — e.g. user `<project>/<host>` mapping to `/<project>/<host>/*`, user `<project>/.restic-base-repo` mapping to that exact path. If restic-rest only treats usernames as a single top-level directory component, the scheme has to flatten (usernames like `<project>-<host>`, paths like `/<project>-<host>`), or drop to project-level isolation with a shared credential per project. Verify against the current restic-rest version before building anything else around the current layout.
- **Server-side hardlink dedup across near repos**. Does restic write byte-identical files to disk when two repos in the same project back up identical inputs? Empirical test: init two repos with `--copy-chunker-params`, back up the same directory into each, diff the resulting files under `/srv/restic`. If identical files are frequent, ship an opt-in `server_hardlink_dedup: true` flag on the server role that schedules a `hardlink(1)` pass. If not, drop the idea. Same investigation should confirm the "sharing disk blocks does not share access" expectation before any flag flips on.

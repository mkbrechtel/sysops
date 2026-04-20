---
status: draft
---

# Reverse proxy

## Goal

A socket-based reverse proxy role that bridges containerized and
system-level services without exposing application ports on the network.
Provides TLS termination and single ingress.

## Scope

- One RPX (Traefik, Caddy, or nginx — TBD) as the single HTTPS entrypoint.
- Backends reached via **unix sockets** where possible (systemd socket
  units, container-published sockets), not TCP ports on localhost.
- ACME / Let's Encrypt for TLS certificates.
- Per-project virtualhost definitions added by other roles (dependency
  inversion: apps register themselves with the RPX).
- HTTP → HTTPS redirect by default.

## Design notes

- A "register a vhost" interface: roles drop config fragments into a
  directory the RPX watches, or add to an inventory variable consumed by
  the RPX role.
- Socket location convention: `/run/<app>/rpx.sock` (or similar).
- Containers: use podman with `--network=none` + a socket bind-mount; RPX
  reads the socket directly. No bridge network, no exposed ports.
- Auth middleware (basic auth, forward-auth) as optional per-vhost.

## Open questions

- Which RPX — Traefik, Caddy, nginx? Traefik has the richest provider
  model; Caddy has the simplest ACME; nginx is ubiquitous but manual.
- Socket-only backends means we need every app container to publish on a
  socket. Is that a hard rule, or do we allow loopback TCP as a fallback?
- Where do vhost definitions live — per-role fragments in `/etc/rpx/`,
  or a central inventory variable aggregated by the RPX role?
- Certificate storage — integrate with the stashed `certificates` role, or
  let the RPX handle ACME itself?
- Wildcard certs? DNS-01 challenges require DNS API credentials; see
  `dns.feature.md` overlap.

---
status: draft
---

# Reverse proxy — Traefik backend

## Goal

A Traefik-based reverse proxy role that implements the web-service socket pattern (see `reverse-proxy.feature.md`). Picks up app sockets from `/run/web-services/<service>/http.sock`, terminates TLS, handles ACME, and serves as the single HTTPS ingress. Chosen per host for dynamic / container-heavy workloads where Traefik's provider model shines.

## Scope

- Install and configure Traefik as a systemd service.
- Single HTTPS entrypoint; HTTP → HTTPS redirect by default.
- **Service discovery via the file provider** pointed at the registration-fragment directory (translated from the pattern's neutral fragments by this role).
- **Socket discovery**: a second file-provider feed that maps each `/run/web-services/<service>/http.sock` to a Traefik service, joined to the vhost via the fragment.
- **ACME** via Traefik's built-in resolver (HTTP-01 by default; DNS-01 when the host declares DNS API credentials via the secrets role).
- Middleware support: basic auth, forward-auth, rate-limiting — driven by the neutral fragment schema.

## Design notes

### Why Traefik here

Its provider model is a natural fit for "watch a directory and react to changes", which is exactly what the fragment-based registration wants. ACME is built in. Dashboards and dynamic reconfiguration are part of the package, not bolt-ons.

### File provider, not socket activation

Traefik can discover services many ways; we use the file provider because the fragment directory is already the canonical source of truth. A docker/podman provider would tie the role to one container runtime; the file provider works equally well for socket-only apps and container apps because the discovery happens via the filesystem, not via a container API.

## Open questions

- **Dashboard exposure.** Internal-only (unix socket, accessed via ssh tunnel), or a protected vhost with forward-auth? Leaning internal-only.
- **Static vs. dynamic config split.** Traefik has a hard split; we need to pin down which knobs live in static (entrypoints, providers, ACME) and which are dynamic (everything fragment-driven).
- **Upgrade cadence.** Traefik's v2→v3 transition was painful. Pin a major, upgrade deliberately, or track latest?

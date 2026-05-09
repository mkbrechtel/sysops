---
status: draft
---

# Reverse proxy — Traefik backend

## Goal

A Traefik-based reverse proxy role that serves application sockets following the web-service socket pattern (see `reverse-proxy.pattern.md`). Sits on the outside of `/run/https/<service>/http.sock`, terminates TLS, handles ACME. Chosen per host for dynamic / container-heavy workloads where Traefik's provider model shines.

Directionality (matching the pattern ticket): **left = outside, right = inside**. Traefik is to the right of the network and to the left of the app socket.

## Scope

- Install and configure Traefik as a systemd service.
- Single HTTPS entrypoint; HTTP → HTTPS redirect by default.
- **Traefik configuration owned by the deployer** — vhost-to-socket mappings, middlewares, auth, rate limits all live in the Traefik role's inventory, not handed in by app roles.
- **ACME** via Traefik's built-in resolver (HTTP-01 by default; DNS-01 when the host declares DNS API credentials via the secrets role).

## Design notes

### Why Traefik

Provider model, built-in ACME, native dashboard. Sweet spot is dynamic service discovery, but for our pattern Traefik is just one possible RPX with a unix-socket backend; the dynamic-discovery story is incidental here.

## Open questions

- **Dashboard exposure.** Internal-only (unix socket, accessed via ssh tunnel), or a protected vhost? Leaning internal-only.
- **Static vs. dynamic config split.** Traefik has a hard split; pin down which knobs live where.
- **Upgrade cadence.** Traefik's v2→v3 transition was painful. Pin a major, upgrade deliberately, or track latest?

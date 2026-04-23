---
status: draft
---

# Reverse proxy — nginx backend

## Goal

An nginx-based reverse proxy role that implements the web-service socket pattern (see `reverse-proxy.feature.md`). Picks up app sockets from `/run/web-services/<service>/http.sock`, terminates TLS, handles ACME via a companion, and serves as the single HTTPS ingress. Chosen per host for predictable, static vhost layouts where nginx's tuning surface and operational familiarity win.

## Scope

- Install and configure nginx as a systemd service.
- Single HTTPS entrypoint; HTTP → HTTPS redirect by default.
- **Vhost generation**: translate the pattern's neutral fragment directory into per-vhost `server {}` blocks under `/etc/nginx/conf.d/` (or a similar managed directory), `proxy_pass` pointing at `unix:/run/web-services/<service>/http.sock`.
- **ACME** via a companion (certbot or lego) with nginx reload on renewal. HTTP-01 by default; DNS-01 when the host declares DNS API credentials via the secrets role.
- Middleware support: basic auth (`auth_basic`), forward-auth (via `auth_request`), rate limiting (`limit_req`) — driven by the neutral fragment schema.

## Design notes

### Why nginx here

Static vhost layouts, decades of tuning know-how, and operationally boring in a good way. Anyone who has run web services has debugged nginx at 3am and can read its logs without a manual. For hosts that are not container-heavy and where "reconfigure live" is not a requirement, the simpler mental model wins.

### Fragment → vhost translation

The role owns a small template that turns each neutral fragment into an nginx `server` block. Because nginx reloads are cheap and well-understood, we regenerate all vhosts and `nginx -t && systemctl reload nginx` on any fragment change — no live-watch provider needed.

### ACME companion

nginx itself does not do ACME. Certbot with the nginx plugin (or lego) keeps certs under `/etc/letsencrypt/` and triggers an nginx reload on renewal. Keep the companion choice behind a variable; default to certbot for ubiquity.

## Open questions

- **Dynamic modules.** We'll want at least `stream` for any raw-TCP side-channel and possibly `http_auth_request_module` for forward-auth. Does the Debian-packaged nginx ship what we need, or do we need `nginx-extras` / building from source?
- **Reload blast radius.** A bad fragment would break *all* vhosts on reload. `nginx -t` before reload is mandatory; do we also want per-fragment validation in the role before the fragment lands on disk?
- **Log format.** Stick with nginx defaults or define a structured (JSON) format compatible with the alerta / log-shipping story?

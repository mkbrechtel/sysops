---
status: draft
---

# Reverse proxy — nginx backend

## Goal

An nginx-based reverse proxy role that serves application sockets following the web-service socket pattern (see `reverse-proxy.pattern.md`). Sits on the outside of `/run/web-services/<service>/http.sock`, terminates TLS, handles ACME via a companion. Chosen per host for predictable, static vhost layouts where nginx's tuning surface and operational familiarity win.

Directionality (matching the pattern ticket): **left = outside, right = inside**. nginx is to the right of the network and to the left of the app socket.

## Scope

- Install and configure nginx as a systemd service.
- Single HTTPS entrypoint; HTTP → HTTPS redirect by default.
- **nginx configuration owned by the deployer** — vhost-to-socket mappings (`proxy_pass unix:/run/web-services/<service>/http.sock`), `auth_basic` / `auth_request`, `limit_req`, headers all live in the nginx role's inventory, not handed in by app roles.
- **ACME** via a companion (certbot or lego) with nginx reload on renewal. HTTP-01 by default; DNS-01 when the host declares DNS API credentials via the secrets role.

## Design notes

### Why nginx

Static vhost layouts, decades of tuning know-how, operationally boring in a good way. For hosts that are not container-heavy and where "reconfigure live" is not a requirement, the simpler mental model wins.

### ACME companion

nginx itself does not do ACME. Certbot with the nginx plugin (or lego) keeps certs under `/etc/letsencrypt/` and triggers an nginx reload on renewal. Default to certbot for ubiquity; companion choice behind a variable.

## Open questions

- **Dynamic modules.** `http_auth_request_module` for forward-auth, possibly `stream` for raw-TCP side-channels. Debian-packaged nginx vs. `nginx-extras` vs. building from source — TBD.
- **Reload blast radius.** A bad config breaks *all* vhosts on reload. `nginx -t` before reload is mandatory; do we want template-time validation in the role too?
- **Log format.** nginx defaults vs. structured (JSON) for the alerta / log-shipping story.

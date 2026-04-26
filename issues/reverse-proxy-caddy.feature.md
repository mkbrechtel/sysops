---
status: draft
---

# Reverse proxy — Caddy backend (default)

## Goal

A Caddy-based reverse proxy role that serves application sockets following the [web-service socket pattern](reverse-proxy.pattern.md). Terminates HTTPS/TLS, handles ACME. Connects to web-service sockets at `/run/web-services/<service>/http.sock`. Default reverse-proxy implementation in the collection.

Directionality (matching the pattern ticket): **left = outside, right = inside**. Caddy is to the right of the network and to the left of the app socket.

## Scope

- Install Caddy as a systemd service.
- Single HTTPS entrypoint per Caddy instance; HTTP → HTTPS redirect by default. Multiple Caddy instances per host are *possible* — a service can run its own private Caddy on the outside of its own socket — but **the scope of this role is the host-wide central Caddy installation only**. Per-service Caddy instances are deployed by the consuming service's own role using this role as a building block; that wiring is not in scope here.
- **Caddyfile owned by the deployer.** This role provides the Caddy installation, unit, and reload mechanism; the actual site configuration (which vhosts proxy to which sockets, what auth, what headers) is inventory on the RPX side, not handed in by app roles.
- **ACME** via Caddy's built-in resolver. HTTP-01 by default; DNS-01 when the host declares DNS API credentials via the secrets role (one of Caddy's DNS provider plugins, picked per provider).
- **Hot reload** via `caddy reload --config /etc/caddy/Caddyfile` on Caddyfile change. No restart, no dropped connections.
- **oauth2-proxy chaining** documented as the canonical "gate a service with project login" pattern. The service's own role places `oauth2-proxy` on the outside of the app socket; Caddy is on the outside of oauth2-proxy. Caddy role doesn't own this — but provides example Caddyfile snippets.

## Design notes

### Why Caddy as default

For single-host, small-team setups, Caddy gets the common case right with the least configuration:

- ACME built in. HTTP-01 just works; DNS-01 is a plugin away.
- Live reload is first-class.
- The Caddyfile reads like English. A reader unfamiliar with the role can debug a vhost by squinting at `/etc/caddy/Caddyfile`.
- Unix-socket reverse-proxy is one directive (`reverse_proxy unix//path/to/socket`), not a workaround.
- Sane defaults for HTTP/2, HTTP/3, security headers, log format. Less yak-shaving than nginx; less moving-target than Traefik's v2/v3 churn.

### Per-service vs. host-wide Caddy

The pattern explicitly does not mandate a single host-wide RPX. Both shapes are supported:

- **Host-wide instance**: one Caddy on the host, one Caddyfile, several vhosts each `reverse_proxy`-ing to a service socket.
- **Per-service instance**: a service role can spin up its own Caddy instance (this role parameterized by instance name) on the outside of its own socket — useful when the service wants its own auth chain or its own ACME identity. The service's role owns the deployment; this role provides the building block.

### oauth2-proxy chaining

Reading outside → inside:

```
client → network → Caddy (TLS, vhost) → oauth2-proxy (auth) → app socket
```

Each hop on the right of the network is a unix socket. Caddy's `reverse_proxy` points at oauth2-proxy's socket; oauth2-proxy in turn proxies to the app socket. The chain is a service-deployment choice; the Caddy role just provides the outer Caddy.

### Reload, not restart

`caddy reload` is graceful: existing connections finish, new connections use the new config, no listening-socket interruption. Restart is reserved for binary upgrades.

## Open questions

- **Caddy distribution**: Debian's packaged Caddy lags upstream and ships without DNS provider plugins. Options: (a) upstream `.deb` from cloudsmith, kept current via the standard `apt` flow; (b) build with `xcaddy` to bake in chosen DNS plugins; (c) static binary from GitHub releases with a custom unit. Lean toward (a) for the no-DNS-01 case, (b) when DNS-01 is needed.
- **Caddyfile vs. JSON config**: Caddyfile is human-friendly; JSON is what the API speaks. We're file-driven; Caddyfile reads better. Confirm Caddyfile.
- **Admin API exposure**: Caddy's admin API on `localhost:2019` by default lets anything local reload config or read state. Disable, bind to a unix socket with restricted perms, or leave it at `localhost`? Lean toward unix socket only.
- **Logging**: Default JSON access log (good for log shipping), or Combined-Log-style for human grep? Default JSON.

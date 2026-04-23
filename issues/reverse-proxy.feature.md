---
status: draft
---

# Reverse proxy — web-service socket pattern

## Goal

Define the collection's convention for how applications expose themselves to a reverse proxy: a unix-socket-based contract that decouples app roles from any specific RPX implementation. This ticket defines the *pattern*. The two implementations live in separate tickets:

- `reverse-proxy-traefik.feature.md` — Traefik backend.
- `reverse-proxy-nginx.feature.md` — nginx backend.

An application role only needs to follow this pattern to be servable by either backend, without knowing which is deployed.

## Scope

### The socket convention

Every application that wants to be served via the reverse proxy puts an HTTP socket at:

```
/run/web-services/<service>/http.sock
```

The reverse proxy — whichever implementation — discovers the socket by this path convention and proxies HTTP traffic to it. No app knows or cares whether Traefik or nginx is on the other end; no RPX config needs to know the app's port or internals.

### What the pattern specifies

- **Path**: `/run/web-services/<service>/http.sock`, where `<service>` is the service name (matches the project/service terminology ticket).
- **Protocol on the socket**: plain HTTP (not HTTPS — TLS is the RPX's job).
- **Ownership / perms**: the per-service directory is owned such that the app writes the socket and the RPX reads it. Exact group/ACL setup to be pinned down (open question).
- **Vhost registration**: apps drop a fragment describing their vhost (name, optional auth, etc.) into a watched directory. The fragment schema is shared across backends so an app role is RPX-agnostic; each backend role translates the fragment into its native config.

### What the pattern does not specify

- TLS termination, ACME, redirects — those are per-backend concerns.
- How the backend is configured internally.
- What happens on the public side of the reverse proxy.

### Containers

Podman with `--network=none` plus a bind-mount of `/run/web-services/<service>/` into the container. The container writes its HTTP socket into that directory. No bridge network, no exposed ports, no localhost TCP. This is part of the pattern because the whole point is "no app-side TCP."

## Design notes

### Why a pattern ticket separate from implementations

Two backends, one contract. Putting the contract in its own ticket makes it explicit that the contract is load-bearing and the backend choice is not. If we ever add a third backend (Caddy, HAProxy, …) it slots in under the same pattern without churning every app role. Splitting also keeps the implementation tickets focused on "how does Traefik/nginx find and proxy these sockets" rather than relitigating the contract.

### Why plain HTTP on the socket

TLS on the app side would duplicate work and move cert management into every app. TLS is a property of public ingress, not of the app; terminating at the RPX keeps it there.

### Why a shared fragment schema for vhost registration

So an app role does not have to know which backend is deployed. The app role writes a neutral fragment ("vhost foo.example.com, no auth, rate-limit N/s"); the deployed backend's role consumes fragments and emits its native config. Cost: one more translation layer. Benefit: one app role works with any backend.

## Open questions

- **Fragment schema detail.** A small neutral YAML schema covering vhost name, per-vhost auth (none / basic / forward), rate limiting, headers, redirects — what's the minimum that covers real cases without turning into "reimplement both backends' config languages"?
- **Socket permissions.** Group-based (shared `web-services` group the RPX is in, sockets g+rw), ACL-based, or per-service user+RPX peer check? Group is simplest; ACLs are more explicit.
- **Multiple sockets per service.** Does a service ever need more than one — e.g. a separate admin socket? If yes, `/run/web-services/<service>/<name>.sock`, and the fragment names which socket to proxy where. If no, keep it single.
- **Non-HTTP protocols.** Websockets come free over the same HTTP socket. But gRPC, raw TCP, or HTTP/3 over QUIC don't. In scope for this pattern or punt to a separate ticket?

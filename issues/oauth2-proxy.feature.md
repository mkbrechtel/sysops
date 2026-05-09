---
status: draft
---

# oauth2-proxy

## Goal

A role that deploys oauth2-proxy in front of services that need OIDC-based access control, integrating with the collection's reverse-proxy pattern (web-service socket on the inside, Caddy/Traefik/nginx on the outside) and the secrets role for OIDC client credentials.

## Scope

Two integration shapes that oauth2-proxy supports natively, both worth shipping:

### 1. In-line chaining (proxy mode)

oauth2-proxy sits between the reverse proxy and the application: the reverse proxy proxies to oauth2-proxy's socket; oauth2-proxy enforces auth and proxies to the app socket. Reading outside → inside:

```
client → network → reverse proxy → oauth2-proxy → app socket
```

Each hop on the right of the network is a unix socket. Best for services that just need "log in or you don't get past this layer at all."

### 2. forward_auth check (out-of-band auth)

The reverse proxy keeps proxying directly to the app socket but issues a sub-request to oauth2-proxy on every (or selected) requests via `forward_auth` (Caddy) / `auth_request` (nginx) / Traefik's forward-auth middleware. oauth2-proxy returns 200 + identity headers (`X-Auth-Request-Email`, etc.) on success, 401/redirect on failure. The reverse proxy enforces the verdict and forwards augmented headers to the app.

```
client → network → reverse proxy ──auth-check──> oauth2-proxy
                          │
                          └────────> app socket  (with X-Auth-* headers)
```

Best for services that take signed-in identity into account but don't need oauth2-proxy strictly in the data path — and for services that want sub-request granularity (auth this path, skip auth on `/health`, etc.).

### Other concerns

- One oauth2-proxy instance per project (typically), pinned to that project's OIDC client.
- OIDC client ID + secret from the secrets role.
- Cookie/session secret from the secrets role.
- Provider configuration (Keycloak, Authentik, Google, etc.) parameterized.
- Allowed-email / allowed-group rules per service.

## Design notes

- The two shapes are not exclusive — a host can run several oauth2-proxy instances, some in-line, some forward-auth, depending on what each service needs.
- Forward-auth lets the reverse proxy stay closer to the app socket (one less hop in the data path for static assets, etc.) at the cost of an auth sub-request per gated request. In-line chaining is simpler to reason about but inserts oauth2-proxy into every byte.
- oauth2-proxy publishes its own unix socket following the [web-service socket pattern](reverse-proxy.pattern.md), so the reverse proxy talks to it the same way it talks to any other service.

## Open questions

- **Scope per project, scope per service, or both?** A single project-wide oauth2-proxy gating multiple services is cheaper to operate; per-service instances allow narrower OIDC client scopes. Default likely "one per project" with per-service as an override.
- **Session cookie domain.** Sharing a single oauth2-proxy across `*.<zone>` works smoothly when the cookie domain is `.<zone>`. Picking that requires the proxy and all gated services to live under the same parent zone.
- **Group / role mapping.** Does the role expose just allowed-email / allowed-group, or also more elaborate rules (per-path allow lists, header-based RBAC)? Lean minimal — push elaborate rules into the application.
- **Logout flow.** OIDC logout is famously inconsistent across providers. Worth specifying which providers we test against.

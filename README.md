# golang-proxy

A lightweight, single-purpose Go reverse proxy that forwards **all** traffic from
`/` to a configurable backend. It is built to be Docker/Kubernetes-ready and to
sit in front of a service you want to protect or expose.

Built on [`httputil.NewSingleHostReverseProxy`](https://pkg.go.dev/net/http/httputil#NewSingleHostReverseProxy),
with optional secret-based access control and per-IP lockout to deter brute-force
guessing of the secret.

## Features

- **Single upstream** — every request is proxied to one target URL.
- **Header preservation** — forwards cookies, session tokens, `Origin`, and the
  original `Host` header (honoring `X-Forwarded-Host` when set).
- **Real client IP** — sets `X-Real-Ip` when it is not already present, so the
  backend knows the true client address even when you are behind another reverse
  proxy (Traefik, nginx, HAProxy, …).
- **Optional access control** — when `PROXY_SECRET` is set, every request must
  carry a matching `X-API-Secret` header or it is rejected with `403 Forbidden`.
- **Brute-force lockout** — repeated failed secret attempts lock an IP out for a
  growing cooldown period (see [Access control](#access-control)).
- **Self-signed backends** — TLS verification to the upstream is disabled by
  design so the proxy can talk to backends with self-signed certificates.

## How it works

1. On startup the proxy reads its configuration from environment variables and
   builds a reverse proxy pointed at `PROXY_TARGET_URL`.
2. For each incoming request it resolves the client IP (preferring the leftmost
   entry of `X-Forwarded-For`, falling back to the TCP peer address).
3. If the IP is currently locked out, the request is rejected immediately.
4. If `PROXY_SECRET` is configured and the `X-API-Secret` header does not match,
   the failure is recorded for that IP and the request is rejected with `403`.
5. Otherwise the proxy restores/forwards the relevant headers and hands the
   request to the upstream. A successful request clears any pending lockout for
   the IP.
6. If the backend cannot be reached, the client receives `502 Bad Gateway` with a
   `Backend unavailable` message.

### Access control

When `PROXY_SECRET` is set:

- A correct `X-API-Secret` header lets the request through and **clears** any
  lockout for the client IP.
- A missing or incorrect secret returns `403 Forbidden: invalid or missing
  secret` and records a failed attempt.
- Each failed attempt extends the lockout. An IP is locked out for
  `failed_attempts × 2 minutes` measured from the most recent failed attempt.
  While locked, requests return `403 Forbidden: On cooldown for next try
  because of invalid or missing secret`.

If `PROXY_SECRET` is not set, no secret checking is performed and the proxy is
open.

## Configuration

All configuration comes from environment variables. Nothing is hardcoded.

| Variable             | Required | Default | Description                                                                    |
| -------------------- | -------- | ------- | ------------------------------------------------------------------------------ |
| `PROXY_TARGET_URL`   | Yes      | —       | Upstream URL to proxy to, e.g. `http://10.0.0.5:8080`. Must be a valid URL.   |
| `SERVER_PORT`        | No       | `3000`  | Port the proxy listens on.                                                      |
| `PROXY_SECRET`       | No       | —       | If set, requests must send a matching `X-API-Secret` header.                    |

If `PROXY_TARGET_URL` is missing or cannot be parsed, the proxy logs an error
and exits.

## Running

### Locally

You must set a target URL:

```bash
PROXY_TARGET_URL=http://127.0.0.1:8080 go run .
```

With a secret and a custom port:

```bash
PROXY_TARGET_URL=http://127.0.0.1:8080 SERVER_PORT=8080 PROXY_SECRET=s3cr3t go run .
```

### Build a binary

```bash
go build -o go-proxy .
```

### Docker

```bash
docker build -t go-proxy .
docker run --rm -p 3000:3000 \
  -e PROXY_TARGET_URL=http://127.0.0.1:8080 \
  go-proxy
```

The image is multi-stage and ships a small static binary on Alpine (entrypoint
`./server`).

### Docker Compose

```bash
docker compose up
```

Adjust `PROXY_TARGET_URL` (and optionally add `PROXY_SECRET` / `SERVER_PORT`) in
`docker-compose.yml` before running.

### Kubernetes

Inject the configuration via a `ConfigMap`/`Secret` on the Deployment:

```yaml
env:
  - name: PROXY_TARGET_URL
    value: http://my-backend:8080
  - name: SERVER_PORT
    value: "3000"
  - name: PROXY_SECRET
    valueFrom:
      secretKeyRef:
        name: proxy-secret
        key: value
```

## Verifying your changes

There is no test suite or CI. To sanity-check the code:

```bash
go vet ./...
gofmt -l .
```

The module is `go-proxy` on Go 1.23 (see `go.mod`).

## Notes

- `static/index.html` is a manual dev-only test page. It is listed in
  `.dockerignore` and is **not** served by the proxy.
- TLS verification to the backend is intentionally disabled
  (`InsecureSkipVerify: true`) to support self-signed backend certificates.

# AGENTS.md

Project-specific guidance for AI agents working in this repository.

## Project overview

Single-purpose Go reverse proxy (Docker/Kubernetes ready). `main.go` is the
only code file and the sole entrypoint.

- Proxies all traffic from `/` to a configurable backend via
  `httputil.NewSingleHostReverseProxy`.
- Preserves forwarding headers (cookies, session tokens, Origin, Host,
  `X-Forwarded-Host`) and sets `X-Real-Ip` when absent.
- No routing logic, no RegexpHandler, no static file server—proxying only.

## Runtime configuration (env vars)

All config comes from environment variables. `main.go` reads exactly these:

- `PROXY_TARGET_URL` (REQUIRED): upstream URL, e.g. `http://10.0.0.5:8080`.
  Missing or unparseable value → `log.Fatal` and exit.
- `SERVER_PORT` (optional): listen port, default `3000`.
- `PROXY_SECRET` (optional): if set, every request must carry a matching
  `X-API-Secret` header or it is rejected with `403 Forbidden`.

Do not hardcode target URLs or secrets—always source them from env.

## Behavioral gotchas

- TLS verification to the backend is intentionally disabled
  (`InsecureSkipVerify: true`) to support self-signed backend certificates.
  This is deliberate, not a bug—do not "fix" it without being asked.
- `static/index.html` is a manual dev-only test page (it is in `.dockerignore`
  and is NOT served by the proxy). Do not treat it as part of the app.

## Essential commands

- Run locally (must set target):
  - `PROXY_TARGET_URL=http://127.0.0.1:8080 go run .`
- Build a binary:
  - `go build -o go-proxy .`
- Vet:
  - `go vet ./...`
- Format check / edit:
  - `gofmt -l .`
  - `gofmt -w .`

No test suite, linter config, or CI exists. Module is `go-proxy` on Go 1.23
(`go.mod`); verify with `go vet ./...` and `gofmt -l .`.

## Docker / K8s notes

- Build image: `docker build -t go-proxy .` (multi-stage; final image is
  alpine, entrypoint `./server`).
- Run with compose: `docker compose up` (adjust `PROXY_TARGET_URL` in
  `docker-compose.yml`).
- For Kubernetes, inject `PROXY_TARGET_URL`, `SERVER_PORT`, and
  `PROXY_SECRET` via ConfigMap/Secret.

# AGENTS.md

Project-specific guidance for AI agents working in this repository.

## Project overview

Single-purpose Go reverse proxy designed for commercial use (Docker/Kubernetes ready):
- Proxies all traffic from `/` to a configurable backend via `httputil.ReverseProxy`.
- Preserves browser headers: cookies, session tokens, Origin, Host, etc.
- No routing logic, no RegexpHandler, no static file server—only proxying.

Key files:
- `main.go`: single entrypoint; reads environment and starts the proxy.
- `Dockerfile`, `docker-compose.yml`: deployment helpers for Docker/K8s-style setups.

## Required configuration

All runtime config is via environment variables (required in production/Docker):

- `PROXY_TARGET_URL` (REQUIRED): upstream backend URL, e.g. `http://10.0.0.5:8080`.
  - If missing or invalid, the server logs an error and exits.
- `SERVER_PORT` (optional): port to listen on; default is `3000`.

Agents must not hardcode target URLs; always use these env vars (or explicit config) instead.

## Essential commands

- Run locally (must set PROXY_TARGET_URL):
  - `PROXY_TARGET_URL=http://127.0.0.1:8080 go run .`
- Build a binary:
  - `go build -o go-proxy .`
- Vet/lint:
  - `go vet ./...`
- Format check/edit:
  - `gofmt -l .`
  - `gofmt -w .`

No test suite or CI currently exists.

## Docker / K8s notes

- Build image:
  - `docker build -t go-proxy .`
- Run with docker compose (adjust PROXY_TARGET_URL):
  - `docker compose up`
- For Kubernetes, inject `PROXY_TARGET_URL` and `SERVER_PORT` via ConfigMap/Secret.
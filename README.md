# Sky Migration Registry

**Status: engineering beta.** This repository is a bounded migration registration and validation service written in Go. It records migration metadata and content digests; it does **not** execute SQL against a database.

## API

- `GET /health` — liveness.
- `GET /ready` — readiness.
- `GET /metrics` — local request/rejection/registration counters.
- `POST /api/v1/migrations` — register `{version,name,sql}` after validation.
- `GET /api/v1/migrations` — return ordered metadata records without exposing SQL bodies.

A repeated version with identical name/content is idempotent. Reusing a version with different content returns `409 Conflict`. SQL payloads are bounded to 256 KiB, names are restricted to safe characters, unknown JSON fields are rejected, and stored list responses expose SHA-256/size metadata rather than SQL text.

## Verification

```bash
gofmt -w .
go vet ./...
go test -race -count=1 ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
CGO_ENABLED=0 go build ./...
docker build -t sky-migrations .
```

CI enforces those gates plus non-root container configuration.

## Example

```bash
curl -sS -X POST http://localhost:8080/api/v1/migrations \
  -H 'content-type: application/json' \
  -d '{"version":1,"name":"create_users","sql":"CREATE TABLE users(id BIGINT PRIMARY KEY);"}'
```

## Product boundary

This is deliberately a migration **registry**, not a database migration executor. It currently uses process memory only. It does not claim durable history, transaction execution, rollback, schema introspection, PostgreSQL/MySQL/SQLite connectivity, distributed coordination, authentication/RBAC, encryption at rest, HA, or production deployment. Those require separate implementation and evidence.

For SKYCOIN4444, the stable HTTP contract can be used by deployment tooling to validate and fingerprint migration plans before a database-specific executor is introduced.

See `SECURITY.md` for security assumptions.

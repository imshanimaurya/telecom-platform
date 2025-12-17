# telecom-platform

Go monorepo-style project skeleton for a telecom platform (routing, pricing, wallet, campaigns, telephony providers).

## Quick start

- Run the API (once Go is installed):
  - `go run ./cmd/api`
- Health check:
  - `GET http://localhost:8080/healthz`

## Structure

- `cmd/api` – HTTP API entrypoint
- `internal/*` – application modules (not exported)
- `pkg/*` – reusable packages intended for external reuse
- `migrations` – database migrations

## Next steps

- Add config loader (`internal/config`) reading env + files
- Add router (chi/gin), middleware (logging, auth, RBAC)
- Add persistence layer + migrations tool

## Wallet invariants

This repo uses a prepaid wallet with an **immutable append-only ledger**.

- All wallet operations are **workspace-scoped** (`workspace_id` required).
- **No balance updates without a ledger entry**.
- Balance is stored in a **projection table** (`wallet_balances`) that is updated **in the same DB transaction** as the ledger insert.
- All money operations must run inside a DB transaction.

### Required DB constraints (recommended)

- `wallet_ledger` must be append-only (enforced by application; you can also add DB permissions/triggers).
- Idempotency: add a unique constraint to support safe retries:
  - `UNIQUE (workspace_id, wallet_id, idempotency_key)`

# Ledger

A write-ahead, double-entry ledger. Second pass at ideas from [jobWorker](../jobWorker) and [kvStore](../kvStore) — this time the journal is the one source of truth, and everything else (balances, the account index) is just a replay of it.

Reach Me: sundayprincedev@gmail.com

## What it is

- Integer money only — minor units in `int64`, each currency carries its own decimal places, no floats until the display edge → [`money/currency.go`](internal/money/currency.go)
- Account kinds ported from [Bank](../Bank) and trimmed down: customer current/savings, merchant, house, suspense → [`accounts/`](internal/accounts/)
- No direct balance mutation on an account. `Deposit`/`Withdraw`/`Transfer` methods were deleted on purpose — the ledger is the only thing allowed to move money, by replaying entries → [`accounts/base.go`](internal/accounts/base.go)
- Double-entry `Entry`/`Line`, balanced per currency (not globally), with overflow-checked sums → [`ledger/entry.go`](internal/ledger/entry.go)
- Append-only journal: CRC-checked records, torn-tail truncation on recovery, a poisoned writer on any write/sync failure → [`journal/writer.go`](internal/journal/writer.go) · [`journal/recovery.go`](internal/journal/recovery.go)
- `ledger.Journal` interface (`Append`, `Recover`, `Close`) so the file-backed journal can be swapped for Postgres later without touching the ledger → [`ledger/journal.go`](internal/ledger/journal.go)
- Centralized sentinel errors, one file per owning package, so nothing grows its own ad hoc `errors.go` → [`errors/`](internal/errors/)
- KYC tiers (1–3) with per-tier transfer limits, seeded onto customer and merchant accounts, ahead of a `can_transfer` check → [`constants/kyc.go`](internal/constants/kyc.go)
- Seed data: accounts + operations, not entries. Operations are requests that flow through the system and get turned into entries — never pre-baked → [`data/seed.json`](internal/data/seed.json)

## Run

```bash
go build ./...
go test ./...
```

No CLI yet — `cmd/ledger/main.go` is still a stub. `ledger.Open`/`Post`, the thing that actually replays the journal and applies entries to balances, hasn't been written.

## Design

[`ledger-design-v0.md`](../ledger-design-v0.md) is the running design log — not a spec written once, but a list of decisions as they get made, with the reasoning behind each.

## What's Next

- [ ] `ledger.Open`/`Post`: ID map, replay, trial balance
- [ ] Job worker wired to `Post` — everything here is a job
- [ ] `can_transfer`: KYC tier limits actually enforced
- [ ] Evidence log for failed/rejected attempts
- [ ] Postgres implementation of `ledger.Journal`

## Status

Early — most packages exist in isolation, nothing end-to-end yet

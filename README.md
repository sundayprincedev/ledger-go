# Ledger

A write-ahead, double-entry ledger. Second pass at ideas from [jobWorker](../jobWorker) and [kvStore](../kvStore) — this time the journal is the one source of truth, and everything else (balances, the account index) is just a replay of it.

Reach Me: sundayprincedev@gmail.com

## What it is

- Integer money only — minor units in `int64`, each currency carries its own decimal places, no floats until the display edge → [`money/currency.go`](internal/money/currency.go)
- Account kinds ported from [Bank](../Bank) and trimmed down: customer current/savings, merchant, house, suspense → [`accounts/`](internal/accounts/)
- No direct balance mutation on an account. `Deposit`/`Withdraw`/`Transfer` methods were deleted on purpose — the ledger is the only thing allowed to move money, by replaying entries → [`accounts/base.go`](internal/accounts/base.go)
- Double-entry `Entry`/`Line`, balanced per currency (not globally), with overflow-checked sums → [`ledger/entry.go`](internal/ledger/entry.go)
- Startup read pipeline: recover and clean, decode each record, register every written ID as completed; a duplicate ID warns, a bad record stops startup → [`jobs/read_pipeline.go`](internal/jobs/read_pipeline.go)
- Append-only journal: CRC-checked records, torn-tail truncation on recovery, a poisoned writer on any write/sync failure → [`journal/writer.go`](internal/journal/writer.go) · [`journal/recovery.go`](internal/journal/recovery.go)
- `ledger.Journal` interface (`Append`, `Recover`, `Close`) so the file-backed journal can be swapped for Postgres later without touching the ledger → [`ledger/journal.go`](internal/ledger/journal.go)
- Centralized sentinel errors, one file per owning package, so nothing grows its own ad hoc `errors.go` → [`errors/`](internal/errors/)
- KYC tiers (1–3) with per-tier transfer limits, seeded onto customer and merchant accounts and enforced by the `kyc` stage of the write pipeline → [`constants/kyc.go`](internal/constants/kyc.go)
- Seed data: accounts + operations, not entries. Operations are requests that flow through the system and get turned into entries — never pre-baked → [`data/seed.json`](internal/data/seed.json)

## Run

```bash
go build ./...
go test ./...
go run ./cmd/ledger -journal /tmp/ledger.log
```

`cmd/ledger` loads the seed, builds the stores, turns the seeded transfers into jobs, and runs them through the write pipeline with one write worker. Before the worker starts, it recovers the journal (truncating a torn tail), decodes every record and registers each entry ID as completed. Running again against the same journal therefore writes nothing twice: the already written transfers are rejected at `register`. SIGTERM and SIGINT stop new work and give what is already running 4 seconds to finish before it is cancelled; a second signal exits at once.

## Design

[`ledger-design-v0.md`](../ledger-design-v0.md) is the running design log — not a spec written once, but a list of decisions as they get made, with the reasoning behind each.

## What's Next

- [x] ID map filled by replay at startup
- [ ] Trial balance
- [x] Write worker runs the write pipeline end to end from `cmd/ledger`
- [x] KYC tier limits enforced at the `kyc` stage (per-currency limits still open)
- [x] In-memory balances applied after the append, before the job is marked completed
- [x] Ack on every success; a completed duplicate is re-acked as `already_completed`
- [x] Replay applies recovered entries to balances
- [ ] Registry pending state and the resolver pipeline
- [ ] Evidence log for failed/rejected attempts
- [ ] Postgres implementation of `ledger.Journal`

## Status

Early — `cmd/ledger` runs end to end for direct transfers: replay, KYC, line building, journal append, graceful shutdown. Bulk transfers, balances and the other operation types are not built

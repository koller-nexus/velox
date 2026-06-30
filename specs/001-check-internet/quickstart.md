# Quickstart & Validation: Check Internet & Nearest Provider

**Feature**: 001-check-internet | **Date**: 2026-06-30

Runnable scenarios that prove the feature works end to end. References
[contracts/](./contracts/) and [data-model.md](./data-model.md) instead of
duplicating detail.

## Prerequisites

- Go 1.26.4 (`go version`)
- Network access for integration/manual runs (unit tests need none)
- Dev tools for the security gate: `gosec`, `govulncheck`, `golangci-lint`

## Build

```bash
make build          # or: CGO_ENABLED=0 go build -o bin/velox ./cmd/velox
./bin/velox --version
```

## Validation Scenarios

### S1 — Full speed test (FR-001, US1)

```bash
./bin/velox --check-internet
```

Expected: online, with latency, jitter, download Mbps, upload Mbps, and the
selected server; exit code `0`. See [cli-interface.md](./contracts/cli-interface.md).

### S2 — Machine-readable output (FR-003, SC-009)

```bash
./bin/velox --check-internet --json | jq .
```

Expected: one JSON object validating against
[result.schema.json](./contracts/result.schema.json); throughput in labeled Mbps.

### S3 — Offline behavior (FR-001 AS#2)

```bash
# Disable network, then:
./bin/velox --check-internet; echo "exit=$?"
```

Expected: clear "offline" message on stderr; `exit=1`; throughput phases skipped.

### S4 — Location consent gate (FR-004, SC-002)

```bash
./bin/velox consent --reset
./bin/velox --check-internet        # interactive TTY
```

Expected: an approve/decline prompt appears before any location lookup, stating
the public IP is sent to a geolocation service.

### S5 — Consent remembered (FR-005, SC-003)

```bash
# After approving in S4:
./bin/velox --check-internet
./bin/velox consent --status
```

Expected: no second prompt; `--status` reports `granted`.

### S6 — Decline / non-interactive fallback (FR-007, SC-004)

```bash
./bin/velox consent --reset
./bin/velox --check-internet --json < /dev/null | jq '.distanceKm'
```

Expected: no prompt (no TTY → treated as declined), run completes against the
fallback server, `distanceKm` is `null`; no IP-geo lookup performed.

### S7 — Manual server override (FR-010)

```bash
./bin/velox --check-internet --server "wss://<host>/ndt/v7/download"
```

Expected: test runs against the given server; selection bypassed.

### S8 — Cancellation (FR-011)

```bash
./bin/velox --check-internet   # press Ctrl-C mid-run
```

Expected: prompt cancels cleanly, terminal restored, non-zero exit, no hung process.

### S9 — Security scan / CI gate (FR-012/013, Q5)

```bash
make security        # runs gosec + govulncheck, HIGH+ gating
echo "exit=$?"
```

Expected: findings listed with severity + location; exit non-zero iff a finding
is HIGH or CRITICAL; MEDIUM/LOW shown as warnings.

## Automated Test Gates (Constitution)

```bash
gofmt -l .                 # must print nothing
go vet ./...
golangci-lint run
go test -race ./...        # unit tests, network-free
go test -race -tags=integration ./test/integration/...   # opt-in, live M-Lab
```

All must pass before merge (Constitution: Technical Standards & Quality Gates).

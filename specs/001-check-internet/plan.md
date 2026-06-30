# Implementation Plan: Check Internet & Nearest Provider

**Branch**: `001-check-internet` | **Date**: 2026-06-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-check-internet/spec.md`

## Summary

`velox --check-internet` runs a full internet speed test — confirm connectivity,
then measure latency/jitter, download throughput, and upload throughput against
the nearest open test server. Servers are discovered from M-Lab's open registry;
measurement uses the ndt7 protocol over TLS (TCP/HTTP-upgraded, no elevated
privileges). The nearest server is ranked using consent-gated, IP-based
geolocation; without consent velox falls back to a default server. A maintainer
security command runs gosec + govulncheck, gating CI at HIGH+ severity. The
project ships open source with a maintained `CHANGELOG.md`.

## Technical Context

**Language/Version**: Go 1.26.4

**Primary Dependencies**:
- Standard library first: `net/http`, `context`, `flag`, `encoding/json`, `time`, `os`, `os/signal`.
- `github.com/m-lab/ndt7-client-go` (Apache-2.0) — canonical ndt7 measurement client (download/upload/latency). Justified: re-implementing ndt7 + WebSocket framing is large and error-prone; this is the reference implementation.
- M-Lab Locate API v2 accessed via stdlib `net/http` (no client library needed).
- IP geolocation via stdlib `net/http` against a configurable, no-API-key endpoint.
- Dev/CI only (not linked into the binary): `gosec`, `govulncheck`, `golangci-lint`.

**Storage**: Local consent + config file under `os.UserConfigDir()/velox/config.json` (JSON). No database.

**Testing**: Go `testing` — table-driven unit tests co-located (`*_test.go`), `net/http/httptest` for HTTP fakes, interface fakes for the ndt7 client and geo lookup, `go test -race`. Integration tests under `test/integration` behind `//go:build integration`.

**Target Platform**: Single static binary (CGO disabled) for Linux, macOS, Windows on amd64 + arm64.

**Project Type**: CLI tool (single project).

**Performance Goals**: Connectivity confirmed < 5s (SC-001); full test completes < 60s (SC-001b); negligible startup-to-first-output overhead.

**Constraints**: No elevated OS privileges (TCP/HTTP/WSS only, no ICMP/raw sockets); all network ops context-bounded and cancellable (Ctrl-C); race-free saturation; consent required before any IP-geo lookup; non-interactive sessions default consent to "declined".

**Scale/Scope**: Single-user CLI; ~6-8 internal packages; one user-facing command plus consent management and a maintainer security command.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Evaluated against Velox Constitution v1.0.0:

| Principle | Gate | Status |
|-----------|------|--------|
| I. Clean Code & Idiomatic Go | gofmt + go vet + golangci-lint clean; small functions; wrapped errors | PASS — enforced in CI gates and review |
| II. CLI-First Interface | args→stdout, diagnostics→stderr, `--json`, SIGINT cancel, meaningful exit codes, `--help`/`--version` offline | PASS — designed in CLI contract |
| III. Test-First (NON-NEGOTIABLE) | TDD; no real network in unit tests; interfaces + fakes/httptest; regression test per bug | PASS — measurement, locate, geo, consent all behind interfaces |
| IV. Measurement Accuracy & Reliability | defined units/windows; context timeouts; cancellable; `-race` clean; graceful degradation | PASS — explicit timeouts + fallback paths |
| V. Simplicity & Minimal Dependencies | stdlib first; deps justified + govulncheck clean; single static binary; no speculative abstraction | PASS — one runtime dep (ndt7 client) justified below |

**Initial gate: PASS.** No violations requiring Complexity Tracking. Dependency
justifications recorded in `research.md`; the single third-party runtime
dependency (`m-lab/ndt7-client-go`) is explicitly justified and is the canonical
implementation of the chosen open protocol.

## Project Structure

### Documentation (this feature)

```text
specs/001-check-internet/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── cli-interface.md       # Command/flag contract, exit codes, output schema
│   ├── result.schema.json     # JSON output schema for --json
│   ├── consent.schema.json    # Consent/config file schema
│   └── external-apis.md       # M-Lab Locate v2 + IP-geo request/response contract
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
cmd/
└── velox/
    └── main.go              # Entry point: wire deps, parse args, dispatch, set exit code

internal/
├── cli/                     # Flag parsing, command dispatch, output rendering (human + JSON)
├── speedtest/               # Orchestrates a run: connectivity → latency → download → upload
├── locate/                  # M-Lab Locate v2 client (server discovery) over net/http
├── ndt7/                    # ndt7 client wrapper (adapts m-lab/ndt7-client-go behind an interface)
├── geo/                     # IP-based geolocation lookup + haversine distance ranking
├── consent/                 # Consent store: load/save/reset; interactive prompt; non-TTY default
├── config/                  # Config dir/file resolution (os.UserConfigDir), JSON read/write
└── version/                 # Version/build metadata for --version

test/
└── integration/             # //go:build integration — end-to-end against live M-Lab (opt-in)

scripts/
├── security.sh              # Runs gosec + govulncheck with HIGH+ gating (maintainer/CI command)
└── build.sh                 # Cross-compile static binaries

.github/workflows/ci.yml     # gofmt -l, go vet, golangci-lint, go test -race, security.sh
CHANGELOG.md                 # Keep a Changelog format
LICENSE                      # Open-source license
Makefile                     # build, test, lint, security, vuln targets
go.mod / go.sum
```

**Structure Decision**: Single-project Go CLI using the idiomatic `cmd/` +
`internal/` layout. Unit tests are co-located with their packages
(`internal/<pkg>/<file>_test.go`) per Go convention; integration tests live in
`test/integration` behind a build tag so the default `go test ./...` stays
network-free (Principle III). The security command is a maintainer/CI script
(`scripts/security.sh` + Makefile target), not linked into the shipped binary,
keeping the runtime minimal (Principle V).

## Complexity Tracking

> No constitution violations. Section intentionally empty.

The single third-party runtime dependency (`m-lab/ndt7-client-go`) is justified
under Principle V in `research.md` and does not constitute a violation.

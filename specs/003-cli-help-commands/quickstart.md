# Quickstart: Help Commands & Additional Useful Commands

**Feature**: `003-cli-help-commands` | **Date**: 2026-07-01

Runnable checks that prove the feature works end to end. See
[contracts/cli-interface.md](./contracts/cli-interface.md) for the full command
contract and the `*.schema.json` files for `--json` shapes.

## Prerequisites

```bash
# From repo root
make build            # -> ./bin/velox
# or: go build -o bin/velox ./cmd/velox
```

Unit tests require no network. The `servers`/`ping` live checks below need
internet; the offline checks explicitly must NOT need it.

## Scenario 1 — Discoverable help (US1 / FR-001, FR-002, FR-007)

```bash
./bin/velox help
```
Expected: an overview listing every command (`help`, `version`, `servers`,
`ping`, `config`, `consent`) each with a one-line summary; exit code `0`.

```bash
./bin/velox help consent
./bin/velox consent --help
```
Expected: both print the *same* detailed usage for `consent` to stdout; exit
`0`; no consent prompt appears.

Verify overview completeness (FR-007): every command name printed by
`velox help` also resolves under `velox help <name>`.

## Scenario 2 — Offline guarantee (FR-008, SC-003)

```bash
# Simulate no network (example; use your platform's method)
./bin/velox help    >/dev/null && echo help-ok
./bin/velox version >/dev/null && echo version-ok
./bin/velox config  >/dev/null && echo config-ok
```
Expected: all three succeed (exit `0`) and return within ~1 s with no network
access and no consent prompt.

## Scenario 3 — Unknown command / flag (FR-004, SC-004)

```bash
./bin/velox frobnicate; echo "exit=$?"
./bin/velox servers --bogus; echo "exit=$?"
```
Expected: a clear error on **stderr** naming the problem and suggesting
`velox help`; `exit=2` for both.

## Scenario 4 — version subcommand (FR-010)

```bash
diff <(./bin/velox version) <(./bin/velox --version) && echo "identical"
```
Expected: identical output; exit `0`; no network.

## Scenario 5 — servers listing (FR-011)

```bash
./bin/velox servers
./bin/velox servers --json | jq .
```
Expected (human): up to ~5 nearest servers with machine/site, city+country, and
distance km, with the selected server marked. Expected (`--json`): a document
validating against `contracts/servers.schema.json` (`maxItems: 5`,
`selected: true` on exactly one entry when a selection is possible). Consent is
requested once if unset and interactive; when denied or non-interactive, output
falls back to proximity order with `distanceKm` null.

## Scenario 6 — ping / latency only (FR-013)

```bash
./bin/velox ping
./bin/velox ping --json | jq '{online, latencyMs, jitterMs}'
```
Expected: reports latency and jitter only (no download/upload numbers); returns
quickly; `--json` validates against `contracts/ping.schema.json`. Exit `0` when
online, `1` when offline.

## Scenario 7 — config inspection (FR-012)

```bash
./bin/velox config
./bin/velox config --json | jq '{configPath, consent}'
```
Expected: prints the config path/dir, consent decision (+ decidedAt), and
effective settings; read-only; no network. `--json` validates against
`contracts/config.schema.json`.

## Scenario 8 — backward compatibility (FR-005)

```bash
./bin/velox            # overview, exit 0
./bin/velox --help     # overview, exit 0
./bin/velox --version  # version line, exit 0
```
Expected: unchanged from today; existing tests
(`TestNoArgsIsUsageOnBareAndOKExit`, `TestVersionNoNetwork`) stay green.

## Automated validation

```bash
make all      # gofmt check + vet + lint + race tests
go test -race ./internal/cli/...
```
Expected: all pass, including new per-command tests (help output, offline
guarantee, `--json` shapes, exit codes) and the preserved existing tests.

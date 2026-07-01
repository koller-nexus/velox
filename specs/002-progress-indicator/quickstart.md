# Quickstart: Validate the `velox --check-internet` loading indicator

**Feature**: 002-progress-indicator | **Date**: 2026-06-30

Runnable checks that prove the loading indicator works end to end and does not
break existing behavior. Contracts:
[cli-interface.md](./contracts/cli-interface.md),
[progress-reporter.md](./contracts/progress-reporter.md). Requirements/criteria
live in [spec.md](./spec.md).

## Prerequisites

- Go 1.26.4 (`go version`).
- Build the binary: `make build` (or `go build -o bin/velox ./cmd/velox`).
- A working internet connection for the live scenarios.
- A real terminal (TTY) for the interactive scenarios.

## Automated checks (offline, must pass)

```bash
go test ./...          # unit tests, incl. internal/progress and updated speedtest/cli
go test -race ./...    # indicator goroutine + SetPhase must be race-free (I5)
gofmt -l . ; go vet ./...
golangci-lint run
```

Expected: all green; new tests in `internal/progress`, `internal/speedtest`, and
`internal/cli` pass. `go test -race` reports no data race.

## Scenario 1 — Indicator shows on an interactive terminal (US1)

```bash
./bin/velox --check-internet
```

Expected:
- Within ~1s a spinner appears on screen and updates, cycling through
  `selecting server…`, `checking connectivity…`, `measuring download…`,
  `measuring upload…` with elapsed seconds (SC-001/SC-002).
- When the test finishes, the spinner line is gone and the results occupy a clean
  line (SC-007). No leftover spinner glyph above the results.

## Scenario 2 — stdout stays clean when redirected (US2)

```bash
./bin/velox --check-internet > result.txt 2> progress.log
cat result.txt          # human results only — no spinner glyphs / escape codes
grep -c $'\x1b' result.txt   # expect 0 (no escape sequences in stdout)
```

Because stdout is redirected (non-TTY for that stream) the indicator is suppressed
for anything written to a file. With stderr also redirected to a file it is
suppressed there too (SC-003).

## Scenario 3 — `--json` yields exactly one valid JSON document (US2)

```bash
./bin/velox --check-internet --json | jq .    # must parse; single object
./bin/velox --check-internet --json | python3 -c 'import sys,json; json.load(sys.stdin); print("OK")'
```

Expected: `jq`/`json.load` succeed → stdout is one valid JSON document, unaffected
by the indicator (SC-004). Running the same command interactively (not piped) may
show a spinner on stderr while stdout still pipes clean JSON.

## Scenario 4 — Suppressed in non-interactive / CI (US2)

```bash
./bin/velox --check-internet | cat        # piped: no animation, no escape codes on stdout
NO_COLOR=1 ./bin/velox --check-internet   # honored as non-interactive → suppressed
TERM=dumb ./bin/velox --check-internet    # dumb terminal → suppressed
```

Expected: no spinner frames, no cursor-control sequences (FR-004/SC-003).

## Scenario 5 — `--no-progress` and `--verbose` disable the indicator on a TTY (US2/FR-011/FR-009)

```bash
./bin/velox --check-internet --no-progress    # in a real terminal
./bin/velox --check-internet --verbose        # verbose text narrates progress; no spinner
```

Expected: `--no-progress` → no spinner at all; output identical to pre-feature
behavior; exit code unchanged (FR-010). `--verbose` → the animated indicator is
suppressed and the existing `velox: …` diagnostic lines are printed cleanly with no
garbling (FR-009).

## Scenario 6 — Cancel cleanly with Ctrl-C (US3)

```bash
./bin/velox --check-internet     # then press Ctrl-C mid-run
```

Expected: the animation stops immediately, the cursor is visible, and the next
shell prompt starts on a clean line with no stranded spinner (SC-005). Exit code is
non-zero.

## Scenario 7 — Offline fails fast without a lingering spinner (US1 AS#4)

```bash
# Disable networking, then:
./bin/velox --check-internet
```

Expected: the indicator stops as soon as connectivity fails; the offline message is
printed on stderr; exit code `1`; no spinner left spinning.

## Success mapping

| Scenario | Validates |
|----------|-----------|
| 1 | FR-001, FR-002, FR-006 · SC-001, SC-002, SC-007 |
| 2 | FR-003, FR-004 · SC-003 |
| 3 | FR-005 · SC-004 |
| 4 | FR-004 · SC-003 |
| 5 | FR-011, FR-009 · FR-010 |
| 6 | FR-007 · SC-005 |
| 7 | FR-001 · SC-007 |
| Automated | FR-008 (no skew via presentation-only design), Principle III/IV |

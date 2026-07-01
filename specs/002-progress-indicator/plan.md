# Implementation Plan: Loading Indicator for `velox --check-internet`

**Branch**: `002-progress-indicator` | **Date**: 2026-06-30 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-progress-indicator/spec.md`

## Summary

Add a live loading indicator to `velox --check-internet` so a run that takes tens
of seconds no longer looks frozen. When stderr is an interactive terminal, an
animated spinner shows the current phase (selecting server → checking connectivity
→ measuring download → measuring upload) plus elapsed seconds, then clears itself
before the final results are printed. The indicator writes only to stderr, so
human and `--json` results on stdout are never touched. In any non-interactive
context (pipe, redirect, CI, dumb/`NO_COLOR` terminal) it is suppressed entirely,
and a new `--no-progress` flag disables it even on a TTY.

Technical approach: a small stdlib-only `internal/progress` package renders the
spinner and is TTY-gated via `os.ModeCharDevice` (no new dependency). Phase
visibility comes from a one-method `speedtest.Reporter` interface the runner calls
at each phase boundary; the CLI implements the mapping from `speedtest.Phase` to a
display label and drives the spinner. This is presentation-only: no change to what
is measured, the result schema, or exit codes.

## Technical Context

**Language/Version**: Go 1.26.4 (unchanged; `go.mod`).

**Primary Dependencies**:
- Standard library only for this feature: `os`, `io`, `sync`, `time`, `context`, `fmt`, `strings`.
- TTY detection via `os.File.Stat()` + `os.ModeCharDevice` — **no new third-party dependency** (Principle V). `golang.org/x/term` is deliberately not added.
- Reuses existing runtime dep `github.com/m-lab/ndt7-client-go` indirectly (unchanged).

**Storage**: None. The indicator is transient in-memory UI state; nothing persisted.

**Testing**: Go `testing`, table-driven, co-located `*_test.go`, `go test -race`.
Spinner logic tested via a pure frame-rendering function and a buffer-backed
indicator forced enabled/disabled; no real TTY required. Runner phase-event
ordering tested with a recording fake `Reporter`.

**Target Platform**: Same single static binary (Linux, macOS, Windows; amd64+arm64). ANSI escape sequences used are portable across modern terminals; Windows 10+ consoles support them.

**Project Type**: CLI tool (single project) — extends the existing `001-check-internet` feature.

**Performance Goals**: Indicator visible < 1s after invocation (SC-001); repaint at ~8–12 fps; zero measurable effect on reported latency/download/upload (FR-008/SC-006) — the spinner runs on its own goroutine/ticker and performs no network I/O.

**Constraints**: stderr-only output (FR-003); suppressed unless stderr is a TTY (FR-004); stdout stays a single valid JSON document under `--json` (FR-005); terminal restored (cursor visible, clean line) on completion, failure, and SIGINT (FR-006/FR-007); no change to results, units, or exit codes (FR-010).

**Scale/Scope**: One new internal package (`internal/progress`), one new one-method interface in `internal/speedtest`, edits to `internal/cli` (flag + wiring + adapter) and `internal/cli/help.go`; ~3 focused test files.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Evaluated against Velox Constitution v1.0.0:

| Principle | Gate | Status |
|-----------|------|--------|
| I. Clean Code & Idiomatic Go | gofmt + go vet + golangci-lint clean; small functions; wrapped errors; doc comments on exported ids | PASS — small `progress` package, single-method interface |
| II. CLI-First Interface | results→stdout, diagnostics/progress→stderr; `--json` stdout stays pure; SIGINT restores terminal; `--help` offline documents new flag; exit codes unchanged | PASS — indicator is stderr-only, gated, and cleared |
| III. Test-First (NON-NEGOTIABLE) | TDD; no real network/TTY in unit tests; behavior behind interfaces + fakes; regression tests | PASS — `Reporter` interface + buffer-backed indicator make it fully testable offline |
| IV. Measurement Accuracy & Reliability | no measurement change; context timeouts unchanged; cancellable; `-race` clean | PASS — presentation-only; goroutine guarded by mutex + done channel |
| V. Simplicity & Minimal Dependencies | stdlib-first; no new dependency; no speculative abstraction; single static binary | PASS — zero new deps; `Reporter` has one method and one consumer |

**Initial gate: PASS.** No violations; Complexity Tracking not required. The
`--no-progress` flag and the one-method `Reporter` interface are each justified by
a concrete, present need (explicit opt-out per clarification; per-phase labels per
FR-002), not speculation.

**Post-Design re-check (after Phase 1): PASS.** The design adds no runtime
dependency, keeps the `progress` package free of any `speedtest`/`cli` import
(clean dependency direction), and confines terminal-control sequences to the
TTY-enabled path. No new violations introduced.

## Project Structure

### Documentation (this feature)

```text
specs/002-progress-indicator/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── cli-interface.md        # CLI delta: --no-progress flag + progress behavior contract
│   └── progress-reporter.md    # Internal Reporter interface + Indicator behavior contract
├── checklists/
│   └── requirements.md  # Spec quality checklist (from /speckit-specify)
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
internal/
├── progress/                     # NEW: stdlib-only terminal loading indicator
│   ├── indicator.go              #   Indicator: Start/SetPhase/Stop; TTY gating; spinner goroutine; frame renderer
│   └── indicator_test.go         #   Tests: disabled no-op (non-TTY), pure frame() rendering, idempotent Stop
│
├── speedtest/
│   ├── result.go                 # (unchanged) Phase constants reused as reporter keys
│   ├── run.go                    # EDIT: add Reporter param to Run; emit Phase(...) at each boundary
│   └── run_test.go               # EDIT: pass reporter; assert phase-event ordering + nil-safe
│
└── cli/
    ├── app.go                    # EDIT: add --no-progress flag; StderrF field; build indicator; wire phases
    ├── progress.go               # NEW: phaseReporter adapter (speedtest.Phase -> label) + indicator factory
    ├── help.go                   # EDIT: document --no-progress in usage
    ├── render.go                 # (unchanged)
    └── app_test.go               # EDIT: update fake SpeedRunner signature; --no-progress parsing test

cmd/velox/main.go                 # (unchanged) SIGINT already cancels ctx; runRoot defer restores terminal

README.md                         # EDIT (docs task): document --no-progress + loading behavior
CHANGELOG.md                      # EDIT (docs task): add "Added: loading indicator / --no-progress"
```

**Structure Decision**: Keep the idiomatic `cmd/` + `internal/` layout. The new
`internal/progress` package is a generic, dependency-free terminal spinner that
knows nothing about speedtest — the `internal/cli` package owns the
`speedtest.Phase → display label` mapping and implements the `speedtest.Reporter`
interface. This preserves clean dependency direction (`cli → progress`,
`cli → speedtest`, `speedtest` defines the `Reporter` it consumes, `progress`
imports only stdlib) and keeps the indicator reusable by other future commands
without pulling in measurement types. Unit tests stay co-located and network-/TTY-free.

## Complexity Tracking

> No constitution violations. Section intentionally empty.

No new runtime dependency is introduced. The `Reporter` interface is a single
method with exactly one production consumer (the CLI spinner) and one production
producer (the runner), satisfying Principle V's "second concrete use" test via the
recording fake used in tests plus the real adapter.

# Phase 0 Research: Loading Indicator for `velox --check-internet`

**Feature**: 002-progress-indicator | **Date**: 2026-06-30

All three spec clarifications (indicator content, non-TTY behavior, opt-out flag)
were resolved in `/speckit-clarify`, so no `NEEDS CLARIFICATION` markers remain.
This document records the design decisions that turn those answers into an
implementable plan.

## Decision 1: TTY detection without a new dependency

- **Decision**: Detect whether the indicator's target stream (stderr) is an
  interactive terminal with the standard library only:
  `fi, err := f.Stat(); charDev := err == nil && fi.Mode()&os.ModeCharDevice != 0`.
  Treat the stream as **non-interactive** (indicator disabled) when `f` is `nil`
  (as in tests), when it is not a character device, when `NO_COLOR` is a non-empty
  environment variable, or when `TERM` is unset or equal to `dumb`.
- **Rationale**: Constitution Principle V mandates stdlib-first and justified
  dependencies. `os.ModeCharDevice` distinguishes a terminal from a
  pipe/redirect/regular file on all target platforms with no cgo. Because a
  `TERM=dumb` terminal is still a character device, the explicit `TERM`
  empty/`dumb` check is what actually routes dumb terminals to the disabled path —
  `os.ModeCharDevice` alone would not. Together these satisfy FR-004 and SC-003.
- **Alternatives considered**:
  - `golang.org/x/term.IsTerminal` — robust and also gives width, but adds a new
    (indirect→direct) dependency for a check we can do in one stdlib line.
    Rejected under Principle V; revisit only if width-aware truncation becomes
    required.
  - `github.com/mattn/go-isatty` — popular, but same dependency objection and
    less standard than `x/term`. Rejected.

## Decision 2: Spinner rendering + terminal control

- **Decision**: Render a single-line indicator on stderr of the form
  `⠋ measuring download… 12s` and repaint it in place using a carriage return and
  clear-to-end-of-line: write `"\r" + frame + "\x1b[K"` each tick. Hide the cursor
  on start (`\x1b[?25l`) and restore it on stop (`\x1b[?25h`), and clear the line
  on stop (`\r\x1b[K`). Use a Braille/ASCII spinner frame set cycled by a
  `time.Ticker` at ~100ms. **Emit none of these sequences when disabled.**
- **Rationale**: `\r` + `\x1b[K` is the minimal, widely-portable way to animate a
  single line without a TUI framework, keeping the package tiny (Principle I/V).
  Clearing on stop satisfies FR-006/SC-007 (no stranded artifacts); cursor
  restore satisfies FR-007/SC-005. Modern Windows 10+ consoles process these
  sequences; older/dumb terminals (`TERM` empty/`dumb`) fall into the disabled
  path via the explicit `TERM` check in Decision 1, so no escape sequences reach
  a terminal that cannot handle them.
- **Alternatives considered**:
  - A TUI library (e.g., `bubbletea`, `pterm`, `briandowns/spinner`) — heavyweight
    for one spinner line; violates Principle V. Rejected.
  - Full-screen/alt-buffer rendering — unnecessary and hostile to scrollback.
    Rejected.

## Decision 3: Phase visibility via a one-method `speedtest.Reporter`

- **Decision**: Add `type Reporter interface { Phase(p Phase) }` to
  `internal/speedtest`. `Runner.Run` gains a trailing `reporter Reporter`
  parameter and calls it at each boundary: `PhaseConnectivity` before the dial,
  `PhaseDownload` before the download call, `PhaseUpload` before the upload call.
  A nil reporter is ignored via a small `report(r, p)` helper. The `internal/cli`
  package implements the interface with a `phaseReporter` adapter that maps each
  `speedtest.Phase` to a human label and calls `indicator.SetPhase(label)`. The
  CLI sets the "selecting server" label itself before `selectServer`, since that
  phase is CLI-orchestrated and outside the runner.
- **Rationale**: Phases run *inside* `Runner.Run` today (connectivity → download
  [+latency] → upload) and the runner returns only a final `MeasurementResult`, so
  the CLI cannot see phase boundaries without an event hook. A one-method
  interface consumed by the runner (defined in the package that calls it) is the
  idiomatic, minimal seam and satisfies FR-002/SC-002. Keeping the `Phase → label`
  mapping in `cli` keeps `progress` free of measurement types (clean layering).
- **Alternatives considered**:
  - Streaming live throughput samples out of the ndt7 client — richer UX but
    explicitly ruled out by the clarification (spinner + label + elapsed, no live
    numbers) and would enlarge the runner/ndt7 contract. Rejected for this feature.
  - CLI wraps the whole `Run` with a single static "testing…" label — cannot name
    the active phase, fails FR-002/SC-002. Rejected.
  - Channel of phase events instead of a callback — more moving parts and
    lifecycle to manage than a synchronous one-method call. Rejected (YAGNI).

## Decision 4: Interaction with `--json` and `--verbose`

- **Decision**: The indicator is enabled only when **all** hold: stderr is an
  interactive terminal (Decision 1), `--no-progress` is not set, **and `--verbose`
  is not set**. `--json` does **not** suppress it, because the spinner writes only
  to stderr while JSON is written to stdout — the two streams never mix
  (FR-003/FR-005). Under `--verbose`, the indicator is **suppressed entirely**: the
  existing verbose diagnostic lines (e.g., `velox: testing against …`) already
  narrate progress, so showing an animated spinner on the same stream would be
  redundant and risk garbling. Letting text diagnostics own stderr in verbose mode
  is the simplest way to guarantee FR-009 with no interleaving logic.
- **Rationale**: Honors the stream contract literally; avoids surprising users who
  run `--json` interactively and still want feedback; and eliminates a whole class
  of cursor/line-interleaving bugs by not mixing an animation with free-form
  verbose text. Keeps stdout byte-for-byte pure for SC-004.
- **Alternatives considered**:
  - Keep the spinner under `--verbose` and clear/repaint its line around every
    verbose write — more moving parts and fragile ordering for no real benefit,
    since verbose already narrates progress. Rejected (Principle V / FR-009).
  - Suppress the spinner whenever `--json` is set — simpler but removes useful
    interactive feedback and is not required by any FR. Rejected; documented so it
    can be revisited if users prefer it.

## Decision 5: Cancellation and terminal restoration

- **Decision**: Rely on the existing signal handling. `cmd/velox/main.go` already
  wraps the root context with `signal.NotifyContext(..., os.Interrupt, SIGTERM)`,
  so SIGINT cancels `ctx`; in-flight network calls return `ctx.Err()` and
  `Run` returns normally. `runRoot` stops the indicator via an explicit
  `Stop()` before rendering results and a deferred `Stop()` as a safety net for
  the cancellation/early-return paths. `Stop()` is idempotent (guarded by a
  `sync.Once`/flag), restores the cursor, and clears the line. Because `os.Exit`
  in `main` runs only after `app.Run` returns, the deferred `Stop()` always
  executes first.
- **Rationale**: Satisfies FR-007/SC-005 without a second signal mechanism
  (Principle V). No SIGINT handler in the `progress` package — the process-level
  handler already owns cancellation.
- **Alternatives considered**: A dedicated signal handler inside `progress` to
  restore the cursor — duplicates existing behavior and risks double-handling.
  Rejected.

## Decision 6: No measurement skew (FR-008)

- **Decision**: The spinner runs on its own goroutine driven by a `time.Ticker`
  and performs no network or measurement work; it only formats a string and writes
  to stderr. Shared state (current label, elapsed start) is protected by a
  `sync.Mutex`; the goroutine exits on a `done` channel closed by `Stop()`.
- **Rationale**: Guarantees the indicator adds no network load and cannot alter
  timing of the measured phases (SC-006), and keeps the concurrency `-race` clean
  (Principle IV).

## Decision 7: Narrow-terminal handling (accepted limitation)

- **Decision**: Keep the rendered line short and fixed (spinner glyph + a concise
  label + `NNs`, well under ~30 columns) rather than querying terminal width. On
  extremely narrow terminals the line may wrap; this is an accepted, documented
  limitation.
- **Rationale**: Robust width-aware truncation requires terminal-size queries that
  the stdlib does not expose (would pull in `golang.org/x/term`). Given labels are
  short by construction, the practical risk is negligible, and avoiding the
  dependency upholds Principle V. Revisit only if real-world reports show wrapping.
- **Alternatives considered**: Add `golang.org/x/term` for `GetSize` — rejected now
  for the same dependency reason as Decision 1.

## Resolved unknowns

| Unknown | Resolution |
|---------|------------|
| What to show during download/upload | Spinner + phase label + elapsed seconds; no live numbers (clarified) |
| Non-TTY behavior | Suppress entirely (clarified) |
| Explicit opt-out | `--no-progress` flag (clarified) |
| TTY detection method | `os.ModeCharDevice` + `TERM` empty/`dumb` + `NO_COLOR` checks, stdlib-only (Decision 1) |
| How the CLI learns phase boundaries | `speedtest.Reporter` one-method interface (Decision 3) |
| `--json` interaction | Not suppressed (spinner is stderr-only; stdout stays pure JSON) (Decision 4) |
| `--verbose` interaction | Indicator suppressed; verbose text narrates progress (Decision 4) |
| Terminal restoration on Ctrl-C | Existing `NotifyContext` + idempotent deferred `Stop()` (Decision 5) |

No open `NEEDS CLARIFICATION` items remain. Ready for Phase 1.

# Phase 1 Data Model: Loading Indicator for `velox --check-internet`

**Feature**: 002-progress-indicator | **Date**: 2026-06-30

This feature is presentation-only. It introduces **no persisted data**, no config
fields, and no changes to the `MeasurementResult` schema
(`specs/001-check-internet/contracts/result.schema.json`). The "entities" here are
transient in-memory UI/control types.

## Entity: Indicator (internal/progress)

The animated loading indicator. Owns terminal output and its own animation
goroutine.

| Field | Type | Notes |
|-------|------|-------|
| `w` | `io.Writer` | Target stream (stderr in production). |
| `enabled` | `bool` | True only when the target is an interactive terminal (character device, `TERM` not empty/`dumb`, `NO_COLOR` unset) and the caller did not disable it (`--no-progress`, `--verbose`). When false, all methods are no-ops that write nothing. |
| `label` | `string` | Current phase label, guarded by `mu`. |
| `start` | `time.Time` | Set on `Start()`; drives the elapsed-seconds display. |
| `frames` | `[]string` | Spinner glyph cycle. |
| `mu` | `sync.Mutex` | Guards `label` (read by the animation goroutine, written by `SetPhase`). |
| `done` | `chan struct{}` | Closed by `Stop()` to end the animation goroutine. |
| `stopped` | `bool` / `sync.Once` | Makes `Stop()` idempotent. |

**Lifecycle / state transitions**:

```
new (idle) --Start()--> animating --SetPhase(x)*--> animating --Stop()--> stopped
                                   \----------------- Stop() ------------> stopped
```

- `New(w io.Writer, f *os.File) *Indicator`: decides `enabled` via TTY detection
  (`f` non-nil and a character device, `TERM` not empty/`dumb`, `NO_COLOR` unset).
  A disabled indicator is still a valid object; every method is a safe no-op. The
  caller-level opt-outs (`--no-progress`, `--verbose`) are applied by the CLI
  factory before construction (it passes a `nil`/disabled indicator when either is
  set).
- `Start()`: if enabled, records `start`, hides cursor, launches the ticker
  goroutine that repaints `frame(glyph, label, elapsed)`; if disabled, does
  nothing.
- `SetPhase(label string)`: updates `label` under `mu`; next repaint reflects it.
  Safe before `Start` and after `Stop` (no-op once stopped).
- `Stop()`: idempotent; closes `done`, waits for the goroutine to exit, clears the
  line, restores the cursor. Never writes when disabled.

**Validation / invariants**:

- When `enabled == false`, the cumulative bytes written to `w` MUST be zero
  (enforces FR-004/SC-003).
- `Stop()` MUST leave no partial line and MUST restore the cursor (FR-006/FR-007).
- No method performs network or measurement work (FR-008).

**Pure helper** (unit-testable without a terminal):

- `frame(glyph, label string, elapsed time.Duration) string` → e.g.
  `"⠋ measuring download… 12s"`. Deterministic; the basis of rendering tests.

## Entity: Reporter (internal/speedtest)

A one-method observer the runner calls at phase boundaries. Decouples measurement
from presentation.

```go
// Reporter receives phase-transition notifications during a run.
// A nil Reporter is ignored. Calls happen synchronously on the run goroutine.
type Reporter interface {
    Phase(p Phase)
}
```

| Aspect | Contract |
|--------|----------|
| Producer | `speedtest.Runner.Run` calls `Phase(p)` once when a phase begins. |
| Emitted phases | `PhaseConnectivity`, then `PhaseDownload`, then `PhaseUpload`, in order. `PhaseLatency` is folded into the download phase (latency is measured during download) and is not separately emitted. |
| Nil-safety | Guarded by `report(r Reporter, p Phase)`; `nil` → no call. |
| Threading | Called from the run goroutine; implementations must be safe if they touch shared state (the CLI adapter forwards to the mutex-guarded `Indicator`). |
| Ordering | Monotonic by phase start; a phase event fires even if that phase later fails. |

## Entity: phaseReporter (internal/cli)

Adapter implementing `speedtest.Reporter`, mapping measurement phases to display
labels and forwarding to the `Indicator`.

| `speedtest.Phase` | Display label |
|-------------------|---------------|
| (CLI-owned, before `selectServer`) | `selecting server…` |
| `PhaseConnectivity` | `checking connectivity…` |
| `PhaseDownload` | `measuring download…` |
| `PhaseUpload` | `measuring upload…` |

- Holds a `*progress.Indicator`; `Phase(p)` looks up the label and calls
  `ind.SetPhase(label)`.
- The `selecting server…` label is set directly by `runRoot` before calling
  `selectServer`, since server selection happens outside the runner.

## Relationships

```
cli.runRoot
  ├── creates progress.Indicator(stderr, StderrF)   // enabled iff TTY && !--no-progress && !--verbose
  ├── ind.Start()
  ├── ind.SetPhase("selecting server…"); selectServer(...)
  ├── speedtest.Runner.Run(ctx, server, dist, phaseReporter{ind})
  │        └── reporter.Phase(PhaseConnectivity|PhaseDownload|PhaseUpload)
  │                 └── ind.SetPhase(mapped label)
  ├── ind.Stop()                                     // before rendering (clean line)
  └── renderHuman / renderJSON  → stdout             // unchanged
```

Dependency direction (no cycles): `cli → progress`, `cli → speedtest`,
`speedtest` defines `Reporter` (consumed internally), `progress` imports **stdlib
only**.

## Out of scope (explicitly no data change)

- `MeasurementResult`, `PhaseOutcome`, `ServerInfo` — unchanged.
- Config / consent store — unchanged; `--no-progress` is a per-invocation flag,
  not a persisted preference.

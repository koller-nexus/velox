# Internal Contract: Progress Reporter & Indicator

**Feature**: 002-progress-indicator | **Date**: 2026-06-30

These are the internal Go contracts the implementation must satisfy. They are the
testable seams that keep the feature offline-testable (Constitution Principle III).

## `speedtest.Reporter` (producer: `speedtest.Runner.Run`)

```go
// Reporter receives phase-transition notifications during a run.
// A nil Reporter is ignored. Calls are synchronous on the run goroutine.
type Reporter interface {
    Phase(p Phase)
}
```

### Runner signature change

```go
// Before:
func (r *Runner) Run(ctx context.Context, server locate.Server, distanceKm *float64) (res MeasurementResult)

// After:
func (r *Runner) Run(ctx context.Context, server locate.Server, distanceKm *float64, reporter Reporter) (res MeasurementResult)
```

The corresponding `cli.SpeedRunner` interface gains the same `reporter` parameter.

### Behavioral contract

| ID | Requirement |
|----|-------------|
| R1 | `Run` calls `reporter.Phase(PhaseConnectivity)` before the connectivity dial. |
| R2 | `Run` calls `reporter.Phase(PhaseDownload)` before the download call (latency is measured within this phase; `PhaseLatency` is not separately reported). |
| R3 | `Run` calls `reporter.Phase(PhaseUpload)` before the upload call. |
| R4 | Phase events fire in order R1 → R2 → R3; a later phase is still announced even if an earlier phase failed only when the run reaches it (if connectivity fails, download/upload are skipped and not announced). |
| R5 | A `nil` reporter causes no calls and no panic (guarded by `report(r, p)`). |
| R6 | Reporting adds no measurable latency and no network work; it MUST NOT change any field of `MeasurementResult` (FR-008/FR-010). |

### Test obligations (speedtest)

- A recording fake `Reporter` asserts the emitted sequence for: healthy run
  (`connectivity, download, upload`), offline run (`connectivity` only), and a
  download-failure run (`connectivity, download, upload` — upload still attempted).
- `Run(ctx, server, dist, nil)` behaves identically to the pre-feature `Run`
  (nil-safe), verified against existing result assertions.

## `progress.Indicator` (consumer of terminal; driven by CLI)

```go
func New(w io.Writer, f *os.File) *Indicator   // enabled iff f is an interactive TTY (char device), TERM not empty/dumb, and NO_COLOR unset
func (i *Indicator) Start()
func (i *Indicator) SetPhase(label string)
func (i *Indicator) Stop()                      // idempotent
```

### Behavioral contract

| ID | Requirement |
|----|-------------|
| I1 | When disabled (non-TTY, `nil` file, `TERM` empty/`dumb`, or `NO_COLOR` set), `Start`/`SetPhase`/`Stop` write **zero** bytes to `w` (FR-004/SC-003). |
| I2 | When enabled, `Start` hides the cursor and begins repainting a single stderr line at a fixed cadence; the line shows the current label and elapsed whole seconds (SC-001/SC-002). |
| I3 | `SetPhase` updates the displayed label on the next repaint; safe to call before `Start` and after `Stop`. |
| I4 | `Stop` is idempotent, ends the animation goroutine, erases the line, and restores the cursor — leaving no partial line (FR-006/FR-007/SC-005/SC-007). |
| I5 | Concurrent `SetPhase` (from the run goroutine) and the internal repaint goroutine are race-free (`go test -race`). |
| I6 | The indicator performs no network/measurement work (FR-008). |

### Pure helper (deterministic, no terminal)

```go
func frame(glyph, label string, elapsed time.Duration) string
// e.g. frame("⠋", "measuring download…", 12*time.Second) == "⠋ measuring download… 12s"
```

### Test obligations (progress)

- `frame(...)` table test for label/elapsed formatting (0s, 5s, 65s).
- Disabled indicator: buffer remains empty across `Start`→`SetPhase`→`Stop` (I1).
- Enabled-for-test indicator (buffer as writer, forced-enabled path): output
  contains the current label and, after `Stop`, ends with the line-clear + cursor-
  restore sequence and no dangling escape (I4). (Force-enable via an unexported
  test constructor to avoid needing a real TTY.)
- `Stop` called twice does not panic and writes the restore sequence at most once
  (I4 idempotency).

## `cli.phaseReporter` (adapter: `speedtest.Reporter` → `progress.Indicator`)

| ID | Requirement |
|----|-------------|
| A1 | Implements `speedtest.Reporter`; `Phase(p)` maps `p` to a label and calls `indicator.SetPhase(label)`. |
| A2 | Mapping: `PhaseConnectivity→"checking connectivity…"`, `PhaseDownload→"measuring download…"`, `PhaseUpload→"measuring upload…"`. |
| A3 | The `"selecting server…"` label is set by `runRoot` before `selectServer` (not via the runner). |
| A4 | With a disabled indicator, `phaseReporter` calls are harmless no-ops (inherits I1). |
| A5 | The CLI indicator factory enables the indicator only when stderr is an interactive TTY **and** neither `--no-progress` nor `--verbose` is set; otherwise it constructs a disabled indicator (FR-004/FR-009/FR-011). |

### Test obligations (cli)

- `--no-progress` parses successfully and the run still returns the correct exit
  code (using the existing buffer-based `App` test harness, where the indicator is
  disabled anyway because stderr is not a TTY).
- The updated fake `SpeedRunner` accepts the `reporter` parameter; existing
  assertions on results/exit codes stay green (proves FR-010 — no result/exit
  change).

# Feature Specification: Loading Indicator for `velox --check-internet`

**Feature Branch**: `002-progress-indicator`

**Created**: 2026-06-30

**Status**: Draft

**Input**: User description: "Criar um loading ao rodar comando velox --check-internet"

## Clarifications

### Session 2026-06-30

- Q: During download/upload, what should the indicator show? → A: Animated spinner + current phase label + elapsed seconds; no live measurement numbers (keeps the runner contract unchanged and presentation-only).
- Q: How should the indicator behave when output is not an interactive terminal (pipe/redirect/CI)? → A: Suppress it entirely — no progress output at all; `--verbose` remains the way to get diagnostics in logs.
- Q: Should there be an explicit opt-out flag for interactive terminals? → A: Yes — add a `--no-progress` flag that disables the indicator even when stderr is a TTY.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See Live Progress During the Speed Test (Priority: P1)

A user runs `velox --check-internet` and, because a full speed test (server
selection, latency, download, upload) can take tens of seconds, sees an animated
loading indicator that tells them the tool is working and which phase is running.
When the test finishes, the indicator disappears and the results are shown on a
clean line. The user is never left staring at a frozen, silent terminal wondering
whether the command hung.

**Why this priority**: This is the entire point of the feature. Without live
feedback, a multi-second network measurement looks indistinguishable from a hang,
which erodes trust in the tool. This story delivers the core value on its own.

**Independent Test**: Run `velox --check-internet` in an interactive terminal on a
working connection and confirm a moving/updating indicator appears within about a
second, reflects each phase as it runs, and is cleared when the final results
print.

**Acceptance Scenarios**:

1. **Given** an interactive terminal and a working connection, **When** the user runs `velox --check-internet`, **Then** a loading indicator appears promptly and keeps visibly updating until results are ready.
2. **Given** the test is running, **When** it moves from one phase to the next (e.g., latency → download → upload), **Then** the indicator reflects the phase currently in progress.
3. **Given** the test completes, **When** the final results are rendered, **Then** the indicator is removed and does not interleave with or sit stranded above the results.
4. **Given** the connection is offline, **When** the check fails fast, **Then** the indicator stops immediately and the offline message is shown without a lingering spinner.

---

### User Story 2 - Clean Output for Scripts, Pipes, and JSON (Priority: P1)

A user pipes or redirects `velox --check-internet` (or adds `--json`) to capture
results for a script, log, or CI job. The loading animation must never corrupt
that output: the results stream stays free of animation and terminal control
characters, and `--json` still yields a single valid JSON document.

**Why this priority**: The tool's CLI contract puts results on stdout and
diagnostics on stderr, and machine-readable output must stay parseable. A loading
indicator that leaks escape sequences into captured output would break scripting —
a regression that must not ship, so it is as critical as the indicator itself.

**Independent Test**: Run the command with output piped to a file (non-TTY) and
confirm the captured content contains no spinner frames or escape sequences; run
with `--json | <json parser>` and confirm it parses as exactly one JSON document.

**Acceptance Scenarios**:

1. **Given** stdout is redirected to a file or pipe (not a terminal), **When** the command runs, **Then** the captured output contains no animation frames or cursor-control escape sequences.
2. **Given** `--json` is requested, **When** the test completes, **Then** stdout contains exactly one valid JSON document and nothing from the loading indicator.
3. **Given** a non-interactive environment (CI, cron, no TTY), **When** the command runs, **Then** the indicator does not animate; progress is either suppressed or shown as plain, non-animated status lines on the diagnostics stream.

---

### User Story 3 - Cancel Cleanly Without Breaking the Terminal (Priority: P2)

A user presses Ctrl-C while the loading indicator is showing. The animation stops
at once, the terminal is restored (cursor visible, no half-drawn line), and the
shell prompt returns cleanly.

**Why this priority**: Cancellation is an existing guarantee of the tool; a loading
indicator that hides the cursor or leaves a partial line would visibly break that
guarantee. It ranks just below the core display because it protects an existing
behavior rather than introducing the feature's primary value.

**Independent Test**: Start `velox --check-internet`, press Ctrl-C mid-run, and
confirm the animation stops, the cursor is visible, and the next prompt starts on
a fresh, clean line.

**Acceptance Scenarios**:

1. **Given** the indicator is animating, **When** the user presses Ctrl-C, **Then** the animation stops immediately and the run is cancelled.
2. **Given** cancellation or a mid-run failure, **When** control returns to the shell, **Then** the terminal cursor is visible and the prompt appears on a clean line with no leftover indicator artifacts.

---

### Edge Cases

- What happens when the terminal is very narrow? → The label is kept short by design (spinner glyph + concise phase + `NNs`, well under ~30 columns), so wrapping is not expected in practice; on an extremely narrow terminal the line may wrap. This is an accepted, best-effort limitation — width-aware truncation is out of scope for v1 (avoids a terminal-size dependency).
- What happens on a "dumb"/non-ANSI terminal (`TERM` empty or `dumb`) or when a common `NO_COLOR`/no-animation preference is set? → Treat it as non-interactive and suppress the indicator entirely rather than emitting control sequences the terminal cannot handle.
- What happens when a phase fails partway (e.g., download succeeds, upload drops)? → The indicator stops on the failing phase and gives way to the clear failure message; it never spins forever.
- What happens when the whole run completes almost instantly (e.g., immediate offline detection)? → A brief or no indicator is acceptable, but it must always be cleared and must not flash stranded artifacts.
- What happens when `--no-progress` is passed on an interactive terminal? → No indicator is shown at all; results render exactly as they would without this feature.
- What happens under `--verbose`? → The animated indicator is suppressed; the existing verbose diagnostic lines narrate progress instead, so the two never interleave or garble each other on stderr.
- What happens when stderr is a terminal but stdout is redirected (or vice versa)? → Interactivity is judged by the stream the indicator is written to; results stay clean on the redirected stream regardless.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: While `velox --check-internet` is running, System MUST display a visible loading indicator that updates over time until the test completes, is cancelled, or fails.
- **FR-002**: The indicator MUST communicate which phase of the run is currently active (e.g., selecting server, checking connectivity, downloading, uploading) and MUST show elapsed time; it MUST NOT display live per-phase measurement numbers (e.g., ticking throughput).
- **FR-003**: The indicator MUST be written only to the diagnostics stream (stderr) and MUST NOT be written to the results stream (stdout), so it never mixes with human-readable or JSON results.
- **FR-004**: When the target stream is not an interactive terminal, System MUST suppress the indicator entirely — emitting no progress output and no animation or cursor-control escape sequences.
- **FR-005**: When `--json` is requested, stdout MUST remain exactly one valid JSON document, entirely unaffected by the indicator.
- **FR-006**: On successful completion, System MUST clear or replace the indicator so it neither interleaves with nor is stranded above the final results.
- **FR-007**: On cancellation (Ctrl-C) or failure, System MUST stop the indicator and restore the terminal to a usable state (cursor visible, no partial line), consistent with the tool's existing signal handling.
- **FR-008**: The indicator MUST NOT add measurable network load or otherwise skew the reported latency, download, or upload figures.
- **FR-009**: When `--verbose` is set, System MUST suppress the animated indicator so the existing verbose diagnostics on stderr remain readable and are never garbled by cursor-control output; verbose diagnostics serve as the progress feedback in that mode.
- **FR-010**: The feature MUST NOT change the content, units, or format of the measurement results, nor the command's exit codes.
- **FR-011**: System MUST provide a `--no-progress` flag that disables the indicator even in an interactive terminal; when set, the run behaves as if no progress display existed and produces no indicator output.

### Key Entities *(include if feature involves data)*

- **Run Phase**: A named stage of the check-internet run that the indicator can display (server selection, latency, download, upload) along with its in-progress / done / failed state; used solely to drive what the indicator shows, not persisted.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On an interactive run over a normal connection, the user sees a loading indicator within 1 second of invoking the command, and it updates at least ~4 times per second until results appear.
- **SC-002**: The indicator identifies the current phase for 100% of the run's phases (server selection, connectivity, download, upload; latency is measured within the download phase and not shown separately).
- **SC-003**: 100% of piped, redirected, or non-TTY runs produce results output containing zero animation frames or terminal control/escape sequences.
- **SC-004**: 100% of `--json` runs emit exactly one document on stdout that a standard JSON parser accepts.
- **SC-005**: After Ctrl-C during the indicator, the terminal cursor is visible and the next shell prompt starts on a clean line in 100% of cancellations.
- **SC-006**: Reported latency, download, and upload values with the indicator active are unchanged versus runs without it, within normal run-to-run variance.
- **SC-007**: On completion, no indicator artifacts remain on screen in 100% of successful runs (the final results occupy a clean line).

## Assumptions

- The loading indicator is a presentation-only addition to the existing `velox --check-internet` flow; it does not alter what is measured, the result schema, or exit codes (aligns with the existing check-internet feature and the project constitution's results-to-stdout / diagnostics-to-stderr contract).
- Progress is diagnostic output and therefore goes to stderr, keeping stdout reserved for human and JSON results.
- "Interactive" is determined by whether the stream the indicator targets (stderr) is a terminal (a character device), with `TERM` unset/`dumb` and `NO_COLOR` unset treated as non-interactive; when it is not interactive, the indicator is suppressed entirely.
- The default presentation is an animated spinner with a short phase label and elapsed seconds; live per-phase measurement numbers (e.g., ticking throughput) and a precise percentage/progress bar are out of scope, since the run is a sequence of phases rather than a single known-size transfer and the runner exposes only a final result (clarified 2026-06-30).
- In non-interactive contexts (pipe, redirect, CI, `TERM` empty/`dumb`, `NO_COLOR`) — and whenever `--verbose` is set — the indicator is suppressed entirely; `--verbose` remains the way to get progress diagnostics in logs (clarified 2026-06-30).
- The feature builds on the existing cancellation/signal handling; it hooks into terminal restoration rather than introducing a separate mechanism.
- Scope is limited to the `velox --check-internet` run; other subcommands (e.g., `velox consent`) are out of scope for this feature. A `--no-progress` flag is in scope as an explicit interactive opt-out (clarified 2026-06-30).

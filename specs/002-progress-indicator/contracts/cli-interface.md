# CLI Contract Delta: velox (loading indicator)

**Feature**: 002-progress-indicator | **Date**: 2026-06-30

This extends the `001-check-internet` CLI contract
(`specs/001-check-internet/contracts/cli-interface.md`). Only the deltas are
listed here; everything else is unchanged.

## New flag on `velox --check-internet`

| Flag | Type | Default | Effect |
|------|------|---------|--------|
| `--no-progress` | bool | false | Disable the loading indicator even when stderr is an interactive terminal (FR-011). Results are unchanged. |

Updated flag table (for reference — additions in **bold**):

| Flag | Type | Default | Effect |
|------|------|---------|--------|
| `--check-internet` | bool | — | Run the full speed test. |
| `--json` | bool | false | Emit a single JSON object to stdout (FR-003/FR-005). |
| `--server <url>` | string | "" | Manually override the test server. |
| `--timeout <dur>` | duration | 60s | Overall run budget. |
| `-v, --verbose` | bool | false | Verbose diagnostics to stderr. |
| **`--no-progress`** | **bool** | **false** | **Disable the loading indicator on a TTY (FR-011).** |

## Loading indicator behavior

The indicator is shown **only when all** of the following hold:

1. The command is `velox --check-internet` (no indicator for `consent`,
   `--help`, `--version`).
2. `--no-progress` is **not** set (FR-011).
3. `--verbose` is **not** set — in verbose mode the text diagnostics narrate
   progress and the animated indicator is suppressed (FR-009).
4. stderr is an interactive terminal — a character device, `TERM` not
   empty/`dumb`, `NO_COLOR` unset (FR-004). Pipe/redirect/CI/dumb terminal ⇒
   suppressed entirely.

When shown, it MUST:

- Write **only to stderr**; stdout is never touched (FR-003).
- Display the active phase and elapsed seconds; cycle through the phases in order:
  `selecting server…` → `checking connectivity…` → `measuring download…` →
  `measuring upload…` (FR-002/SC-002).
- Appear within 1 second of invocation and update visibly until results are ready
  (SC-001).
- Not display live per-phase measurement numbers (clarification: spinner + label +
  elapsed only).
- Be **cleared** (line erased, cursor restored) before the final result is printed,
  and on failure or cancellation (FR-006/FR-007/SC-005/SC-007).

When suppressed, **zero** progress bytes and **zero** escape sequences are written
anywhere (FR-004/SC-003).

## Interaction with existing flags

| Combination | Behavior |
|-------------|----------|
| `--json` + interactive TTY | Indicator still shown on stderr; stdout remains exactly one valid JSON document (FR-005/SC-004). `--json` does **not** suppress the indicator. |
| `--json` + non-TTY (pipe/CI) | Indicator suppressed; stdout is clean JSON. |
| `--verbose` + interactive TTY | Indicator **suppressed**; verbose diagnostics on stderr narrate progress and are never garbled by cursor-control output (FR-009). |
| `--no-progress` (TTY or not) | No indicator at all; run behaves exactly as before this feature (FR-011). |

## Output streams (unchanged contract, restated)

- **stdout**: results only (human summary, or JSON with `--json`).
- **stderr**: prompts, **loading indicator**, diagnostics, errors.

## Exit codes (unchanged)

`0` success · `1` measurement/network failure · `2` usage error. The indicator MUST
NOT change any exit code (FR-010).

## Cancellation (unchanged mechanism, restated for the indicator)

`Ctrl-C` (SIGINT) cancels the in-flight run via the root context. The indicator
stops, clears its line, and restores the cursor before the process exits
non-zero (FR-007/SC-005).

## `--help` delta

`velox --help` MUST list `--no-progress` under the `--check-internet` flags and
MUST continue to work with no network access and no consent prompt (FR-016,
unchanged).

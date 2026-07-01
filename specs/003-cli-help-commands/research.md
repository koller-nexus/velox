# Phase 0 Research: Help Commands & Additional Useful Commands

**Feature**: `003-cli-help-commands` | **Date**: 2026-07-01

No `NEEDS CLARIFICATION` markers remain in the spec (all resolved in
`/speckit-clarify`, Session 2026-07-01). This document records the design
decisions the plan depends on, each grounded in the existing codebase and the
constitution.

## D1 — Command dispatch model

**Decision**: Introduce a small, stdlib-only command registry in
`internal/cli/command.go`. A `Command` value carries `Name`, `Summary`
(one line), a `Usage` string (detailed help), and a `Run(ctx, args) int`
function. `App.Run` looks up `args[0]` in the registry; unknown → usage error;
the root flag form (`--check-internet`, `--version`, `--help`) is preserved for
backward compatibility.

**Rationale**: FR-007 requires `velox help` to list *every* command and stay in
sync. A single registry is the natural single source of truth for both the
overview and per-command help, and it replaces the current ad-hoc
`if args[0] == "consent"` branch in `app.go`. Keeps one obvious place to add
future commands.

**Alternatives considered**:
- **Cobra / urfave/cli**: Rejected — violates Constitution V (minimal
  dependencies); the tool needs only a handful of commands and stdlib `flag`
  already covers parsing.
- **Keep expanding `if/switch` in `Run`**: Rejected — the overview would drift
  from the actual commands (breaks FR-007) and duplicates help text.

## D2 — Per-command `--help` and unknown-input errors

**Decision**: Each command builds its own `flag.FlagSet` with
`flag.ContinueOnError` and output routed to `App.Stderr`, and defines a custom
`Usage` that prints the command's detailed help to **stdout**. A `--help`/`-h`
flag on every command prints that help and returns `ExitOK` without side
effects. Unknown command or unknown flag → clear message to stderr that names
the problem and suggests `velox help`, returning `ExitUsage` (2).

**Rationale**: Matches FR-002/FR-003/FR-004/FR-006 and the existing pattern in
`runRoot`/`runConsent` (both already use `flag.NewFlagSet(..., ContinueOnError)`
with `SetOutput(a.Stderr)`). Routing help text to stdout (not the FlagSet's
stderr default) keeps `velox help <cmd>` pipeable.

**Alternatives considered**: Relying on `flag`'s built-in `-h` usage dump —
rejected because it writes to the FlagSet output and its formatting is not the
detailed, example-bearing help the spec calls for.

## D3 — `ping` (latency-only) measurement path

**Decision**: `ping` reuses the ndt7 **download** subtest connection to obtain
minimum RTT and jitter, reports only latency/jitter (no throughput), and skips
the upload phase entirely. It is bounded by a short timeout. To make a short
window yield a reading instead of an error, `consume` gains an **opt-in**
`partialOnDeadline bool` parameter: only the latency path passes `true`; the
full download call passes `false` and keeps its current contract (returns
`ctx.Err()` on cancellation), so `--check-internet` behavior is unchanged. A
`speedtest`-level `Latency`/`Ping` helper wraps connectivity + the short
download sample and returns a small latency-only result.

**Rationale**: This is the clarified answer (Session 2026-07-01): "reuse the
ndt7 connection and sample RTT over a short window … no ICMP, no elevated
privileges." ndt7 derives RTT from `TCPInfo` during the download subtest
(`internal/ndt7/ndt7.go`, `rttStats`), so the download subtest is the
already-implemented, accurate RTT source. Reporting only latency preserves the
documented "minimum RTT / jitter" definitions (Constitution IV) and adds no
dependency.

**Alternatives considered**:
- **Raw ICMP/TCP ping**: Rejected — raw ICMP typically needs elevated
  privileges (violates the tool's no-privilege promise) and would be a new,
  separate measurement mechanism.
- **Full download phase, hide Mbps**: Rejected — still consumes full bandwidth
  and time; not "quick," contradicting FR-013's intent.

## D4 — `servers` (nearest listing, top ~5)

**Decision**: `servers` calls `locate.Locator.Nearest(ctx)` for candidates,
applies the existing consent gate before any geolocation lookup, and displays
the server velox would select plus the next nearest alternatives — about 5 in
total — each with machine/site, city+country, and approximate distance (km)
when a location estimate is available. A new `geo` helper ranks candidates by
great-circle distance so the top-N can be taken and the chosen one marked; when
no estimate/coords are available it falls back to the Locate proximity order
(the same fallback `geo.SelectNearest` already uses).

**Rationale**: FR-011 + clarification (Session 2026-07-01: "~5 total"). Reuses
`locate.Nearest`, `geo.HaversineKm`, and `geo.SelectNearest`'s existing fallback
semantics. The consent flow mirrors `app.go`'s `selectServer`, so behavior stays
identical to a real run (transparency: "which server *would* velox pick").

**Alternatives considered**: Only the single chosen server (rejected — no
context); all candidates (rejected — noisy list). Adding a full ranking type
hierarchy (rejected — a single sort helper suffices, Constitution V).

## D5 — `config` (local-state inspection, read-only)

**Decision**: `config` prints the config file path (`config.Path()`), its parent
directory, the current consent decision + timestamp, and the effective settings
(`GeoEndpoint`, `FallbackServer` when set) from `config.Load()`. It is
read-only, performs no network I/O, and never prompts. It never prints secret
material (there is none — the store holds only a consent decision and non-secret
overrides).

**Rationale**: FR-012. All needed accessors already exist (`config.Path`,
`config.Load`, `consent.Store.Decision`). Read-only matches the spec's
"inspection" framing and the Assumptions (config *editing* is out of scope).

**Alternatives considered**: A settable `config set …` command — rejected as
out of scope per the spec Assumptions.

## D6 — `version` subcommand

**Decision**: `velox version` prints exactly `version.String()` to stdout and
exits 0 — identical to the `--version` flag. Plain text only (no `--json`).

**Rationale**: FR-010 + clarification (version stays plain text; `--version`
already provides the single-line machine-parseable form). Trivial reuse of
`internal/version`.

## D7 — `--json` output for structured commands

**Decision**: `servers`, `ping`, and `config` each accept `--json`, emitting a
single indented JSON document via a shared `encoding/json` encoder (mirroring
`render.go`'s `renderJSON`). Each command defines a small result struct with
stable field names (see `data-model.md` and `contracts/`). Human output uses
`text/tabwriter` for aligned columns.

**Rationale**: FR-014 + clarification (Session 2026-07-01). Consistent with the
existing `--check-internet --json` contract; scripting-friendly.

**Alternatives considered**: `--json` on `version` too — deferred (single-line
text already parseable; keeps surface minimal).

## D8 — Backward compatibility

**Decision**: Preserve current behavior exactly: bare `velox` and `velox --help`
print the top-level overview (exit 0); `velox --version` prints version;
`velox --check-internet [flags]` unchanged; `velox consent <...>` unchanged
(registered in the registry so it also appears in `help` and gains `--help`).

**Rationale**: FR-005 and existing tests in `internal/cli/app_test.go` (e.g.
`TestNoArgsIsUsageOnBareAndOKExit`, `TestVersionNoNetwork`) must keep passing.

## Open questions

None. All decisions are resolved and consistent with the spec and constitution.

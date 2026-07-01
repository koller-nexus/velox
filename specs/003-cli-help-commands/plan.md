# Implementation Plan: Help Commands & Additional Useful Commands

**Branch**: `003-cli-help-commands` | **Date**: 2026-07-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/003-cli-help-commands/spec.md`

## Summary

Add a discoverable, command-aware help system to velox and four small,
self-contained commands (`version`, `servers`, `config`, `ping`) that are useful
day to day. Today velox dispatches a single `consent` subcommand and otherwise
parses root flags (`--check-internet`, `--version`, `--help`, …) with one static
usage blob. The technical approach is to introduce a lightweight, stdlib-only
**command registry** in `internal/cli` (name, summary, detailed usage, run
function) so that `velox help` and `velox help <command>` stay in sync with the
real command set, every command gains a consistent `--help`, and unknown
commands/flags produce an actionable usage error. The new commands reuse the
existing `locate`, `geo`, `ndt7`/`speedtest`, `config`, and `consent` packages;
no new external dependency is added. Help/version/config never touch the
network; `servers` honors the existing consent gate; `ping` reuses the ndt7
download subtest to obtain latency/jitter only.

## Technical Context

**Language/Version**: Go 1.26.4 (pinned in `go.mod`).

**Primary Dependencies**: Standard library only for the new surface (`flag`,
`encoding/json`, `text/tabwriter` for aligned tables). Existing internal
packages: `internal/cli`, `internal/config`, `internal/consent`,
`internal/geo`, `internal/locate`, `internal/ndt7`, `internal/speedtest`,
`internal/version`. Existing single external dependency
`github.com/m-lab/ndt7-client-go` is reused by `ping`; no new dependency.

**Storage**: JSON config file under the OS user config directory
(`config.Path()` → `<UserConfigDir>/velox/config.json`); read-only for the new
`config`/`version`/`help` commands.

**Testing**: `go test -race ./...`; table-driven unit tests in-package with the
existing fakes (`fakeLocator`, `fakeResolver`, `fakeConsent`, `fakeRunner`) and
a fake `ndt7.Client`; no real network in unit tests. Live integration remains
opt-in under `-tags=integration`.

**Target Platform**: Single static binary for Linux, macOS, Windows (amd64 +
arm64).

**Project Type**: Single-project CLI application.

**Performance Goals**: `help`, `version`, and `config` return in under 1 s and
require no network (SC-003). `ping` completes quickly by running only a
short-window ndt7 download subtest (no upload phase).

**Constraints**: `help`/`version`/`config` MUST be fully offline and MUST NOT
trigger a location lookup or consent prompt (FR-008). No elevated privileges
(rules out raw ICMP for `ping`). Backward compatible: all existing flags and the
`consent` subcommand keep working (FR-005). stdout for results, stderr for
diagnostics; meaningful exit codes `0`/`1`/`2` (Constitution II).

**Scale/Scope**: ~5 new commands, one small command-registry abstraction, and
minor helpers in `geo` (proximity ranking) and `speedtest` (latency-only path).
No persistent data growth.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Assessment | Status |
|-----------|-----------|--------|
| I. Clean Code & Idiomatic Go | New files are small, single-purpose; command registry replaces ad-hoc `if args[0]==` branching; all code passes `gofmt`/`go vet`/`golangci-lint`. | PASS |
| II. CLI-First Interface | Feature *is* CLI surface: help to stdout, errors to stderr, `--json` for structured commands, meaningful exit codes, `--help`/`--version` work offline. | PASS |
| III. Test-First (NON-NEGOTIABLE) | Every command gets failing unit tests first (help output, exit codes, offline guarantee, `--json` shape) using existing fakes; no real network. | PASS |
| IV. Measurement Accuracy & Reliability | `ping` reuses the documented minimum-RTT/jitter definitions from the ndt7 download subtest; all network calls use `context` timeouts and are cancellable. | PASS |
| V. Simplicity & Minimal Dependencies | No new external dependency (no cobra/urfave); stdlib `flag` + a tiny in-package registry; helpers added only where a second concrete use now exists. | PASS |

**Result**: PASS — no violations. Complexity Tracking left empty.

**Post-design re-check (after Phase 1)**: Still PASS. The design added no
external dependency (D1 rejects cobra; D3 reuses the existing ndt7 client), the
new abstractions are one small in-package registry plus two narrowly-scoped
helpers each with a concrete second use (`geo` ranking for top-N, `speedtest`
latency-only path), and every artifact keeps the CLI I/O and offline guarantees
(`contracts/cli-interface.md`). No new violations surfaced during data-model or
contract design.

## Project Structure

### Documentation (this feature)

```text
specs/003-cli-help-commands/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
│   ├── cli-interface.md
│   ├── servers.schema.json
│   ├── ping.schema.json
│   └── config.schema.json
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created here)
```

### Source Code (repository root)

```text
internal/cli/
├── app.go            # Run(): dispatch via command registry (edit)
├── command.go        # NEW: Command struct + registry + dispatch/help helpers
├── help.go           # help command: overview + `help <command>` (rewrite)
├── version.go        # NEW: `velox version` subcommand
├── servers.go        # NEW: `velox servers` (list ~5 nearest; --json)
├── ping.go           # NEW: `velox ping` (latency-only; --json)
├── config.go         # NEW: `velox config` (paths + effective settings; --json)
├── consent.go        # consent subcommand (register in registry; --help)
├── render.go         # add JSON/human renderers for servers/ping/config (edit)
├── progress.go       # unchanged
└── *_test.go         # NEW/edit: per-command unit tests (table-driven, fakes)

internal/geo/
└── select.go         # add RankByDistance (or Rank) helper for top-N (edit)

internal/speedtest/
└── run.go            # add latency-only path (Runner.Latency or Ping) (edit)

internal/ndt7/
└── ndt7.go           # opt-in partial-RTT-on-deadline (latency path only) (edit)
```

**Structure Decision**: Single-project Go CLI (existing layout). The bulk of the
work is in `internal/cli` behind a new `command.go` registry; small, justified
helpers are added to `internal/geo` (ranking for the top-N servers view) and
`internal/speedtest`/`internal/ndt7` (latency-only sampling). Tests live beside
their sources per Go convention, reusing the fakes already in
`internal/cli/app_test.go`.

## Complexity Tracking

> No Constitution violations — section intentionally empty.

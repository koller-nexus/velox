---

description: "Task list for feature 003-cli-help-commands"
---

# Tasks: Help Commands & Additional Useful Commands

**Input**: Design documents from `/specs/003-cli-help-commands/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: INCLUDED — the project constitution mandates Test-First
(NON-NEGOTIABLE) and plan.md's Constitution Check commits to failing unit tests
before implementation. All test tasks below must be written first and must fail
before the matching implementation task.

**Organization**: Tasks are grouped by user story so each story is independently
implementable and testable.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1 (help system) or US2 (additional commands)
- All paths are repository-relative from the repo root.

## Path Conventions

Single-project Go CLI. Production code in `internal/cli/`, `internal/geo/`,
`internal/speedtest/`, `internal/ndt7/`; tests live beside sources as
`*_test.go` (Go convention), reusing fakes from `internal/cli/app_test.go`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm a green baseline before touching the command surface.

- [x] T001 Establish green baseline: run `go build ./...` and `go test -race ./...` from repo root; confirm the existing suite (including `internal/cli/app_test.go`) passes before any change.
- [x] T002 [P] Confirm tooling runs: `make lint` (`.golangci.yml`) and that `make security` (gosec + govulncheck) is available, from repo root.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The command registry and dispatch refactor that BOTH user stories
depend on (research.md D1, D8). No user story work can begin until this is done.

**⚠️ CRITICAL**: Blocks all user stories.

- [x] T003 [P] Write failing test for dispatch of an unknown subcommand → error on stderr containing `velox help` and exit code `2` (ExitUsage), in `internal/cli/command_test.go`.
- [x] T004 [P] Write failing/extended regression tests asserting backward-compatible dispatch still routes: bare invocation → overview exit `0`, `--version`, `--help`, `--check-internet`, and `consent`, in `internal/cli/app_test.go`.
- [x] T005 Define `Command` type (`Name`, `Summary`, `Usage`, `Run func(context.Context, []string) int`) and a registry builder `commands()` (ordered) in `internal/cli/command.go` per data-model.md "Command".
- [x] T006 Implement dispatch + unknown-command usage error in `internal/cli/command.go`: exact-match on `Name`; unknown token → `fmt.Fprintf(stderr, "velox: unknown command %q; run 'velox help'\n", …)` and return `ExitUsage` (FR-004).
- [x] T007 Refactor `App.Run` in `internal/cli/app.go` to dispatch via the registry while preserving existing behavior exactly: bare/`--help` → overview (ExitOK), `--version`, and the `--check-internet` root-flag flow (FR-005, D8).
- [x] T008 Wrap the existing `consent` subcommand as a registry `Command` (`newConsentCommand`) in `internal/cli/consent.go`, preserving its current `--status/--grant/--deny/--reset` flags and exit codes.

**Checkpoint**: Dispatch routes through the registry; all existing tests green; unknown command → exit 2.

---

## Phase 3: User Story 1 - Discoverable, command-aware help (Priority: P1) 🎯 MVP

**Goal**: `velox help`, `velox help <command>`, and `<command> --help` give a
complete, offline, command-aware help surface; unknown input errors point to
`velox help` (FR-001–FR-008).

**Independent Test**: On a machine with no network, run `velox help`,
`velox help consent`, and `velox consent --help`; confirm complete usage on
stdout, exit `0`, and no consent prompt; `velox frobnicate` → stderr error +
`velox help` hint, exit `2`.

### Tests for User Story 1 ⚠️ (write first, must fail)

- [x] T009 [P] [US1] Test `velox help` overview lists every registered command with its one-line summary (FR-001/FR-007), in `internal/cli/help_test.go`.
- [x] T010 [P] [US1] Test `velox help <command>` prints that command's `Usage` to stdout and exit `0`; unknown `<command>` → stderr + exit `2` (FR-002), in `internal/cli/help_test.go`.
- [x] T011 [P] [US1] Test every command's `--help`/`-h` prints its `Usage` to stdout, exits `0`, performs no network/consent/state side effects, and matches `help <command>` output (FR-003/SC-002), in `internal/cli/help_test.go`.

### Implementation for User Story 1

- [x] T012 [US1] Implement the generic help-flag helper in `internal/cli/command.go`: detect `--help`/`-h` on any command's `flag.FlagSet`, print `Usage` to stdout, return `ExitOK`; route unknown flags to a usage error + `velox help` hint (FR-003/FR-004/FR-006).
- [x] T013 [US1] Rewrite `internal/cli/help.go`: implement the `help` command building the overview from the registry via `text/tabwriter` (aligned `Name` + `Summary`, footer pointing to `velox help <command>`), plus `help <command>` lookup (FR-001/FR-002/FR-007).
- [x] T014 [US1] Register the `help` command in `commands()` and make bare invocation and `--help` render the overview from the registry (`internal/cli/command.go`, `internal/cli/app.go`), keeping the old `usage` constant in sync or replacing it.
- [x] T015 [US1] Add detailed `Usage` text and `--help` support to the existing `consent` command and the root/`--check-internet` help (`internal/cli/consent.go`, `internal/cli/help.go`) so all current commands satisfy FR-003.

**Checkpoint**: Help system fully functional and independently testable — MVP complete.

---

## Phase 4: User Story 2 - More useful commands for everyday use (Priority: P2)

**Goal**: Add `version`, `servers`, `config`, and `ping` commands, each
discoverable via the help system and following the CLI contract
(FR-009–FR-014). `--json` for `servers`/`config`/`ping`.

**Independent Test**: With the registry/help in place, invoke each command:
`velox version` equals `--version`; `velox servers [--json]` lists ~5 nearest
(consent-gated); `velox config [--json]` shows paths/settings offline;
`velox ping [--json]` reports latency/jitter only. Validate `--json` against the
schemas in `contracts/`.

### Shared helper tests ⚠️ (write first, must fail)

- [x] T016 [P] [US2] Test `ndt7` `consume` returns the RTT stats gathered so far on context deadline (partial result, not only `ctx.Err()`), in `internal/ndt7/ndt7_test.go` (research.md D3).
- [x] T017 [P] [US2] Test a `geo` ranking helper orders candidates by great-circle distance, marks the nearest, and falls back to registry order when no estimate/coords, in `internal/geo/select_test.go` (research.md D4).
- [x] T018 [P] [US2] Test a `speedtest` latency-only path returns latency/jitter with no throughput fields, using a fake `ndt7.Client`, in `internal/speedtest/run_test.go` (research.md D3).

### Shared helper implementation

- [x] T019 [US2] Adjust `internal/ndt7/ndt7.go` `consume`/`finalize` so a short-window deadline yields the collected RTT statistics for latency sampling instead of an error (depends on T016).
- [x] T020 [P] [US2] Add `geo.RankByDistance` in `internal/geo/select.go` returning an ordered `[]Selection` (nearest first, selected marked), reusing `HaversineKm` and the existing no-estimate fallback (depends on T017).
- [x] T021 [US2] Add a latency-only path (`Runner.Latency`/`Ping`) in `internal/speedtest/run.go` that runs connectivity + a short ndt7 download sample and returns a `LatencyResult` (latency/jitter/server/distance/duration), depends on T019.

### Command tests ⚠️ (write first, must fail)

- [x] T022 [P] [US2] Test `velox version` output equals `velox --version` and requires no network (FR-010), in `internal/cli/version_test.go`.
- [x] T023 [P] [US2] Test `velox servers` human + `--json` (validates against `contracts/servers.schema.json`): consent gate applied, ≤5 entries, exactly one `selected` when a pick exists, `distanceKm` null under fallback (FR-011), in `internal/cli/servers_test.go`.
- [x] T024 [P] [US2] Test `velox ping` human + `--json` (validates against `contracts/ping.schema.json`): latency/jitter only, exit `0` online / `1` offline, `--server` override honored (FR-013), in `internal/cli/ping_test.go`.
- [x] T025 [P] [US2] Test `velox config` human + `--json` (validates against `contracts/config.schema.json`): read-only, no network, reports path/dir/consent/settings (FR-012), in `internal/cli/config_test.go`.

### Command implementation

- [x] T026 [P] [US2] Implement the `version` command + `Usage` in `internal/cli/version.go` reusing `version.String()` (plain text, no `--json`).
- [x] T027 [P] [US2] Implement the `config` command + human/JSON rendering + `Usage` in `internal/cli/config.go` using `config.Path`/`config.Load` and `ConfigView` (data-model.md); offline, read-only.
- [x] T028 [P] [US2] Implement the `servers` command + human (`text/tabwriter`) / JSON rendering + `Usage` in `internal/cli/servers.go` using `Locator.Nearest`, the consent gate (mirroring `selectServer`), and `geo.RankByDistance`; shape `ServerListing` (data-model.md), depends on T020.
- [x] T029 [P] [US2] Implement the `ping` command + human/JSON rendering + `Usage` in `internal/cli/ping.go` using server selection (consent gate, `--server`, `--timeout`) and `Runner.Latency`; shape `LatencyResult` (data-model.md), depends on T021.
- [x] T030 [US2] Register `version`, `servers`, `ping`, and `config` in `commands()` (`internal/cli/command.go`); confirm they appear in `velox help` and reuse the existing `App` dependencies (`Locator`, `Runner`, `Consent`, `NewResolver`, `LoadConfig`) already wired in `NewApp` (`internal/cli/app.go`).

**Checkpoint**: All four commands work, are discoverable in `velox help`, and honor the CLI/`--json` contract.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Docs, quality gates, and end-to-end validation across both stories.

- [x] T031 [P] Update `README.md` Usage section with `help`, `version`, `servers`, `ping`, `config` (and examples), per Constitution "Documentation" gate.
- [x] T032 [P] Add a `CHANGELOG.md` entry describing the new command surface.
- [x] T033 Run `make lint` (gofmt check + vet + golangci-lint) from repo root and resolve findings.
- [x] T034 Run `make security` (gosec + govulncheck) and confirm no new HIGH+ findings.
- [x] T035 Execute the `quickstart.md` scenarios against `./bin/velox` (`make build`) and confirm expected outputs and exit codes (incl. offline scenarios 2 & 8).
- [x] T036 Run full `make all` (gofmt + vet + lint + race tests) and confirm green.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies — start immediately.
- **Foundational (Phase 2)**: depends on Setup — BLOCKS both user stories.
- **User Story 1 (Phase 3)**: depends on Foundational. Delivers the MVP.
- **User Story 2 (Phase 4)**: depends on Foundational; soft-depends on US1 for
  discoverability (each command's action is independently testable via direct
  invocation, but commands appear in `velox help` only once US1's overview
  exists).
- **Polish (Phase 5)**: depends on all targeted stories being complete.

### User Story Dependencies

- **US1 (P1)**: needs only Foundational. No dependency on US2.
- **US2 (P2)**: needs Foundational; integrates with US1's registry/help but is
  independently testable per command.

### Within-story ordering

- Tests before implementation (constitution: test-first).
- US2 helpers before the commands that use them: T019→T021→T029 (ping);
  T020→T028 (servers). `version`/`config` (T026/T027) have no helper deps.
- T030 (registration) after the four command implementations exist.

### Parallel Opportunities

- Phase 1: T002 ∥ T001-followup.
- Phase 2 tests: T003 ∥ T004.
- US1 tests: T009 ∥ T010 ∥ T011 (same new file — write as one batch or split; independent of other phases).
- US2 helper tests T016 ∥ T017 ∥ T018; helper impls T020 ∥ (T019→T021).
- US2 command tests T022 ∥ T023 ∥ T024 ∥ T025 (separate files).
- US2 command impls T026 ∥ T027 ∥ T028 ∥ T029 (separate files; T028/T029 gated on their helpers).
- Polish: T031 ∥ T032.

---

## Parallel Example: User Story 2 command implementations

```bash
# After foundational + helpers (T019–T021) are done, these touch separate files:
Task: "Implement version command in internal/cli/version.go"         # T026
Task: "Implement config command in internal/cli/config.go"           # T027
Task: "Implement servers command in internal/cli/servers.go"         # T028 (needs T020)
Task: "Implement ping command in internal/cli/ping.go"               # T029 (needs T021)
# Then integrate:
Task: "Register the four commands in internal/cli/command.go"        # T030
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1: Setup.
2. Phase 2: Foundational (registry + dispatch) — CRITICAL.
3. Phase 3: US1 help system.
4. **STOP and VALIDATE**: run US1 independent test (help offline, unknown → exit 2).
5. Ship the help system as the MVP.

### Incremental Delivery

1. Setup + Foundational → registry ready.
2. US1 → discoverable help → validate → ship (MVP).
3. US2 → four commands → validate each → ship.
4. Polish → docs + quality gates + quickstart.

---

## Notes

- [P] = different files, no dependency on an incomplete task.
- Reuse the existing fakes in `internal/cli/app_test.go` (`fakeLocator`,
  `fakeResolver`, `fakeConsent`, `fakeRunner`) and add a fake `ndt7.Client` for
  the latency path; no real network in unit tests (Constitution III).
- `help`, `version`, and `config` MUST stay offline (FR-008) — assert no
  locator/resolver/consent calls in their tests.
- Commit after each task or logical group; keep changes minimal and idiomatic.

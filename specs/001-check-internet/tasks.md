---
description: "Task list for Check Internet & Nearest Provider"
---

# Tasks: Check Internet & Nearest Provider

**Input**: Design documents from `/specs/001-check-internet/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: INCLUDED — Velox Constitution Principle III (Test-First) is
NON-NEGOTIABLE. Unit tests are written before implementation and must fail first;
they stay network-free via interfaces/fakes/httptest. Integration tests live
under `test/integration` behind `//go:build integration`.

**Organization**: Grouped by user story for independent implementation/testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no incomplete dependencies)
- **[Story]**: US1=Full speed test, US2=Consent gate, US3=Nearest provider, US4=Vuln scan
- Paths follow plan.md: `cmd/velox/`, `internal/<pkg>/`, `test/integration/`, `scripts/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and tooling

- [X] T001 Create project structure (`cmd/velox/`, `internal/{cli,speedtest,locate,ndt7,geo,consent,config,version}/`, `test/integration/`, `scripts/`) per plan.md
- [X] T002 Initialize Go module in `go.mod` (Go 1.26.4) and add dependency `github.com/m-lab/ndt7-client-go`; run `go mod tidy`
- [X] T003 [P] Add `.golangci.yml` enabling gofmt, govet, errcheck, staticcheck (Constitution Principle I)
- [X] T004 [P] Add `Makefile` with `build`, `test`, `lint`, `security`, `vuln` targets
- [X] T005 [P] Add `.github/workflows/ci.yml` skeleton running `gofmt -l`, `go vet`, `golangci-lint`, `go test -race`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure all user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T006 [P] Implement `internal/version` and wire `--version`/`--help` in `internal/cli` — MUST work offline, no consent prompt (FR-016)
- [X] T007 Implement root context with `signal.NotifyContext` (SIGINT/SIGTERM) in `cmd/velox/main.go`; restore terminal, propagate cancellation (FR-011, R8)
- [X] T008 Implement CLI router in `internal/cli/router.go`: `flag` parsing, command dispatch, exit codes 0/1/2 per `contracts/cli-interface.md`
- [X] T009 [P] Write failing unit tests for `internal/config` (missing file, corrupt JSON, atomic write) in `internal/config/config_test.go`
- [X] T010 Implement `internal/config`: `os.UserConfigDir()/velox/config.json` resolution, atomic load/save, `schemaVersion`, corrupt→defaults (R5, `contracts/consent.schema.json`)
- [X] T011 [P] Implement output renderer in `internal/cli/render.go`: human + JSON, results→stdout, diagnostics→stderr (Constitution Principle II)

**Checkpoint**: Foundation ready — user stories can begin

---

## Phase 3: User Story 1 - Run a Full Internet Speed Test (Priority: P1) 🎯 MVP

**Goal**: `velox --check-internet` measures connectivity, latency, download, and
upload against a server and reports them (human + `--json`), exit 0/1.

**Independent Test**: Run `velox --check-internet` on a live connection → online
with latency + download + upload + server; disable network → offline, exit 1.
Uses a default/fallback server, so it works without consent or geolocation.

### Tests for User Story 1 (write first, must fail) ⚠️

- [X] T012 [P] [US1] Unit test `internal/locate` against an httptest-mocked Locate v2 response → `[]Server` in `internal/locate/locate_test.go`
- [X] T013 [P] [US1] Unit test `internal/ndt7` wrapper with a fake `Client` (download/upload `Throughput`, ctx cancellation) in `internal/ndt7/ndt7_test.go`
- [X] T014 [P] [US1] Unit test `internal/speedtest` orchestrator with fake locate+ndt7 (phase order, offline path, mid-run phase failure) in `internal/speedtest/run_test.go`
- [X] T015 [P] [US1] Unit test `MeasurementResult` JSON encoding conforms to `contracts/result.schema.json` in `internal/speedtest/result_test.go`

### Implementation for User Story 1

- [X] T016 [P] [US1] Implement `internal/locate` client (GET M-Lab Locate v2 via `net/http`, parse → `Server`, extract site code from `machine`) in `internal/locate/locate.go` (`contracts/external-apis.md` §1)
- [X] T017 [P] [US1] Implement `internal/ndt7`: `Client` interface + m-lab wrapper (`Download`/`Upload` over `wss://`, honor ctx) in `internal/ndt7/ndt7.go` (FR-018, §3)
- [X] T018 [US1] Implement `MeasurementResult`, `Phase`, `PhaseOutcome` types in `internal/speedtest/result.go` (data-model.md)
- [X] T019 [US1] Implement orchestrator in `internal/speedtest/run.go`: connectivity→latency→download→upload, per-phase + overall timeout, offline + phase-failure handling (depends T016–T018; FR-001/002/011)
- [X] T020 [US1] Wire `--check-internet`, `--server`, `--timeout`, `--json`, `-v` in `internal/cli`; render result; set exit codes (cli-interface.md)
- [X] T021 [US1] Add default/fallback server constant in `internal/speedtest` so US1 runs without geo/consent (FR-007)
- [X] T022 [P] [US1] Integration test `test/integration/speedtest_test.go` (`//go:build integration`) — live `--check-internet` against M-Lab

**Checkpoint**: MVP — full speed test works standalone

---

## Phase 4: User Story 2 - Approve Location Before Provider Lookup (Priority: P1)

**Goal**: Explicit consent gate before any location lookup; remembered across
runs; revocable; non-interactive defaults to declined.

**Independent Test**: First location-dependent run prompts; decline → no lookup,
fallback used; approve → persisted, no re-prompt next run; `consent --reset`
re-prompts; piped/no-TTY → declined without prompting.

### Tests for User Story 2 (write first, must fail) ⚠️

- [X] T023 [P] [US2] Unit test `internal/consent` state machine (unset→granted/denied, reset, persistence) in `internal/consent/consent_test.go`
- [X] T024 [P] [US2] Unit test TTY detection + non-interactive default = declined (not persisted) in `internal/consent/tty_test.go`

### Implementation for User Story 2

- [X] T025 [US2] Implement `internal/consent`: `ConsentRecord`, load/save via `internal/config`, decision transitions in `internal/consent/consent.go` (FR-005/006/007, data-model.md)
- [X] T026 [US2] Implement interactive prompt (approve/decline; states public IP is sent to geo lookup) writing to stderr in `internal/consent/prompt.go` (FR-004, SC-002)
- [X] T027 [US2] Implement TTY detection in `internal/consent/tty.go`; non-interactive → treat as declined for the run, do not persist (FR-007, SC-004)
- [X] T028 [US2] Implement `velox consent` subcommand (`--status`/`--reset`/`--grant`/`--deny`) in `internal/cli` (cli-interface.md, FR-006)
- [X] T029 [US2] Gate the speed-test flow to request consent when `unset` + TTY before any geo step in `internal/cli` (FR-004)
- [X] T030 [P] [US2] Integration test `test/integration/consent_test.go` — consent persists across runs; reset re-prompts (quickstart S4/S5)

**Checkpoint**: US1 + US2 both work independently

---

## Phase 5: User Story 3 - Select Nearest Provider by Location (Priority: P2)

**Goal**: With consent granted, resolve city-level location from IP, rank Locate
candidates by distance, pick nearest, display server + distance; degrade to
fallback on geo failure.

**Independent Test**: Consent granted → nearest named server + plausible
distance; multiple candidates → closest chosen; geo failure → fallback, no crash.

### Tests for User Story 3 (write first, must fail) ⚠️

- [X] T031 [P] [US3] Unit test `internal/geo` IP-geo parsing (httptest) + haversine correctness in `internal/geo/geo_test.go`
- [X] T032 [P] [US3] Unit test nearest-server selection (rank by haversine; fallback when estimate nil) in `internal/geo/select_test.go`

### Implementation for User Story 3

- [X] T033 [P] [US3] Implement `internal/geo` IP-geo lookup over **HTTPS** (default `https://ipwho.is/`, configurable endpoint, ctx) → `LocationEstimate`, transient/no-persist, in `internal/geo/geo.go` (R4, external-apis.md §2, S1)
- [X] T034 [P] [US3] Implement haversine distance in `internal/geo/distance.go`
- [X] T034a [P] [US3] Add bundled M-Lab site→lat/lon table (embedded JSON or `internal/locate/sites.go`) and resolve `Server.lat/lon` by site code; missing site → candidate excluded from ranking (U1, external-apis.md §1)
- [X] T035 [US3] Implement nearest-server ranking in `internal/geo/select.go`: rank Locate candidates by geographic distance, set `distanceKm`, exclude coord-less candidates, degrade to fallback when geo/estimate unavailable (FR-008/009, SC-005; depends T033/T034/T034a + T016)
- [X] T036 [US3] In `internal/cli`: invoke geo only when consent granted; show server name + distance; `distanceKm` null when declined (FR-007/008, cli-interface.md; depends US2)
- [X] T037 [P] [US3] Integration test `test/integration/nearest_test.go` — granted → distance shown; declined → null distance (quickstart S6)

**Checkpoint**: US1 + US2 + US3 all independently functional

---

## Phase 6: User Story 4 - Scan the Project for Vulnerabilities (Priority: P2)

**Goal**: Maintainer/CI command runs gosec + govulncheck, exits non-zero at HIGH+
severity, reports MEDIUM/LOW as warnings.

**Independent Test**: Run `make security` → findings with severity + location;
exit non-zero iff a HIGH/CRITICAL finding exists; clean repo → exit 0.

### Tests for User Story 4 (write first, must fail) ⚠️

- [X] T038 [P] [US4] Test `scripts/security.sh` exit-code behavior against mocked findings / threshold in `scripts/security_test.sh` (or `internal/...` test harness)

### Implementation for User Story 4

- [X] T039 [US4] Implement `scripts/security.sh`: run `gosec -severity high` + `govulncheck`, non-zero on HIGH+, MEDIUM/LOW as warnings, configurable threshold (FR-012/013, R6, Q5)
- [X] T040 [P] [US4] Wire `make security` / `make vuln` to `scripts/security.sh` in `Makefile`
- [X] T041 [US4] Add security job to `.github/workflows/ci.yml` gating PRs at HIGH+ (FR-013)

**Checkpoint**: All user stories functional

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Project-wide deliverables and validation

- [X] T042 [P] Create `CHANGELOG.md` (Keep a Changelog format) with initial entry (FR-015)
- [X] T043 [P] Add open-source `LICENSE` and state it in repo (FR-014)
- [X] T044 [P] Write `README.md`: usage, flags, metric units + sampling windows, consent/privacy note (SC-009, Constitution Workflow doc rule)
- [X] T045 [P] Implement `scripts/build.sh`: cross-compile static binaries (CGO_ENABLED=0; linux/macos/windows × amd64/arm64) (plan)
- [X] T046 Run full gate locally: `gofmt -l .` (empty), `go vet ./...`, `golangci-lint run`, `go test -race ./...`
- [X] T047 Execute `quickstart.md` scenarios S1–S9 and confirm expected outcomes
- [X] T048 [P] Final lint/refactor pass for small-functions + wrapped-errors (Constitution Principle I)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (P1)**: no dependencies
- **Foundational (P2)**: depends on Setup — BLOCKS all user stories
- **US1 (P3)**: depends on Foundational
- **US2 (P4)**: depends on Foundational (independent of US1)
- **US3 (P5)**: depends on Foundational + US1 (locate/server) + US2 (consent gate)
- **US4 (P6)**: depends on Setup only (independent of US1–US3)
- **Polish (P7)**: depends on all targeted stories

### User Story Dependencies

- **US1 (P1)**: standalone — uses fallback server, no consent/geo
- **US2 (P1)**: standalone — consent store + commands, no measurement needed
- **US3 (P2)**: needs US1's locate candidates + US2's consent decision
- **US4 (P2)**: fully independent (tooling/CI)

### Within Each User Story

- Tests written and FAILING before implementation (Principle III)
- Models before services; services before CLI wiring
- Core before integration

### Parallel Opportunities

- Setup: T003/T004/T005 parallel
- Foundational: T006, T009, T011 parallel (T007/T008/T010 sequential touch shared wiring)
- US1 tests T012–T015 parallel; impl T016/T017 parallel (then T018→T019→T020)
- US2 tests T023/T024 parallel
- US3 tests T031/T032 parallel; impl T033/T034/T034a parallel (then T035→T036)
- US4 T040 parallel with others; T039/T041 sequential
- US1, US2, US4 can be developed in parallel by different people after Foundational

---

## Parallel Example: User Story 1

```bash
# Tests first (all parallel, must fail):
Task: "Unit test internal/locate in internal/locate/locate_test.go"
Task: "Unit test internal/ndt7 wrapper in internal/ndt7/ndt7_test.go"
Task: "Unit test internal/speedtest orchestrator in internal/speedtest/run_test.go"
Task: "Unit test MeasurementResult JSON in internal/speedtest/result_test.go"

# Then parallel implementation of independent packages:
Task: "Implement internal/locate client in internal/locate/locate.go"
Task: "Implement internal/ndt7 wrapper in internal/ndt7/ndt7.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1: Setup
2. Phase 2: Foundational (CRITICAL)
3. Phase 3: US1 — full speed test against fallback server
4. **STOP and VALIDATE**: quickstart S1–S3
5. Demo: `velox --check-internet`

### Incremental Delivery

1. Setup + Foundational → foundation ready
2. US1 → MVP (speed test)
3. US2 → consent gate (privacy)
4. US3 → nearest-provider + distance
5. US4 → security gate (can land any time after Setup)
6. Polish → CHANGELOG, LICENSE, README, cross-compile, full gate

### Parallel Team Strategy

After Foundational: Dev A → US1, Dev B → US2, Dev C → US4; US3 follows US1+US2.

---

## Notes

- [P] = different files, no incomplete dependencies
- Tests MUST fail before implementation (Constitution Principle III)
- Unit tests stay network-free (interfaces + httptest/fakes); integration behind `//go:build integration`
- All network ops context-bounded + cancellable (Principle IV)
- Keep gofmt/vet/golangci-lint/-race green per task; commit after each task or logical group

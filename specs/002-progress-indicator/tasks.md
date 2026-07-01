---
description: "Task list for Loading Indicator for velox --check-internet"
---

# Tasks: Loading Indicator for `velox --check-internet`

**Input**: Design documents from `/specs/002-progress-indicator/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: INCLUDED — Velox Constitution Principle III (Test-First) is
NON-NEGOTIABLE. Unit tests are written before implementation and must fail first;
they stay network-free and TTY-free via interfaces, fakes, and buffer writers.

**Organization**: Grouped by user story for independent implementation/testing.
This feature extends the existing `001-check-internet` codebase; no project
scaffolding is recreated.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no incomplete dependencies)
- **[Story]**: US1=Live progress, US2=Clean output/`--no-progress`, US3=Clean cancel
- Paths follow plan.md: `internal/progress/`, `internal/speedtest/`, `internal/cli/`, `cmd/velox/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish baseline and create the new package location.

- [X] T001 Establish a green baseline before changes: run `go build ./... && go test ./...` and confirm it passes
- [X] T002 [P] Create the new package file `internal/progress/indicator.go` with the `package progress` doc comment only (compiles as an empty package)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The compile-level seam every story depends on. No user-visible
behavior changes here — the build stays green and results/exit codes are identical.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T003 Add the `Reporter` interface (`Phase(p Phase)`), a nil-safe `report(r Reporter, p Phase)` helper, and a trailing `reporter Reporter` parameter on `Runner.Run` (NO phase emissions yet) in `internal/speedtest/run.go`
- [X] T004 Update every `Run` call site to the new signature, passing `nil` so behavior is unchanged: the `SpeedRunner` interface and call in `internal/cli/app.go` (lines ~30 and ~123), the three `r.Run(...)` calls in `internal/speedtest/run_test.go`, the `fakeRunner.Run` method in `internal/cli/app_test.go`, and `runner.Run(...)` in `test/integration/speedtest_test.go`
- [X] T005 [P] Add a `StderrF *os.File` field to `App` in `internal/cli/app.go` (for TTY detection of stderr) and set `StderrF: os.Stderr` in `NewApp`

**Checkpoint**: Foundation ready — `go build ./... && go test ./...` still green; user stories can begin.

---

## Phase 3: User Story 1 - See Live Progress During the Speed Test (Priority: P1) 🎯 MVP

**Goal**: On an interactive terminal, `velox --check-internet` shows an animated
spinner with the current phase (selecting server → checking connectivity →
measuring download → measuring upload) and elapsed seconds, then clears it before
printing results.

**Independent Test**: Run `velox --check-internet` in a real terminal on a live
connection → a spinner appears within ~1s, reflects each phase, and is gone (clean
line) when results print.

### Tests for User Story 1 (write first, must FAIL) ⚠️

- [X] T006 [P] [US1] Failing unit test for the pure `frame(glyph, label, elapsed)` renderer (label + whole-second formatting at 0s/5s/65s) in `internal/progress/indicator_test.go`
- [X] T007 [P] [US1] Failing unit test with a recording `Reporter` asserting phase order — healthy run: `connectivity→download→upload`; offline run: `connectivity` only — plus a nil-reporter safety case, in `internal/speedtest/run_test.go`
- [X] T008 [P] [US1] Failing unit test for `phaseLabel(Phase)` mapping (Connectivity/Download/Upload → labels) in `internal/cli/progress_test.go`

### Implementation for User Story 1

- [X] T009 [P] [US1] Implement the indicator core in `internal/progress/indicator.go`: `New(w io.Writer, f *os.File)` with `enabled` gating (`os.ModeCharDevice`; nil file, `TERM` empty/`dumb`, or `NO_COLOR` set ⇒ disabled), the `frame()` helper, a `Start()` ticker goroutine (~100ms, ≥4 repaints/sec per SC-001) repainting `"\r"+frame+"\x1b[K"` under a `sync.Mutex`, `SetPhase(label)`, and a basic `Stop()` (close done channel, clear line). Disabled ⇒ every method writes nothing
- [X] T010 [P] [US1] Add phase emissions in `internal/speedtest/run.go`: `report(reporter, PhaseConnectivity)` before the dial, `PhaseDownload` before the download call, `PhaseUpload` before the upload call (makes T007 pass)
- [X] T011 [P] [US1] Implement `phaseLabel(Phase) string` and the `phaseReporter` adapter (implements `speedtest.Reporter`, forwards to `Indicator.SetPhase`) in `internal/cli/progress.go`
- [X] T012 [US1] Add the indicator factory `newIndicator()` (TTY-gated via `StderrF`) in `internal/cli/progress.go` and wire `runRoot` in `internal/cli/app.go`: construct + `Start()`, call `SetPhase("selecting server…")` before `selectServer`, pass a `phaseReporter` into `Runner.Run` (replacing the `nil` from T004), and `Stop()` immediately before rendering results
- [X] T013 [US1] Run `go test ./internal/progress/... ./internal/speedtest/... ./internal/cli/...` (T006–T008 now pass) and smoke-test quickstart Scenario 1 in a real terminal

**Checkpoint**: `velox --check-internet` shows a working per-phase spinner on a TTY and clears it before results — independently testable and demoable.

---

## Phase 4: User Story 2 - Clean Output for Scripts, Pipes, and JSON (Priority: P1)

**Goal**: The indicator never corrupts captured output — suppressed entirely on
non-TTY/pipe/CI/`TERM=dumb`/`NO_COLOR`, `--json` stdout stays exactly one valid
JSON document, `--verbose` suppresses the animation (text diagnostics narrate
progress), and a `--no-progress` flag disables it even on a TTY.

**Independent Test**: Pipe the command to a file / `--json | jq` → no spinner
frames or escape sequences; stdout is one valid JSON document. `--no-progress` and
`--verbose` each silence the indicator on a real terminal.

### Tests for User Story 2 (write first, must FAIL) ⚠️

- [X] T014 [P] [US2] Failing unit test: a disabled indicator (nil `*os.File`, separately `NO_COLOR=1`, and separately `TERM=dumb`) writes zero bytes across `Start`→`SetPhase`→`Stop`, in `internal/progress/indicator_test.go`
- [X] T015 [P] [US2] Failing test: `--check-internet --json` yields exactly one JSON document on stdout (parses) with no escape sequences, and `--check-internet --no-progress` parses and exits 0, in `internal/cli/app_test.go`
- [X] T015a [P] [US2] Failing unit test for the pure gating decision `indicatorEnabled(isTTY, noProgress, verbose)` — true only when `isTTY && !noProgress && !verbose` (covers FR-009/FR-011) in `internal/cli/progress_test.go`

### Implementation for User Story 2

- [X] T016 [US2] Add the `--no-progress` bool flag in `runRoot` (`internal/cli/app.go`), introduce the pure helper `indicatorEnabled(isTTY, noProgress, verbose) bool` in `internal/cli/progress.go`, and gate the factory `newIndicator(...)` on it so the indicator is disabled under `--no-progress` **or** `--verbose` (FR-009/FR-011)
- [X] T017 [P] [US2] Document `--no-progress` under the `--check-internet` FLAGS block in `internal/cli/help.go` (must stay offline, no consent — FR-016)
- [X] T018 [US2] Run tests (T014–T015a pass) and validate quickstart Scenarios 2–5 (pipe, `--json`, `NO_COLOR`, `--no-progress`) plus a `--verbose` run — confirm stdout purity, full suppression, and that verbose text is not garbled

**Checkpoint**: Piped/`--json`/CI output is provably clean; `--no-progress` and `--verbose` both suppress the animation. Combined with US1 this is the shippable P1 feature.

---

## Phase 5: User Story 3 - Cancel Cleanly Without Breaking the Terminal (Priority: P2)

**Goal**: Ctrl-C (and any failure/early return) stops the animation and restores
the terminal — cursor visible, no partial line.

**Independent Test**: Start `velox --check-internet`, press Ctrl-C mid-run → the
animation stops, the cursor is visible, and the next prompt is on a clean line.

### Tests for User Story 3 (write first, must FAIL) ⚠️

- [X] T019 [P] [US3] Failing unit test using a force-enabled indicator writing to a buffer: after `Start`→`Stop` the output ends with the line-clear + cursor-restore (`\x1b[?25h`) and leaves no dangling escape; calling `Stop` twice does not panic and restores at most once, in `internal/progress/indicator_test.go`

### Implementation for User Story 3

- [X] T020 [US3] Harden `internal/progress/indicator.go`: hide the cursor (`\x1b[?25l`) on `Start`, restore it (`\x1b[?25h`) and clear the line on `Stop`, and make `Stop` idempotent via `sync.Once` (FR-006/FR-007)
- [X] T021 [P] [US3] Add a deferred `Stop()` safety net in `runRoot` (`internal/cli/app.go`) so the cancellation / early-return paths restore the terminal even when the explicit pre-render `Stop()` is skipped
- [X] T022 [US3] Run `go test -race ./internal/progress/...` (clean) and smoke-test quickstart Scenario 6 (Ctrl-C) and Scenario 7 (offline fails fast)

**Checkpoint**: Cancellation and failure paths leave a clean terminal; `-race` clean.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and full quality gates across all stories.

- [X] T023 [P] Update `README.md`: document the loading indicator behavior and the `--no-progress` flag
- [X] T024 [P] Update `CHANGELOG.md`: add an "Added" entry for the loading indicator and `--no-progress` flag
- [X] T025 Run the full quality gate: `gofmt -l .` (empty), `go vet ./...`, `golangci-lint run`, `go test -race ./...`, `govulncheck ./...`
- [X] T026 Execute `specs/002-progress-indicator/quickstart.md` Scenarios 1–7 end to end and confirm expected outcomes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately.
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories (adds the `Reporter` seam + `StderrF`).
- **User Stories (Phase 3–5)**: All depend on Foundational.
  - US1 (P1) and US2 (P1) form the shippable feature; US2 builds on US1's indicator.
  - US3 (P2) builds on the indicator from US1.
- **Polish (Phase 6)**: Depends on the desired stories being complete.

### User Story Dependencies

- **US1 (P1)**: Starts after Foundational. Self-contained (indicator + reporter + wiring).
- **US2 (P1)**: Reuses US1's `Indicator`/factory; adds `--no-progress`, the `--verbose` suppression gate (FR-009), and the formal clean-output tests. The non-TTY/`TERM=dumb`/`NO_COLOR` suppression path is implemented in US1's gating (T009) and *verified* here.
- **US3 (P2)**: Extends US1's `Stop()` with cursor restore + idempotency and the deferred safety net.

### Within Each User Story

- Tests (T006–T008, T014–T015a, T019) are written and MUST fail before implementation.
- `progress` core before CLI wiring; runner emission before/independent of CLI wiring.
- Story complete before moving to the next priority.

### Parallel Opportunities

- T002 (Setup) and T005 (Foundational) touch new/independent code.
- US1 tests T006/T007/T008 (three different files) run in parallel.
- US1 impl T009/T010/T011 (progress vs speedtest vs cli files) run in parallel; T012 waits on T011.
- US2 tests T014/T015/T015a (three different files) run in parallel; T017 (help.go) parallel with T016.
- Polish T023/T024 (README vs CHANGELOG) run in parallel.

---

## Parallel Example: User Story 1

```bash
# Write the three failing tests together (different files):
Task: "frame() renderer test in internal/progress/indicator_test.go"
Task: "Reporter phase-order test in internal/speedtest/run_test.go"
Task: "phaseLabel() mapping test in internal/cli/progress_test.go"

# Then implement the independent pieces together (different files):
Task: "Indicator core in internal/progress/indicator.go"
Task: "Phase emissions in internal/speedtest/run.go"
Task: "phaseLabel + phaseReporter in internal/cli/progress.go"
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Complete Phase 1 (Setup) and Phase 2 (Foundational).
2. Complete Phase 3 (US1) → a working per-phase spinner on a TTY, cleared before results.
3. **STOP and VALIDATE**: quickstart Scenario 1. Non-TTY runs are already suppressed via the indicator's gating.

### Incremental Delivery

1. Setup + Foundational → seam ready.
2. US1 → visible indicator (functional MVP) → validate.
3. US2 → `--no-progress` + provable clean output for pipes/CI/`--json` (ship US1+US2 together as the P1 feature).
4. US3 → robust cancellation/terminal restoration.
5. Polish → docs + full quality gate + quickstart.

---

## Notes

- [P] tasks = different files, no incomplete dependencies.
- [Story] label maps each task to its user story for traceability.
- Presentation-only: no task changes `MeasurementResult`, units, or exit codes (FR-010).
- No new third-party dependency is added (stdlib-only `internal/progress`; Constitution Principle V).
- Verify tests fail before implementing; keep `go test -race ./...` clean.
- Commit after each task or logical group.

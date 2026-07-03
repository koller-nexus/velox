# Tasks: Nearest Internet Provider

**Input**: Design documents from `/specs/004-nearest-provider/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/, quickstart.md

**Tests**: Included. The velox constitution makes test-first development non-negotiable.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the static asset directory and embed wiring so the provider catalog ships inside the binary.

- [x] T001 Create `embed/` directory and add `embed/providers.json` with an initial catalog of Brazilian ISPs and POPs
- [x] T002 Add `//go:embed` wiring in `internal/provider/catalog.go` to load the embedded JSON at runtime
- [x] T003 Update `Makefile` so `embed/` contents are included in the build and distribution targets

**Checkpoint**: The binary can embed and read the provider catalog without runtime network calls.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core provider-selection infrastructure that MUST be complete before any user story can be implemented.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T004 [P] Define `internal/provider/model.go` with `Provider`, `POP`, `Catalog`, and `NearestResult` types
- [x] T005 [P] Implement `internal/provider/catalog.go` to parse `embed/providers.json`, validate coordinates, and fail gracefully on missing/malformed data
- [x] T006 [P] Implement `internal/provider/select.go` with `FindNearest(location geo.LocationEstimate, catalog Catalog) (NearestResult, bool)` using `geo.HaversineKm` and deterministic tie-breaking
- [x] T007 Write `internal/provider/catalog_test.go` with table-driven tests for loading, validation, and graceful fallback
- [x] T008 Write `internal/provider/select_test.go` with table-driven tests for nearest selection, tie-breaking, and "no nearby POP" handling

**Checkpoint**: Foundation ready — provider catalog loads, selection logic is tested, and user story implementation can now begin.

---

## Phase 3: User Story 1 - See the nearest internet provider in the report (Priority: P1) 🎯 MVP

**Goal**: When location consent is granted, velox reports the nearest provider's name, POP, and distance alongside the usual speed-test output.

**Independent Test**: With consent granted, run `./bin/velox --check-internet` and confirm a `Nearest provider:` line appears. With consent denied, confirm the line is absent and the test still completes.

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation.**

- [x] T009 [P] [US1] Add unit test in `internal/cli/render_test.go` asserting `renderHuman` includes the nearest-provider line when present
- [x] T010 [P] [US1] Add unit test in `internal/speedtest/result_test.go` asserting `MeasurementResult` JSON includes `nearestProvider` when set
- [x] T011 [P] [US1] Add unit test in `internal/provider/select_test.go` verifying deterministic tie-breaking for equidistant POPs

### Implementation for User Story 1

- [x] T012 [US1] Extend `internal/speedtest/result.go` to add `NearestProvider *provider.NearestResult` to `MeasurementResult` with correct JSON tags
- [x] T013 [US1] Update `internal/cli/render.go` `renderHuman()` to print the nearest-provider line when `res.NearestProvider` is non-nil
- [x] T014 [US1] Update `internal/cli/app.go` `selectServer()` to resolve the nearest provider (when consent is granted) and attach it to the result path
- [x] T015 [US1] Cache the resolved location in `internal/cli/app.go` so it is determined at most once per invocation (FR-011)
- [x] T016 [US1] Ensure nearest-provider output respects existing colour/emoji/no-colour conventions in `internal/cli/render.go`

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently.

---

## Phase 4: User Story 3 - Graceful fallback when location or provider data is unavailable (Priority: P1)

**Goal**: Any failure in the nearest-provider path falls back to existing default server selection; the test always completes and never prompts in non-interactive mode.

**Independent Test**: Reset/deny consent, or simulate a location lookup failure, or remove the catalog, then run `./bin/velox --check-internet`; confirm the test completes with normal results, no prompt, and exit code `0` on success.

### Tests for User Story 3

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation.**

- [x] T017 [P] [US3] Add test in `internal/cli/app_test.go` asserting no location lookup and no nearest-provider line when consent is denied
- [x] T018 [P] [US3] Add test in `internal/cli/app_test.go` asserting non-interactive mode (pipe) does not prompt and does not lookup location
- [x] T019 [P] [US3] Add test in `internal/provider/catalog_test.go` asserting a missing/malformed catalog returns a graceful "not available" state instead of an error
- [x] T020 [P] [US3] Add test in `internal/cli/app_test.go` asserting a location-lookup timeout falls back to default server selection and completes

### Implementation for User Story 3

- [x] T021 [US3] Update `internal/cli/app.go` `selectServer()` to skip location resolution and provider selection when consent is not `granted`
- [x] T022 [US3] Update `internal/cli/app.go` to detect non-interactive runs and skip any consent prompt / location lookup when consent is unset
- [x] T023 [US3] Wrap the provider-catalog load in `internal/provider/catalog.go` so a missing/malformed catalog is treated as "unavailable" rather than fatal
- [x] T024 [US3] Add fallback notice (stderr, verbose-only) in `internal/cli/app.go` when nearest-provider information cannot be determined
- [x] T025 [US3] Ensure `MeasurementResult` leaves `NearestProvider` nil in all fallback paths so JSON output remains valid

**Checkpoint**: User Stories 1 and 3 should now both work independently; the feature never breaks an otherwise-working test.

---

## Phase 5: User Story 2 - Direct the test toward the nearest provider (Priority: P2)

**Goal**: An opt-in `--nearest-provider` flag (and optional config preference) biases server selection toward the M-Lab server nearest to the closest ISP POP.

**Independent Test**: Run `./bin/velox --check-internet --nearest-provider` with consent granted and confirm the report identifies the selected server; run without the flag and confirm default server selection is unchanged.

### Tests for User Story 2

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation.**

- [x] T026 [P] [US2] Add test in `internal/cli/app_test.go` asserting `--nearest-provider` selects the M-Lab server closest to the nearest provider POP
- [x] T027 [P] [US2] Add test in `internal/cli/app_test.go` asserting the absence of `--nearest-provider` preserves existing default server selection
- [x] T028 [P] [US2] Add test in `internal/cli/app_test.go` asserting `--nearest-provider` falls back gracefully when no nearest provider is available

### Implementation for User Story 2

- [x] T029 [US2] Add `--nearest-provider` bool flag to root flag set in `internal/cli/app.go` `runRoot()`
- [x] T030 [US2] Add optional `NearestProvider bool` field to `internal/config/config.go` so the preference can be persisted
- [x] T031 [US2] Implement `internal/provider/target.go` with `SelectServerForProvider(candidates []locate.Server, nearest provider.NearestResult) (locate.Server, *float64)` choosing the M-Lab server nearest to the provider POP
- [x] T032 [US2] Wire `--nearest-provider` / config preference into `internal/cli/app.go` `selectServer()` so it overrides default selection only when opted in
- [x] T033 [US2] Update `internal/cli/render.go` to indicate when the displayed server was chosen via the nearest-provider path
- [x] T034 [US2] Update `internal/cli/help.go` to document the new flag

**Checkpoint**: All user stories should now be independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Quality gates, documentation, and cross-story alignment.

- [x] T035 [P] Update `README.md` with the new `--nearest-provider` flag and sample output
- [x] T036 [P] Run `make all` and fix any `gofmt`, `go vet`, `golangci-lint`, or race-test failures
- [x] T037 [P] Run `make security` and address any HIGH+ findings
- [x] T038 [P] Execute the scenarios in `specs/004-nearest-provider/quickstart.md` against a local build
- [x] T039 Add regression tests in `internal/provider/` for any bugs found during quickstart validation
- [x] T040 Review all new code against the velox constitution and justify any deviations in the PR description

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories.
- **User Stories (Phase 3–5)**: All depend on Foundational phase completion.
  - User Story 1 (P1) should be implemented first; it is the MVP.
  - User Story 3 (P1) can be implemented in parallel with or immediately after US1.
  - User Story 2 (P2) builds on US1/US3 and should be implemented last.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2). No dependencies on other stories. This is the MVP.
- **User Story 3 (P1)**: Can start after Foundational (Phase 2). May share paths with US1 but is independently testable through fallback scenarios.
- **User Story 2 (P2)**: Depends on US1 (nearest-provider metadata logic) and US3 (fallback behaviour) being stable.

### Within Each User Story

- Tests MUST be written and FAIL before implementation.
- Models before services (`internal/provider/model.go` before `select.go`).
- Services before CLI integration.
- Core implementation before help/docs updates.
- Story complete before moving to next priority.

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel.
- All Foundational tasks marked [P] can run in parallel (within Phase 2).
- Once Foundational phase completes, US1 and US3 can be worked on in parallel.
- US2 should start after US1 and US3 are stable.
- All Polish tasks marked [P] can run in parallel after implementation is complete.

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Add unit test in internal/cli/render_test.go asserting renderHuman includes nearest-provider line"
Task: "Add unit test in internal/speedtest/result_test.go asserting MeasurementResult JSON includes nearestProvider"
Task: "Add unit test in internal/provider/select_test.go verifying deterministic tie-breaking"

# Launch all models/result changes together:
Task: "Extend internal/speedtest/result.go to add NearestProvider field"
Task: "Update internal/cli/render.go renderHuman() to print nearest-provider line"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. **STOP and VALIDATE**: Run `./bin/velox --check-internet` and verify the nearest-provider line.
5. Deploy/demo if ready.

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready.
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!).
3. Add User Story 3 → Test independently → Deploy/Demo.
4. Add User Story 2 → Test independently → Deploy/Demo.
5. Each story adds value without breaking previous stories.

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together.
2. Once Foundational is done:
   - Developer A: User Story 1 (reporting)
   - Developer B: User Story 3 (fallback / non-interactive)
3. After US1/US3 are stable:
   - Developer C: User Story 2 (opt-in server selection)
4. Stories complete and integrate independently.

---

## Notes

- [P] tasks = different files, no dependencies.
- [Story] label maps task to specific user story for traceability.
- Each user story should be independently completable and testable.
- Verify tests fail before implementing.
- Commit after each task or logical group.
- Stop at any checkpoint to validate a story independently.
- Avoid: vague tasks, same-file conflicts, cross-story dependencies that break independence.

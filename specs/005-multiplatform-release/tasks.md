# Tasks: Multiplatform Release Distribution

**Input**: Design documents from `specs/005-multiplatform-release/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/cli-release-interface.md](contracts/cli-release-interface.md), [quickstart.md](quickstart.md)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. No new Go runtime code is required; most tasks are configuration, scripts, and documentation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare the repository for release automation tooling.

- [x] T001 Create `.goreleaser.yaml` at repository root with project metadata, build matrix, archives, checksums, changelog, and release config
- [x] T002 [P] Create `.github/workflows/release.yml` to run GoReleaser on tags matching `v*`
- [x] T003 [P] Verify the `HOMEBREW_TAP_TOKEN` and `GITHUB_TOKEN` secrets are documented or configured for the repository

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Align existing build tooling with the release pipeline before any user story can be completed.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T004 Update `scripts/build.sh` to exclude `windows/arm64` and align target list with `.goreleaser.yaml`
- [x] T005 Update `Makefile` `LDFLAGS` to use `github.com/koller-nexus/velox/internal/version` consistently (already correct; verify no drift)
- [x] T006 Confirm `internal/version/version.go` exposes `Version`, `Commit`, and `Date` variables injectable via `-ldflags`
- [ ] T007 Add `goreleaser` to developer tooling notes in README or Makefile help (optional but recommended)

**Checkpoint**: Foundation ready — `make cross` produces the same five targets that GoReleaser will use.

---

## Phase 3: User Story 1 - One-Command Install on macOS and Linux (Priority: P1) 🎯 MVP

**Goal**: Provide a POSIX shell installer that auto-detects OS/arch, downloads the correct archive, verifies its SHA256 checksum, and installs the binary.

**Independent Test**: In a fresh macOS or Linux environment without Go, run `curl -fsSL https://raw.githubusercontent.com/koller-nexus/velox/main/scripts/install.sh | sh` and confirm `velox version` works.

### Implementation for User Story 1

- [x] T008 [US1] Create `scripts/install.sh` with OS/arch detection, latest-version resolution, checksum verification, and `/usr/local/bin` install with `~/.local/bin` fallback
- [ ] T009 [US1] Add `shellcheck` validation step for `scripts/install.sh` in CI or Makefile
- [x] T010 [US1] Test `install.sh` locally with `VERSION` pointing to an existing release (or mock server) on macOS and Linux

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently.

---

## Phase 4: User Story 2 - Manual Download from Release Page (Priority: P1)

**Goal**: Ensure GitHub Releases publishes platform-specific archives and a checksums file so users can download and verify manually.

**Independent Test**: A user can download the Windows `.zip` (or any Unix `.tar.gz`), extract it, and run `velox version` without extra dependencies.

### Implementation for User Story 2

- [x] T011 [US2] Verify `.goreleaser.yaml` archives block names files as `velox_{{.Version}}_{{.Os}}_{{.Arch}}` and includes `README.md` and `LICENSE`
- [x] T012 [US2] Verify `.goreleaser.yaml` checksum block generates `checksums.txt` with SHA256 algorithm
- [ ] T013 [US2] Run `goreleaser release --snapshot --clean` locally and confirm five archives plus `checksums.txt` are produced in `dist/`
- [x] T014 [US2] Verify the Windows archive contains `velox.exe` and no external runtime dependencies

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently.

---

## Phase 5: User Story 3 - Version and Build Metadata for Support (Priority: P2)

**Goal**: Ensure every official release binary embeds and exposes the release version, commit, and build date.

**Independent Test**: Running `velox version` on any published binary shows the same version as the release tag plus commit and date.

### Implementation for User Story 3

- [x] T015 [US3] Verify `.goreleaser.yaml` `ldflags` inject `Version`, `Commit`, and `Date` into `github.com/koller-nexus/velox/internal/version`
- [x] T016 [US3] Add or update `internal/cli/version_test.go` to assert `version.String()` includes non-empty version and commit when built with ldflags
- [x] T017 [US3] Run a snapshot release, extract one archive, and confirm `./velox version` prints the injected metadata

**Checkpoint**: User Story 3 is independently verifiable.

---

## Phase 6: User Story 4 - Package Manager Install on macOS via Homebrew (Priority: P2)

**Goal**: Allow macOS users to install and upgrade velox through a Homebrew tap.

**Independent Test**: On macOS with Homebrew, `brew tap koller-nexus/tap && brew install velox` installs the binary and `velox version` works.

### Implementation for User Story 4

- [x] T018 [US4] Verify `.goreleaser.yaml` `brews` block points to `koller-nexus/homebrew-tap` and uses `HOMEBREW_TAP_TOKEN`
- [ ] T019 [US4] Create the `koller-nexus/homebrew-tap` repository if it does not exist
- [ ] T020 [US4] Confirm the generated formula installs `velox` and runs `velox version` as a test

**Checkpoint**: User Story 4 is ready once a real release publishes the first formula.

---

## Phase 7: User Story 5 - Fallback Install for Go Users (Priority: P3)

**Goal**: Document and verify the universal `go install` fallback.

**Independent Test**: A system with Go installed can run `go install github.com/koller-nexus/velox/cmd/velox@latest` and obtain a working binary.

### Implementation for User Story 5

- [x] T021 [US5] Verify `go install github.com/koller-nexus/velox/cmd/velox@latest` works and places a `velox` binary in `$GOPATH/bin` or `$GOBIN`
- [x] T022 [US5] Confirm the installed binary reports a development version when built this way (no tag ldflags)

**Checkpoint**: User Story 5 is documented and verified.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and final validation across all stories.

- [x] T023 [P] Update `README.md` with a dedicated "Installation" section covering macOS/Linux shell install, Homebrew, Windows manual download, and `go install` fallback
- [x] T024 [P] Update `README.md` `Install` snippet to reference pre-built binaries and the install script
- [ ] T025 Run all steps in `quickstart.md` (snapshot release, checksum verification, install script test, version metadata check)
- [x] T026 [P] Run `make all` to ensure formatting, vet, lint, race tests, and security checks still pass
- [ ] T027 Push a test tag `v0.0.0-test` to validate the GitHub Actions release workflow end-to-end, then clean up the tag and draft release

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion — blocks all user stories.
- **User Stories (Phase 3–7)**: All depend on Foundational phase completion.
  - US1 and US2 are P1 and can proceed in parallel after foundation.
  - US3, US4, US5 are lower priority and can be worked in parallel once their prerequisites are ready.
- **Polish (Phase 8)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational. No dependencies on other stories.
- **User Story 2 (P1)**: Can start after Foundational. No dependencies on other stories.
- **User Story 3 (P2)**: Depends on US2 release artifacts existing (snapshot sufficient). Can run in parallel with US1/US2 implementation.
- **User Story 4 (P2)**: Depends on US2 release artifacts and the Homebrew tap repository existing.
- **User Story 5 (P3)**: Only requires the module path to be valid and publicly fetchable; no release artifact dependency.

### Within Each User Story

- Configuration/script tasks before validation tasks.
- Validation tasks must run after the artifact they test exists.

### Parallel Opportunities

- T001, T002, T003 in Phase 1 can run in parallel.
- T004, T005, T006 in Phase 2 can run in parallel.
- US1 and US2 can be implemented in parallel after Phase 2.
- US3, US4, US5 can be implemented in parallel once US2 snapshot artifacts are available.
- T023 and T024 in Phase 8 can run in parallel.

---

## Parallel Example: User Story 1 + User Story 2

```bash
# After Phase 2 foundation:
Task: "Create scripts/install.sh ..."          # US1
Task: "Verify .goreleaser.yaml archives ..."   # US2
Task: "Verify .goreleaser.yaml checksums ..."  # US2
```

---

## Implementation Strategy

### MVP First (User Stories 1 and 2)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1 (shell installer).
4. Complete Phase 4: User Story 2 (release artifacts + checksums).
5. **STOP and VALIDATE**: Run a snapshot release and test the install script.
6. Deploy/demo if ready.

### Incremental Delivery

1. Setup + Foundational → foundation ready.
2. US1 + US2 → shell installer + GitHub releases (MVP!).
3. US3 → version metadata injection and test.
4. US4 → Homebrew tap.
5. US5 → `go install` fallback documentation.
6. Polish → README updates and final validation.

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together.
2. Once Foundational is done:
   - Developer A: US1 (shell installer)
   - Developer B: US2 (GoReleaser artifacts + checksums)
3. US3/US4/US5 can be picked up in parallel after snapshot artifacts exist.

---

## Notes

- [P] tasks = different files, no dependencies.
- [Story] label maps task to specific user story for traceability.
- Each user story should be independently completable and testable.
- Commit after each task or logical group.
- Stop at any checkpoint to validate a story independently.
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence.

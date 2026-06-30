<!--
SYNC IMPACT REPORT
==================
Version change: TEMPLATE (unversioned) → 1.0.0
Bump rationale: Initial ratification — placeholders replaced with concrete principles.

Modified principles:
  - [PRINCIPLE_1_NAME] → I. Clean Code & Idiomatic Go
  - [PRINCIPLE_2_NAME] → II. CLI-First Interface
  - [PRINCIPLE_3_NAME] → III. Test-First (NON-NEGOTIABLE)
  - [PRINCIPLE_4_NAME] → IV. Measurement Accuracy & Reliability
  - [PRINCIPLE_5_NAME] → V. Simplicity & Minimal Dependencies

Added sections:
  - Technical Standards & Constraints (was [SECTION_2_NAME])
  - Development Workflow & Quality Gates (was [SECTION_3_NAME])

Removed sections: none

Templates requiring updates:
  ✅ .specify/templates/plan-template.md     — Constitution Check gate is generic; aligns, no edit needed
  ✅ .specify/templates/spec-template.md     — no constitution-coupled sections; aligns
  ✅ .specify/templates/tasks-template.md     — task categories cover test-first + CLI; aligns
  ✅ .specify/templates/checklist-template.md — generic; aligns

Follow-up TODOs: none
-->

# Velox Constitution

Velox is a command-line tool that measures internet connection speed (download,
upload, latency/jitter), in the spirit of `speedtest-cli`. This constitution
defines the non-negotiable engineering principles that govern its development.

## Core Principles

### I. Clean Code & Idiomatic Go

Code MUST be readable, idiomatic, and self-explanatory before it is clever.

- All code MUST pass `gofmt` and `go vet` with zero diffs/warnings; CI rejects
  unformatted code.
- Names MUST reveal intent: exported identifiers documented per Go doc
  conventions, no single-letter names outside short loop/receiver scopes.
- Functions MUST do one thing; a function exceeding ~50 lines or 3 levels of
  nesting MUST be justified in review or refactored.
- No dead code, no commented-out blocks, no TODO without an issue reference.
- Errors MUST be wrapped with context (`fmt.Errorf("...: %w", err)`) and never
  silently discarded.

Rationale: A measurement tool is only trusted if its logic is auditable. Clean,
idiomatic code keeps the math and network paths verifiable by any reader.

### II. CLI-First Interface

Every capability MUST be reachable from the command line following the
text-in/text-out contract.

- Input via flags/args/stdin; results to stdout; diagnostics and errors to
  stderr.
- MUST support both human-readable output (default) and machine-readable
  `--json` output for scripting.
- MUST honor standard signals: `Ctrl-C` (SIGINT) cancels in-flight measurement
  cleanly and restores the terminal.
- Exit codes MUST be meaningful: `0` success, non-zero on failure, distinct
  codes for network vs. usage errors.
- `--help` and `--version` MUST always work without network access.

Rationale: The CLI is the product surface. Predictable, composable I/O lets
velox slot into scripts, CI, and monitoring pipelines.

### III. Test-First (NON-NEGOTIABLE)

Tests are written before implementation. No production code merges without
covering tests.

- Red-Green-Refactor: write a failing test, make it pass, then refactor.
- Network, timing, and server selection MUST be testable behind interfaces;
  unit tests MUST NOT make real network calls (use `httptest`/fakes).
- Every bug fix MUST add a regression test reproducing the bug first.
- Table-driven tests preferred for unit coverage; integration tests cover the
  full measurement flow against a controlled test server.

Rationale: Speed measurements are easy to get subtly wrong. Tests are the only
defense against silent accuracy regressions.

### IV. Measurement Accuracy & Reliability

Reported numbers MUST be correct, reproducible, and clearly defined.

- Units, sampling windows, and the definition of each metric (download, upload,
  latency, jitter) MUST be documented and consistent across runs.
- All network operations MUST use `context.Context` with explicit timeouts and
  be cancellable; no unbounded reads or hangs.
- Transient failures (DNS, connection reset, partial transfer) MUST be handled
  gracefully — retry where sound, degrade with a clear message otherwise.
- Concurrency used to saturate the link MUST be race-free (`go test -race`
  clean) and must not double-count bytes.

Rationale: An inaccurate or flaky speed test is worse than none. Correctness and
graceful failure are the core value proposition.

### V. Simplicity & Minimal Dependencies

Prefer the standard library and the simplest design that works (YAGNI).

- The standard library is the default; a third-party dependency MUST be
  justified by clear value and reviewed for maintenance and license.
- Each added dependency MUST pass `govulncheck`; no known-vulnerable modules.
- No speculative abstraction — interfaces and config knobs are added when a
  second concrete use exists, not before.
- The binary MUST stay self-contained: a single static binary with no required
  runtime beyond the OS.

Rationale: A small, dependency-light tool is fast to install, easy to audit,
and cheap to maintain.

## Technical Standards & Constraints

- Language: Go (latest stable minor release; pinned via `go.mod`).
- Tooling gates (all MUST pass in CI): `gofmt -l` (empty), `go vet`,
  `golangci-lint`, `go test -race ./...`, `govulncheck`.
- Distribution: single static binary; cross-compiled for Linux, macOS, Windows
  (amd64 + arm64).
- Performance: startup-to-first-output overhead MUST stay negligible; the tool
  MUST NOT add measurable load that skews its own measurement.
- Logging: human output is the default; verbose/debug detail gated behind
  `-v`/`--verbose` and written to stderr, never polluting stdout results.

## Development Workflow & Quality Gates

- Branching: feature branches; no direct commits to the default branch.
- Commits: Conventional Commits format; atomic, scoped, with a clear "why".
- Code review: every change requires review; the reviewer MUST verify
  compliance with the Core Principles above.
- CI is the gate: a PR cannot merge unless all tooling gates (see Technical
  Standards) are green.
- Documentation: user-facing flag/behavior changes MUST update `--help` text
  and the README in the same PR.

## Governance

This constitution supersedes all other development practices for velox. When a
practice conflicts with a principle here, the principle wins.

- Amendments MUST be proposed via PR, documented with rationale, and approved by
  a project maintainer before merge.
- Versioning of this document follows semantic versioning:
  - MAJOR: backward-incompatible removal or redefinition of a principle.
  - MINOR: a new principle or materially expanded guidance is added.
  - PATCH: clarifications, wording, or non-semantic refinements.
- All PRs and reviews MUST verify compliance; any deviation MUST be justified in
  the PR description and, if accepted, captured as an amendment.
- Complexity MUST be justified — when in doubt, the simpler design that upholds
  these principles wins.

**Version**: 1.0.0 | **Ratified**: 2026-06-30 | **Last Amended**: 2026-06-30

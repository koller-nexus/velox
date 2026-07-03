# Implementation Plan: Multiplatform Release Distribution

**Branch**: `[005-multiplatform-release]` | **Date**: 2026-07-03 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/005-multiplatform-release/spec.md`

## Summary

Distribute the velox CLI as pre-built static binaries for macOS, Linux, and Windows without requiring users to install Go. A release is triggered by pushing a semantic-version tag and produces platform archives, SHA256 checksums, a Homebrew tap, and a shell installer. The existing `version` subcommand already exposes build metadata; the release pipeline must inject the correct values.

## Technical Context

**Language/Version**: Go 1.26.4 (pinned in `go.mod`)

**Primary Dependencies**: GoReleaser v2 (release automation), GitHub Actions (CI/CD), GitHub Releases (artifact hosting), optional Homebrew tap repository.

**Storage**: N/A

**Testing**: `go test -race ./...`, GoReleaser dry-run (`goreleaser release --snapshot --clean`), shellcheck for `install.sh`.

**Target Platform**: Linux amd64/arm64, macOS amd64/arm64, Windows amd64

**Project Type**: CLI tool

**Performance Goals**: Startup overhead of the installed binary must remain negligible; the distribution mechanism must not affect runtime measurement accuracy.

**Constraints**: Static binaries only (`CGO_ENABLED=0`); no new Go runtime dependencies; `--version` and `version` must work offline; macOS notarization is out of scope.

**Scale/Scope**: One repository, one CLI binary, public GitHub releases.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Assessment | Notes |
|-----------|------------|-------|
| I. Clean Code & Idiomatic Go | ✅ Pass | No Go code changes required beyond aligning ldflags with existing package path. |
| II. CLI-First Interface | ✅ Pass | `--version` and `version` already offline; release metadata injection preserves this contract. |
| III. Test-First | ✅ Pass | Existing `version_test.go` covers subcommand parity; new tests target the release script/pipeline (shellcheck, snapshot). |
| IV. Measurement Accuracy & Reliability | ✅ Pass | Distribution artifacts do not touch measurement logic. |
| V. Simplicity & Minimal Dependencies | ✅ Pass | GoReleaser is a build-time/dev tool, not a runtime dependency; binary remains self-contained. |

## Project Structure

### Documentation (this feature)

```text
specs/005-multiplatform-release/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
.
├── cmd/velox/main.go               # entrypoint (no change)
├── internal/version/version.go     # build metadata variables (already exist)
├── internal/cli/version.go         # version subcommand (already exists)
├── scripts/
│   ├── build.sh                    # existing cross-compile helper
│   └── install.sh                  # NEW: curl|sh installer
├── .github/workflows/
│   ├── ci.yml                      # existing
│   └── release.yml                 # NEW: GoReleaser on tag
├── .goreleaser.yaml                # NEW: release configuration
├── Makefile                        # existing (align VERSION/LDFLAGS)
└── README.md                       # UPDATE: installation section
```

**Structure Decision**: Keep the single-project layout. Add release automation files under `.github/workflows/`, `scripts/`, and root config. The existing `internal/version` package already owns build metadata; reuse it instead of introducing new variables in `main`.

## Complexity Tracking

No constitution violations required justification.

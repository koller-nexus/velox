# Implementation Plan: Nearest Internet Provider

**Branch**: `004-nearest-provider` | **Date**: 2026-07-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/004-nearest-provider/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add a "nearest provider" capability to velox. When the user has granted
location-consent, velox will detect the user's approximate city-level location,
compare it against a bundled catalog of ISP points of presence (POPs), and
report the closest provider/POP in the speed-test output. An opt-in flag (or
saved config) will bias server selection toward the M-Lab test server nearest to
that provider's POP, with graceful fallback to the existing registry-based
selection whenever location, consent, or the catalog is unavailable.

## Technical Context

**Language/Version**: Go 1.26.4 (pinned by `go.mod`).

**Primary Dependencies**: Standard library plus the existing
`github.com/m-lab/ndt7-client-go` runtime dependency. No new external runtime
libraries required.

**Storage**: Local JSON config file (`config.json`) for the consent decision and
optional feature preferences; provider catalog bundled as an embedded static file
(e.g. JSON) shipped inside the binary.

**Testing**: `go test -race ./...`; table-driven unit tests with fakes for
network and geolocation; existing `httptest` patterns for HTTP clients.

**Target Platform**: Cross-compiled static CLI binary for Linux, macOS, Windows
(amd64 + arm64), run interactively and non-interactively.

**Project Type**: CLI tool / command-line speed test.

**Performance Goals**: Nearest-provider computation (location lookup, catalog
scan, distance ranking) must add less than ~1 second of wall time and must not
block or delay the actual ndt7 measurement.

**Constraints**:

- No precise/GPS location; city-level IP-based geolocation only.
- No persistence of location data to disk; in-memory per invocation.
- No new consent surface; reuse existing `consent` decision.
- No third-party runtime services beyond the existing location endpoint lookup.
- Must remain accurate and honest: no estimated latency shown for providers;
  "target nearest provider" resolves to the nearest M-Lab server to the provider
  POP, not an ISP-operated endpoint.

**Scale/Scope**: Single-user CLI invocation; provider catalog focuses initially
on major Brazilian ISPs (Claro, Vivo, Oi, TIM, Algar) and can expand per
release.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Clean Code & Idiomatic Go | ✅ pass | New code follows existing package conventions (`geo`, `locate`, `config`, `cli`). Interfaces are small and injected for tests. |
| II. CLI-First Interface | ✅ pass | Feature is exposed via flag (`--nearest-provider`) and/or config; JSON output extended; no TUI or interactive prompts beyond existing consent flow. |
| III. Test-First (NON-NEGOTIABLE) | ✅ pass | Plan includes fakes for geo resolver and provider catalog; unit tests before implementation; race-clean requirement preserved. |
| IV. Measurement Accuracy & Reliability | ✅ pass | No estimated latency shown; nearest-provider metadata is clearly distance-based; fallback keeps existing accurate measurement path. |
| V. Simplicity & Minimal Dependencies | ✅ pass | Catalog embedded as static JSON; Haversine already in `geo`; no new external module required. |

## Project Structure

### Documentation (this feature)

```text
specs/004-nearest-provider/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)

```text
internal/
├── cli/
│   ├── app.go              # extend selectServer() and flag parsing
│   ├── render.go           # extend human/JSON output with nearest-provider
│   └── ...existing tests
├── config/
│   ├── config.go           # optional: nearest-provider preference
│   └── config_test.go
├── consent/
│   └── ...reuse as-is
├── geo/
│   ├── geo.go              # existing LocationEstimate/Resolver
│   ├── distance.go         # existing HaversineKm
│   ├── select.go           # existing server ranking
│   └── ...provider ranking added here or in new provider package
├── locate/
│   ├── locate.go           # Server type reused
│   └── sites.go            # existing metro coordinate table
├── provider/
│   ├── catalog.go          # ISP POP catalog + loading
│   ├── catalog_test.go
│   ├── model.go            # Provider, POP, NearestResult types
│   ├── select.go           # nearest provider selection logic
│   └── select_test.go
├── speedtest/
│   ├── result.go           # extend MeasurementResult with nearest provider
│   └── run.go              # unchanged measurement path
└── ndt7/
    └── ...reuse as-is

embed/
└── providers.json          # bundled ISP POP catalog

test/
└── integration/            # opt-in live tests, if needed
```

**Structure Decision**: Add a new `internal/provider` package for the ISP POP
catalog and nearest-provider selection. Extend `internal/cli` for flag/config
handling and rendering. Reuse `internal/geo` for Haversine distance and
`internal/locate` for server coordinates. Embed the catalog under a top-level
`embed/` directory so it is included in the static binary.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations. All gates pass without requiring complexity trade-offs.

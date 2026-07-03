# Research Notes: Nearest Internet Provider

**Feature**: Nearest Internet Provider  
**Date**: 2026-07-01  
**Spec**: [spec.md](spec.md)  
**Plan**: [plan.md](plan.md)

## Resolved Decisions

### 1. What does "target the nearest provider" mean?

- **Decision**: The feature biases server selection toward the M-Lab ndt7 test
  server that is geographically closest to the nearest provider's POP. It does
  not connect to an ISP-operated endpoint.
- **Rationale**: velox's core measurement is built on M-Lab ndt7. ISP POPs are
  not velox-compatible measurement targets. Routing the user to the M-Lab server
  nearest to that POP preserves measurement accuracy and the existing trusted
  infrastructure while still honoring the user's intent of proximity-based
  selection.
- **Alternatives considered**:
  - Connect directly to ISP POPs → rejected: no compatible test endpoints,
    would break the measurement model and accuracy guarantees.
  - Metadata-only (show provider, don't change server) → rejected for the
    opt-in target mode, but kept as the default behaviour: P1 still shows the
    provider without changing the server.

### 2. Should a latency estimate be shown per provider?

- **Decision**: No. The nearest-provider metadata shows provider name, POP, and
  distance only.
- **Rationale**: The constitution treats measurement accuracy as
  non-negotiable. Showing an "estimated" latency derived from distance would be
  an unmeasured, potentially misleading number. Real latency only appears from
  the actual ndt7 measurement.
- **Alternatives considered**:
  - Show approximate latency with an "~est." label → rejected: still presents a
    fabricated number next to precise measurements.
  - Show real latency only when measured → acceptable in principle, but the
    provider metadata is emitted before the measurement, so it is simpler and
    clearer to omit it from the provider block.

### 3. Provider catalog format and delivery

- **Decision**: A static JSON file bundled into the binary with `//go:embed`,
  defining providers and their POPs (provider name, POP label, latitude,
  longitude, optional country/region tags).
- **Rationale**: Keeps velox a single self-contained binary, adds no runtime
  network dependency for the catalog, and aligns with Principle V (minimal
  dependencies). The initial catalog focuses on the major Brazilian ISPs named
  in the request; it can be expanded per release.
- **Alternatives considered**:
  - External service / live API → rejected: adds runtime dependency, privacy
    risk, and offline failure mode.
  - MaxMind-style local database → rejected: overkill for a small, curated POP
    list; heavier and licensed differently.

### 4. Distance calculation and tie-breaking

- **Decision**: Reuse the existing `geo.HaversineKm` for great-circle distance.
  For ties (two POPs at the same distance within float precision), break ties
  deterministically by provider name + POP label lexical order.
- **Rationale**: Haversine is already implemented, tested, and accurate enough
  for city-level proximity. Deterministic tie-breaking ensures reproducible
  output and tests.
- **Alternatives considered**:
  - Vincenty or more precise geodesic formula → rejected: no meaningful
    improvement at city-level granularity; adds complexity.
  - Random tie-break → rejected: breaks reproducibility and tests.

### 5. Consent and privacy

- **Decision**: Reuse the existing `consent` package and stored decision. The
  location lookup is performed only when consent is `granted`; otherwise the
  feature silently falls back. Location data is held in memory only and never
  persisted.
- **Rationale**: No new consent surface, minimal privacy impact, and consistent
  with the existing privacy model documented in the README.
- **Alternatives considered**:
  - Separate consent for provider catalog → rejected: unnecessary friction;
    the existing location consent already covers IP-based geolocation.

### 6. Integration with server selection

- **Decision**: The nearest-provider logic produces a "preferred test target"
  (the M-Lab server nearest to the provider POP). This target is used only when
  the user opts in via `--nearest-provider` or an equivalent saved config
  preference. The default path remains unchanged.
- **Rationale**: Preserves existing behaviour for all users unless they
  explicitly request the new mode; aligns with User Story 2 and FR-008.

### 7. Fallback strategy

- **Decision**: Any failure in the nearest-provider path (no consent, location
  lookup error, missing catalog, no provider match) falls back to the existing
  `geo.SelectNearest` behaviour using registry-ordered M-Lab candidates. The
  test always completes; availability notices are non-fatal and written to
  stderr (or omitted in non-interactive mode).
- **Rationale**: Reliability is a core principle; a measurement tool must not
  fail because of an auxiliary metadata feature.

## Open Questions for Implementation

None. All scope-defining questions were resolved in this research phase.

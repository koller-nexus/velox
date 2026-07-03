# Feature Specification: Nearest Internet Provider

**Feature Branch**: `004-nearest-provider`

**Created**: 2026-07-01

**Status**: Draft

**Input**: User description: "Adicionar provedor de internet mais próximo à localização do usuário — detectar a localização (via IP), determinar o provedor/POP mais próximo geograficamente, exibir no relatório e opcionalmente direcionar o teste ao servidor mais próximo, com fallback robusto e respeito à privacidade."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See the nearest internet provider in the report (Priority: P1)

A user runs a velox speed test and, alongside the usual latency/download/upload
figures, sees which internet provider point-of-presence (POP) is geographically
closest to them, together with the approximate distance. This turns a bare
speed number into a locally meaningful result ("your closest provider POP is
Vivo, São Paulo — Centro, ~2 km away").

**Why this priority**: This is the core value of the feature and is deliverable
on its own. Showing the nearest provider as report context requires location
detection, the provider catalog, the distance calculation, and the display — a
complete, independently useful slice — without changing how or where the actual
measurement runs.

**Independent Test**: With location consent granted, run a normal test and
confirm the report includes a "Nearest provider" line with provider name, POP
city, and distance; run again with consent denied and confirm the line is
absent and the test still completes.

**Acceptance Scenarios**:

1. **Given** location consent is granted and the user's approximate location is
   determined, **When** the user runs a speed test, **Then** the report shows
   the nearest provider's name, its POP location, and the distance in kilometres.
2. **Given** consent is granted, **When** the machine-readable output is
   requested, **Then** the nearest-provider details appear as structured fields
   in that output.
3. **Given** two provider POPs are equidistant from the user, **When** the
   nearest provider is selected, **Then** the selection is deterministic (same
   input always yields the same provider) rather than arbitrary between runs.

---

### User Story 2 - Direct the test toward the nearest provider (Priority: P2)

A user wants the speed test to target the server associated with their nearest
provider rather than the default selection, and asks for this explicitly (e.g. a
`--nearest-provider` option) or via saved configuration.

**Why this priority**: Valuable but secondary — it depends on P1's location and
proximity logic already working, and most users are served well by seeing the
information without changing server selection. It is a distinct, testable
enhancement layered on top of P1.

**Independent Test**: Run the test requesting the nearest-provider target and
confirm the report states which provider/server was used for the measurement;
confirm that omitting the option preserves today's default server-selection
behaviour unchanged.

**Acceptance Scenarios**:

1. **Given** consent is granted and a nearest provider is found, **When** the
   user requests the nearest-provider target, **Then** the test runs against the
   server associated with that nearest provider and the report identifies it.
2. **Given** the user does not request the nearest-provider target, **When** the
   test runs, **Then** server selection behaves exactly as it does today.

---

### User Story 3 - Graceful fallback when location or provider data is unavailable (Priority: P1)

A user without location consent, running in a non-interactive environment, or
hitting a location-lookup failure still gets a complete speed test. velox
silently degrades to its existing default/fastest server selection and clearly
signals that nearest-provider information is unavailable, without errors or
prompts that would break scripts.

**Why this priority**: Reliability is non-negotiable for a measurement tool. The
feature must never turn a working test into a failed or blocked one. This is a
P1 guardrail that ships together with P1.

**Independent Test**: Disable/deny location, or simulate a location-lookup
failure, then run the test; confirm it completes with normal results, no
nearest-provider line, no prompt in non-interactive mode, and the usual exit
code.

**Acceptance Scenarios**:

1. **Given** location consent is denied or unset in a non-interactive run,
   **When** the user runs a test, **Then** no location lookup occurs, the test
   completes normally, and the report omits nearest-provider details.
2. **Given** consent is granted but the location lookup fails or times out,
   **When** the user runs a test, **Then** the test still completes against a
   fallback server and the report notes that the nearest provider could not be
   determined.
3. **Given** the provider catalog is missing or unreadable, **When** the user
   runs a test, **Then** the test still completes and nearest-provider details
   are omitted with a clear, non-fatal notice.

---

### Edge Cases

- **No nearby POP**: the user's location has no provider POP within a reasonable
  radius — the system reports "no nearby provider found" rather than showing a
  misleadingly distant match.
- **Non-interactive execution** (CI, cron, pipe, redirected output): no consent
  prompt is shown and no location lookup is attempted; the test proceeds on the
  default server.
- **Consent revoked between runs**: a previously shown nearest provider stops
  appearing once consent is withdrawn.
- **Repeated runs in one session**: the location is determined at most once per
  invocation and reused, avoiding duplicate lookups.
- **Nearest-provider target requested but unavailable** (no consent, no match,
  or lookup failed): the request degrades to the fallback server with a clear
  notice instead of failing.
- **Machine-readable output** must remain valid and parseable whether or not
  nearest-provider details are present.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST determine the user's approximate (city-level)
  geographic location before identifying a nearest provider.
- **FR-002**: System MUST NOT perform any location lookup unless the user's
  existing location-lookup consent has been granted, and MUST reuse the current
  consent mechanism rather than introducing a separate one.
- **FR-003**: System MUST maintain a catalog of internet providers and their
  points of presence, each with a geographic location, covering the major
  providers relevant to the user base.
- **FR-004**: System MUST compute the geographic distance between the user's
  location and each catalogued provider POP.
- **FR-005**: System MUST select the provider POP with the smallest distance and
  determine the corresponding provider, resolving ties deterministically.
- **FR-006**: System MUST present the selected provider's name, POP location, and
  distance (in kilometres) in the default human-readable report.
- **FR-007**: System MUST include the nearest-provider details in the
  machine-readable output when that output is requested, and MUST keep that
  output valid whether or not the details are present.
- **FR-008**: System MUST provide a way for the user to request that the test
  target the nearest provider's associated server (explicit option and/or saved
  configuration), while leaving default behaviour unchanged when not requested.
- **FR-009**: System MUST fall back to its existing default/fastest server
  selection whenever location cannot be determined, consent is not granted, no
  provider match is found, or the provider catalog is unavailable — the test
  MUST still complete successfully.
- **FR-010**: System MUST clearly indicate when nearest-provider information is
  unavailable and why (e.g. no consent, lookup failed, no nearby provider),
  without emitting a fatal error for these expected conditions.
- **FR-011**: System MUST determine the user's location at most once per
  invocation and reuse it for all nearest-provider work in that run.
- **FR-012**: System MUST NOT persist the user's precise location to disk or
  transmit it to third parties beyond what the consented location lookup already
  requires; location data is used in memory for the current run only.
- **FR-013**: System MUST behave non-interactively when output is not a terminal
  or consent is unset: no prompt, no lookup, silent fallback.
- **FR-014**: System MUST present nearest-provider output in a style consistent
  with the rest of velox's report (same formatting conventions, colour/emoji
  behaviour, and stdout/stderr discipline as existing output).
- **FR-015**: System MUST NOT display any estimated or inferred latency figure
  for the nearest provider; only measured latency from the actual test may be
  shown, and the nearest-provider metadata itself reports name, POP, and distance
  only.
- **FR-016**: "Directing the test to the nearest provider" MUST resolve to
  selecting the compatible test server (from velox's existing server set) that is
  geographically nearest to the nearest provider's POP, while keeping the
  measurement path on trusted infrastructure.

### Key Entities *(include if data involved)*

- **User Location**: the user's approximate position for this run — coordinates
  and city/country, at city-level granularity; transient, held in memory only.
- **Provider**: an internet provider that can be reported as "nearest" — has a
  display name and one or more points of presence.
- **Point of Presence (POP)**: a physical location associated with a provider —
  has a geographic position and a human-readable place label (e.g. city/area);
  the unit against which distance is measured.
- **Nearest-Provider Result**: the outcome shown to the user — the chosen
  provider, its POP place label, and the distance to the user; optionally an
  associated test target derived from the existing compatible server set.
- **Provider Catalog**: the maintained collection of providers and their POPs
  that the selection searches over.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: When location consent is granted and a location is resolved, 100%
  of speed-test runs display a nearest-provider result (name, POP place,
  distance) or an explicit "unavailable" notice — never a blank or malformed
  line.
- **SC-002**: For a user with a known city-level location, the provider selected
  as "nearest" is the one whose POP has the smallest true geographic distance in
  100% of verification cases (correct minimum, correct distance within rounding).
- **SC-003**: 100% of runs still produce a complete speed-test result when
  location is unavailable, consent is denied, the lookup fails, or the catalog is
  missing — no failed or hung runs attributable to this feature.
- **SC-004**: Nearest-provider work adds no user-perceptible delay to the test in
  the common case — the added time before results appear stays within a small
  fraction of total test time (target: under ~1 second added, and never blocking
  the measurement itself).
- **SC-005**: In non-interactive runs, 0% of executions produce a location prompt
  or attempt a location lookup without prior consent.
- **SC-006**: Machine-readable output remains valid and parseable in 100% of
  runs, with nearest-provider fields present when available and cleanly absent
  when not.

## Assumptions

- The existing velox location-consent gate (granted / denied / unset, stored
  locally) is reused as-is; this feature adds no new consent surface.
- Location is derived from the user's public IP at city-level granularity; GPS or
  device-level precise location is out of scope.
- The provider catalog is bundled with the tool and focuses initially on the
  major providers relevant to the primary user base (e.g. the large Brazilian
  ISPs named in the request), and can be expanded over time; it is not a live
  external service in the first version.
- Geographic distance is a straight-line ("as the crow flies") surface distance
  between two coordinates; it approximates proximity, not network path length.
- Displaying the nearest provider is informational by default; changing the
  actual test target is opt-in and does not alter default server selection.
- The feature does not add a required third-party runtime dependency or network
  service beyond the location lookup that velox already performs under consent.

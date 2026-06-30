# Feature Specification: Check Internet & Nearest Provider

**Feature Branch**: `001-check-internet`

**Created**: 2026-06-30

**Status**: Draft

**Input**: User description: "use go 1.26.4, tenha comandos para verificar vulnerabilidade, projeto é open source, crie changelog.md, use gosec, ao rodar velox --check-internet, vai chegar a internet, precis chegar o provedor mais proximo de acordo com sua localizacao, precisa tem um aprovar para solicitar a localizacao"

## Clarifications

### Session 2026-06-30

- Q: Scope of `velox --check-internet` — connectivity-only or full speed test? → A: Full speed test (connectivity + latency + download + upload throughput against the nearest server).
- Q: Where does the candidate test-server list come from? → A: An open measurement registry (M-Lab); no proprietary/Ookla network and no API key.
- Q: How is latency/connectivity probed (privilege impact)? → A: TCP/HTTP(S) round-trips only; MUST run as a normal user with no elevated OS privileges.
- Q: How is approximate location resolved after consent? → A: IP-based geolocation (coarse, city-level); consent covers sending the user's public IP to the lookup.
- Q: Default severity at which the vulnerability scan fails CI? → A: High and above (HIGH/CRITICAL fail; MEDIUM/LOW reported as warnings); threshold stays configurable.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run a Full Internet Speed Test (Priority: P1)

A user runs `velox --check-internet` to measure their connection end to end: it
confirms connectivity, then reports round-trip latency, jitter, and download and
upload throughput against the selected test server. This is the core slice — the
user gets a complete, trustworthy picture of their internet speed in one command.

**Why this priority**: A full speed measurement is the core promise of the tool.
It delivers standalone value and is the foundation every other feature builds on.

**Independent Test**: Run `velox --check-internet` on a machine with a live
connection and confirm it reports online plus latency, download, and upload
figures; disable the network and confirm it reports offline with a non-zero exit
code.

**Acceptance Scenarios**:

1. **Given** a working connection, **When** the user runs `velox --check-internet`, **Then** the tool reports the connection is online and shows latency, download throughput, and upload throughput, exiting with code 0.
2. **Given** no connection, **When** the user runs `velox --check-internet`, **Then** the tool reports offline with a clear message on stderr and a non-zero exit code.
3. **Given** the user adds a machine-readable flag, **When** the test completes, **Then** the same result (latency, download, upload) is emitted as structured JSON suitable for scripting.
4. **Given** the connection drops partway through the test, **When** a phase fails, **Then** the tool reports which phase failed with a clear message and a non-zero exit code rather than reporting a misleading number.

---

### User Story 2 - Approve Location Before Provider Lookup (Priority: P1)

Before velox uses the user's location to find the nearest provider/test server,
it MUST ask for explicit consent. The user can approve or decline. If declined,
velox still works (falls back to a default/manual server) but never resolves
location. The consent decision is remembered so the user is not re-prompted on
every run.

**Why this priority**: Requesting location is privacy-sensitive. Consent is a
hard, non-negotiable gate and is required for User Story 3 to function lawfully
and transparently. It ships alongside P1 because it must exist before any
location lookup ever happens.

**Independent Test**: Run the command for the first time and confirm a consent
prompt appears; decline and confirm no location lookup occurs and the run
continues with a fallback; approve and confirm the choice is persisted and not
re-asked next run.

**Acceptance Scenarios**:

1. **Given** no prior consent decision, **When** a command needs location, **Then** velox presents a clear approve/decline prompt explaining what data is used and why.
2. **Given** the user declines, **When** the command proceeds, **Then** velox performs no location lookup and uses a fallback server, stating that location is disabled.
3. **Given** the user approves, **When** the command runs again later, **Then** velox does not re-prompt and proceeds using the remembered consent.
4. **Given** a prior decision exists, **When** the user runs a reset/revoke action, **Then** the stored consent is cleared and the next location-dependent run prompts again.
5. **Given** a non-interactive session (no TTY), **When** consent is unknown, **Then** velox defaults to "declined" and continues with the fallback rather than blocking.

---

### User Story 3 - Select Nearest Provider by Location (Priority: P2)

With consent granted, velox determines the user's approximate location and
selects the nearest available provider/test server, so connectivity checks and
future speed tests run against the closest, most representative endpoint. The
chosen server (name and distance) is shown to the user.

**Why this priority**: Picking the nearest server materially improves the
accuracy and relevance of results, but it depends on P1 (a working check) and P2
(consent). It is the key differentiator over a naive reachability test.

**Independent Test**: With consent granted, run the check and confirm velox
reports a named nearby server and a plausible distance; with several servers
available, confirm it picks the closest.

**Acceptance Scenarios**:

1. **Given** consent is granted, **When** the user runs the check, **Then** velox resolves an approximate location and reports the nearest server with its distance.
2. **Given** multiple candidate servers, **When** selection runs, **Then** velox chooses the one with the smallest geographic distance to the user.
3. **Given** consent is granted but location cannot be resolved, **When** the lookup fails, **Then** velox degrades gracefully to a fallback server and explains why.
4. **Given** no server is reachable near the user, **When** selection runs, **Then** velox reports that no nearby provider was found and suggests a manual override.

---

### User Story 4 - Scan the Project for Vulnerabilities (Priority: P2)

As an open-source project, velox ships a developer/maintainer command that scans
the codebase and its dependencies for known vulnerabilities and insecure code
patterns, producing a pass/fail report usable locally and in CI.

**Why this priority**: Security hygiene is mandatory for a trusted open-source
networking tool, but it serves maintainers/contributors rather than end users of
the speed test, so it ranks below the user-facing connectivity flow.

**Independent Test**: Run the vulnerability command on the repository and confirm
it produces a report listing findings (or "no issues found") and returns a
non-zero exit code when actionable vulnerabilities exist.

**Acceptance Scenarios**:

1. **Given** the repository, **When** a maintainer runs the vulnerability scan, **Then** velox reports any vulnerable dependencies and insecure code patterns with severity and location.
2. **Given** a clean codebase, **When** the scan runs, **Then** it reports no issues and exits 0.
3. **Given** a finding above the configured severity threshold, **When** the scan runs in CI, **Then** it exits non-zero to fail the build.

---

### Edge Cases

- What happens when the network drops mid-check? → velox reports a clear failure with a non-zero exit code rather than hanging; all network work is time-bounded.
- How does the tool behave with no TTY (CI/cron/pipe)? → consent defaults to declined; location is never requested; fallback server is used.
- What happens when the consent store is missing or corrupt? → treat as "no decision" and prompt again (interactive) or default to declined (non-interactive).
- What happens when location is approved but the geolocation source is unreachable? → degrade to fallback server and inform the user; do not fail the whole run.
- What happens when DNS works but no test server responds? → report "no nearby provider found" with guidance to specify one manually.
- What happens when the user runs on a restricted/offline network? → connectivity check reports offline; provider selection and throughput phases are skipped.
- What happens when the connection drops between the download and upload phases? → report the completed phase(s), mark the failed phase clearly, and exit non-zero rather than reporting a misleading total.
- What happens when the open server registry (M-Lab) is unreachable? → degrade to a fallback/user-specified server and inform the user; do not fail the whole run if a usable server exists.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a `velox --check-internet` command that runs a full speed test against the selected server: confirm connectivity, then measure latency, download throughput, and upload throughput.
- **FR-002**: System MUST report latency, jitter, download throughput, and upload throughput, with units and the sampling window clearly defined, and exit with code 0 on success, non-zero on failure.
- **FR-003**: System MUST emit results in both a human-readable format (default) and a machine-readable JSON format on request, with results on stdout and diagnostics on stderr.
- **FR-004**: System MUST request explicit user approval before performing any location resolution, clearly stating that the user's public IP address is sent to a geolocation lookup and for what purpose.
- **FR-005**: System MUST persist the user's consent decision and MUST NOT re-prompt on subsequent runs unless the decision is reset/revoked.
- **FR-006**: System MUST provide a way to revoke/reset a previously stored consent decision.
- **FR-007**: When consent is declined or unavailable (e.g., non-interactive session), System MUST continue without resolving location, using a default or user-specified fallback server.
- **FR-008**: With consent granted, System MUST resolve the user's approximate (city-level) location from their public IP and select the nearest available provider/test server, reporting the selected server's name and distance.
- **FR-009**: System MUST handle location-resolution and server-selection failures gracefully, degrading to a fallback and explaining the degradation rather than failing the whole run.
- **FR-010**: System MUST allow the user to manually override the selected server.
- **FR-011**: System MUST time-bound all network operations and support cancellation (e.g., Ctrl-C) without leaving the terminal in a broken state.
- **FR-012**: System MUST provide a maintainer command to scan dependencies and source code for known vulnerabilities, reporting findings with severity and location.
- **FR-013**: The vulnerability scan MUST return a non-zero exit code when findings meet or exceed a configurable severity threshold (default: High and above — HIGH/CRITICAL fail the build, MEDIUM/LOW reported as warnings), so it can gate CI.
- **FR-014**: The project MUST be published under an open-source license, with the license clearly stated in the repository.
- **FR-015**: The project MUST maintain a `CHANGELOG.md` documenting notable changes per release, following a recognized changelog convention.
- **FR-016**: `--help` and `--version` MUST work without any network access and without triggering a consent prompt.
- **FR-017**: System MUST source candidate test servers from an open, openly-licensed measurement registry (M-Lab); it MUST NOT depend on a proprietary network (e.g., Ookla) or require an API key.
- **FR-018**: System MUST perform all probing (latency and throughput) over TCP/HTTP(S) and MUST run as a normal user without elevated OS privileges (no root/administrator, no raw sockets).

### Key Entities *(include if feature involves data)*

- **Measurement Result**: Outcome of a speed test — online/offline status, latency, jitter, download throughput, upload throughput, timestamp, and the server it was measured against.
- **Consent Record**: The user's stored decision to allow or deny location use — decision value, and the time it was made; resettable by the user.
- **Location Estimate**: Approximate (city-level) user location derived from the public IP, used only to rank servers; transient, used for selection and not retained beyond what is needed.
- **Provider/Server**: A candidate test endpoint from the open measurement registry (M-Lab) — identifier/name, location, and computed distance/latency to the user.
- **Vulnerability Finding**: A reported issue from the scan — affected component, severity, description, and location reference.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can determine whether their internet is up within 5 seconds of running the command on a normal connection (connectivity is confirmed before throughput phases begin).
- **SC-001b**: A full speed test (connectivity + latency + download + upload) completes and reports all figures within 60 seconds on a normal connection.
- **SC-002**: 100% of first-time location-dependent runs present a consent prompt before any location data is requested.
- **SC-003**: After a user approves or declines once, 0 additional consent prompts appear on later runs until consent is reset.
- **SC-004**: When consent is declined or the session is non-interactive, location is requested in 0% of runs.
- **SC-005**: With consent granted and a location estimate available, the tool selects the geographically nearest available server (smallest great-circle distance) in 100% of such runs; ties are broken by the registry's proximity ordering.
- **SC-006**: Every release has a corresponding `CHANGELOG.md` entry.
- **SC-007**: The vulnerability scan can be run in CI and fails the build when a finding at or above the configured threshold exists, in 100% of such cases.
- **SC-008**: No command leaves a hung process; every network operation completes or is cancelled within its configured timeout in 100% of runs.
- **SC-009**: Download and upload throughput are reported in standard, clearly-labeled units (e.g., Mbps) in 100% of successful runs.
- **SC-010**: The full speed test runs successfully without any elevated OS privileges (no root/administrator) in 100% of supported environments.

## Assumptions

- `velox --check-internet` runs a full speed test: connectivity, latency/jitter, and download + upload throughput against the selected server (clarified 2026-06-30).
- Candidate test servers come from an open, openly-licensed measurement registry (M-Lab); no proprietary network (e.g., Ookla) and no API key (clarified 2026-06-30).
- All probing uses TCP/HTTP(S) and runs without elevated OS privileges; no ICMP/raw sockets (clarified 2026-06-30).
- Location is approximate, IP-based geolocation (city-level), not OS GPS; the consent gate covers sending the public IP to the lookup (clarified 2026-06-30).
- Consent is asked once and remembered in local user configuration, with an explicit reset path; the default when no decision exists in a non-interactive context is "declined".
- The runtime is Go 1.26.4 and the vulnerability tooling includes static analysis (gosec) and dependency vulnerability scanning (govulncheck), gating CI at HIGH+ by default; specific tool wiring is decided in the plan, not the spec.
- The project is distributed as open source with a stated license and a maintained `CHANGELOG.md`.
- Users run velox from a terminal; non-interactive environments (CI, cron, pipes) are supported with safe non-prompting defaults.
- A fallback/default test server exists for when location is unavailable, consent is declined, or the registry is unreachable.

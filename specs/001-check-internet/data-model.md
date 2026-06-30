# Phase 1 Data Model: Check Internet & Nearest Provider

**Feature**: 001-check-internet | **Date**: 2026-06-30

Domain entities derived from the spec. Types are conceptual (Go types shown for
clarity); validation rules trace to functional requirements.

---

## MeasurementResult

The outcome of a `velox --check-internet` run (spec entity "Measurement Result").

| Field | Type | Notes |
|-------|------|-------|
| `online` | bool | Connectivity confirmed before throughput phases (FR-001). |
| `latencyMs` | float64 | Minimum RTT, milliseconds (FR-002). |
| `jitterMs` | float64 | RTT variation over the window (FR-002). |
| `downloadMbps` | float64 | Mean download goodput, Mbps (FR-002, SC-009). |
| `uploadMbps` | float64 | Mean upload goodput, Mbps (FR-002, SC-009). |
| `server` | Server | Server the test ran against. |
| `distanceKm` | *float64 | Client→server distance; nil when consent declined (FR-007). |
| `startedAt` | time.Time | Run start timestamp (UTC). |
| `durationMs` | int64 | Total wall-clock of the run. |
| `phaseStatus` | map[Phase]PhaseOutcome | Per-phase success/failure (FR-001 AS#4). |

**Validation / rules**:
- If `online == false`, throughput phases are skipped and the result is reported
  as offline with a non-zero exit (FR-001 AS#2).
- Units are fixed and labeled (ms, Mbps); never emit a bare number (SC-009).
- A failed phase records its outcome and does not overwrite completed phases with
  misleading values (FR-001 AS#4, edge case).

**Phase** (enum): `connectivity` | `latency` | `download` | `upload`.

**PhaseOutcome**: `{ ok bool, error string (optional), value float64 (optional) }`.

---

## ConsentRecord

The user's stored decision about location use (spec entity "Consent Record").

| Field | Type | Notes |
|-------|------|-------|
| `decision` | ConsentDecision | `granted` \| `denied` \| `unset`. |
| `decidedAt` | time.Time | When the decision was made; zero when `unset`. |

**ConsentDecision** state machine (FR-004/005/006/007):

```text
unset --(user approves)--> granted
unset --(user declines)--> denied
unset --(non-interactive / no TTY)--> treated as denied for this run, store stays unset
granted --(consent --reset)--> unset
denied  --(consent --reset)--> unset
```

**Rules**:
- A location-dependent run with `decision == unset` AND an interactive TTY MUST
  prompt before any IP-geo lookup (FR-004, SC-002).
- Once `granted` or `denied`, no re-prompt on later runs until reset (FR-005,
  SC-003).
- `unset` + non-interactive → behave as `denied` for the run; do not persist
  (FR-007, SC-004).

---

## LocationEstimate

Transient approximate location used only to rank servers (spec entity
"Location Estimate"). Never persisted.

| Field | Type | Notes |
|-------|------|-------|
| `lat` | float64 | Latitude (city-level), from IP geolocation (FR-008, Q4). |
| `lon` | float64 | Longitude (city-level). |
| `city` | string | Optional, for display. |
| `country` | string | Optional, for display. |
| `source` | string | Geo endpoint used (for transparency/debug). |

**Rules**:
- Produced only when `ConsentRecord.decision == granted` (FR-004).
- Held in memory for the duration of the run; not written to disk (privacy).
- On lookup failure: nil → degrade to fallback ordering/server (FR-009).

---

## Server

A candidate ndt7 test endpoint from the M-Lab registry (spec entity
"Provider/Server").

| Field | Type | Notes |
|-------|------|-------|
| `machine` | string | M-Lab machine identifier/name (FR-008 display). |
| `city` | string | Server city. |
| `country` | string | Server country. |
| `lat` | float64 | Server latitude (for haversine). Resolved from a bundled M-Lab site→coordinate table keyed by the site code in `machine` — Locate v2 does not guarantee per-server coords (U1). |
| `lon` | float64 | Server longitude. Same source as `lat`. |
| `downloadURL` | string | `wss://` ndt7 download URL from Locate. |
| `uploadURL` | string | `wss://` ndt7 upload URL from Locate. |
| `isFallback` | bool | True if this is the configured default fallback (FR-007/009). |

**Rules**:
- Sourced from M-Lab Locate v2 (FR-017); never from a proprietary network.
- Selected by smallest haversine distance to `LocationEstimate` when available
  (FR-008), else by Locate ordering or fallback (FR-007/009).
- If no server is reachable, report "no nearby provider found" with manual-
  override guidance (edge case, FR-010).

---

## Config

Persisted user configuration (`os.UserConfigDir()/velox/config.json`).

| Field | Type | Notes |
|-------|------|-------|
| `consent` | ConsentRecord | Embedded consent decision (FR-005). |
| `geoEndpoint` | string | Override for IP-geo lookup URL (R4); empty = default `https://ipwho.is/`. Should be HTTPS (S1). |
| `fallbackServer` | *Server | Optional user-specified fallback (FR-007/010). |
| `schemaVersion` | int | For forward-compatible migrations. |

**Rules**:
- Created on first decision; missing/corrupt file → treated as defaults +
  `consent.decision = unset` (R5, edge case); never crash.
- Written atomically (temp file + rename) to avoid corruption.

---

## VulnerabilityFinding

A reported issue from the maintainer security scan (spec entity
"Vulnerability Finding"). Produced by `scripts/security.sh`, not the runtime
binary.

| Field | Type | Notes |
|-------|------|-------|
| `tool` | string | `gosec` \| `govulncheck`. |
| `id` | string | Rule/CVE identifier. |
| `severity` | Severity | `low` \| `medium` \| `high` \| `critical`. |
| `description` | string | Human-readable summary. |
| `location` | string | File:line or module@version. |

**Rules**:
- The scan exits non-zero when any finding has `severity >= threshold` (default
  `high`) (FR-013, Q5); `medium`/`low` reported as warnings.
- Output is consumable locally and in CI (FR-012).

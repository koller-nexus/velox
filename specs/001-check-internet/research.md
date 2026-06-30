# Phase 0 Research: Check Internet & Nearest Provider

**Feature**: 001-check-internet | **Date**: 2026-06-30

All Technical Context unknowns are resolved below. No `NEEDS CLARIFICATION`
markers remain.

---

## R1. CLI framework: stdlib `flag` vs Cobra

- **Decision**: Use the standard library `flag` package with a small manual
  command router (e.g., dispatch on `--check-internet`, `consent`, sub-flags).
- **Rationale**: Constitution Principle V (stdlib first, no speculative deps).
  The surface is small — one primary action plus consent management. `flag`
  covers it with zero dependencies and a single static binary.
- **Alternatives considered**: `spf13/cobra` (rich subcommands/completion, but a
  heavy dependency tree unjustified for this surface); `urfave/cli` (similar
  trade-off). Revisit only if the command set grows materially.

## R2. Server discovery: M-Lab Locate API v2

- **Decision**: Discover ndt7 servers via the M-Lab Locate API v2 nearest
  endpoint (`https://locate.measurementlab.net/v2/nearest/ndt/ndt7`) using
  stdlib `net/http`. The response returns candidate servers with `machine`,
  `location` (city/country), and ready-to-use measurement URLs (download/upload
  `wss://` for ndt7).
- **Rationale**: Matches the spec's "open, openly-licensed registry (M-Lab)"
  decision (FR-017); no API key; returns proximity-ranked servers plus the exact
  measurement URLs ndt7 needs.
- **Alternatives considered**: Hard-coded server list (goes stale, FR-017 wants a
  registry); Ookla/Speedtest pool (license risk, rejected in spec).
- **Failure handling**: On Locate timeout/unreachability, degrade to a configured
  fallback server (FR-009, edge case) rather than failing the run.

## R3. Measurement protocol: ndt7 via `m-lab/ndt7-client-go`

- **Decision**: Use ndt7 (NDT v7) for latency, download, and upload, via the
  `github.com/m-lab/ndt7-client-go` library, wrapped behind a local `ndt7.Client`
  interface in `internal/ndt7`.
- **Rationale**: ndt7 runs over WebSocket-over-TLS (`wss://`) — TCP/HTTP-upgraded,
  no raw sockets, **no elevated privileges** (FR-018, Q3). The M-Lab client is
  the canonical, Apache-2.0 reference implementation; it reports application-layer
  throughput and minimum RTT. Wrapping it behind an interface keeps unit tests
  network-free (Principle III) and isolates the one third-party dependency.
- **Dependency justification (Principle V)**: Re-implementing ndt7 framing,
  WebSocket upgrade, and the measurement state machine would be large and
  error-prone; the library is purpose-built and openly licensed. This is the only
  third-party runtime dependency. CGO stays disabled → single static binary.
- **Alternatives considered**: Implement ndt7 from scratch on
  `golang.org/x/net/websocket` (high effort, high bug risk); use plain HTTP
  GET/POST throughput against arbitrary servers (not standardized, less
  comparable, no canonical server pool).
- **Metric definitions (Principle IV / FR-002)**: latency = minimum RTT reported
  by ndt7 during the test; jitter = RTT variation over the sampling window;
  download/upload = mean application-layer goodput over the measurement window,
  reported in Mbps. Sampling window and units documented in output and `--help`.

## R4. Location resolution: consent-gated IP geolocation + haversine

- **Decision**: When consent is granted, resolve approximate (city-level)
  client coordinates via a configurable, no-API-key IP-geolocation endpoint over
  **HTTPS** (default candidate: `https://ipwho.is/` — free, no key, TLS, returns
  `latitude`/`longitude`/`city`/`country`; endpoint is overridable via
  config/flag). HTTPS is mandatory by default so the public IP is never sent in
  cleartext (privacy posture behind FR-004). Compute haversine distance from
  client lat/lon to each Locate candidate's coordinates, pick the nearest, and
  display server name + distance.
- **Rationale**: Satisfies FR-008 (resolve from public IP, report distance) and
  Q4 (IP-based, city-level). The consent gate (FR-004) covers sending the public
  IP to this lookup. Haversine on the client keeps ranking transparent and
  testable.
- **When consent is declined / non-interactive**: skip the IP-geo lookup
  entirely; use the Locate API's default proximity ordering or a configured
  fallback server; do not display a computed distance (FR-007). No location data
  leaves the machine via the dedicated lookup.
- **Alternatives considered**: OS GPS/location services (permission prompts,
  platform-specific, overkill — rejected in Q4); rely solely on Locate's
  source-IP ranking (works but cannot show client→server distance and gives the
  user no explicit consent moment); `http://ip-api.com/json` (free/no-key but
  HTTPS requires a paid tier — rejected as default because cleartext leaks the IP
  to on-path observers). The geo endpoint is configurable so users can self-host
  or switch providers for privacy.
- **Privacy note**: The geolocation provider choice is configurable precisely
  because it involves sending the public IP to a third party; default is
  documented and overridable.

## R5. Consent + config storage

- **Decision**: Store a JSON config at `os.UserConfigDir()/velox/config.json`
  containing the consent decision (`granted` / `denied`), timestamp, and optional
  overrides (geo endpoint, fallback server). Managed by `internal/consent` +
  `internal/config`.
- **Rationale**: `os.UserConfigDir()` is cross-platform (XDG on Linux,
  `~/Library/Application Support` on macOS, `%AppData%` on Windows). JSON is
  stdlib, human-inspectable. Consent asked once, remembered (FR-005), reset via
  `velox consent --reset` (FR-006).
- **Non-interactive default**: if no decision exists and stdin/stdout is not a
  TTY, treat consent as **declined** and proceed with fallback (FR-007, SC-004).
  TTY detection via `os.Stdin`/`os.Stdout` `Stat()` char-device check (stdlib).
- **Corrupt/missing store**: treat as "no decision" → prompt (interactive) or
  decline (non-interactive); never crash (edge case).
- **Alternatives considered**: env var only (not persistent), OS keychain
  (overkill for a non-secret boolean).

## R6. Security tooling: gosec + govulncheck (maintainer command)

- **Decision**: Provide `scripts/security.sh` (and `make security` / `make vuln`)
  that runs `gosec` (static analysis) and `govulncheck` (dependency CVEs),
  exiting non-zero when findings are at or above the configured threshold
  (default **HIGH**), per FR-012/FR-013 and Q5. Wired into `.github/workflows/ci.yml`.
- **Rationale**: gosec + govulncheck are developer/CI tools, not runtime features;
  keeping them out of the shipped binary honors Principle V (minimal runtime).
  HIGH+ default fails on HIGH/CRITICAL, reports MEDIUM/LOW as warnings — balances
  security with noise (Q5). Threshold configurable via script flag/env.
- **gosec severity gating**: invoke with `-severity high` (and `-confidence`
  tuning) so the gate matches the chosen threshold; allow override for stricter
  local runs.
- **Alternatives considered**: embedding scans in the `velox` binary (couples a
  speed-test tool to dev tooling — rejected); CI-only (loses local `make security`
  parity — rejected, we provide both).

## R7. Output formatting (human + JSON)

- **Decision**: Default human-readable summary to stdout; `--json` emits a single
  structured object matching `contracts/result.schema.json`. Diagnostics/progress
  to stderr. Exit codes: `0` success, `1` measurement/network failure, `2` usage
  error (per CLI contract).
- **Rationale**: Principle II (text in/out, JSON for scripting, stdout/stderr
  separation, meaningful exit codes). Keeping JSON schema-pinned makes scripting
  and SC-009 (labeled units) verifiable.
- **Alternatives considered**: TUI/progress bars by default (adds deps, pollutes
  piped output — keep behind a TTY check, plain output when piped); multiple
  output formats (YAML/CSV) — deferred, no requirement.

## R8. Cancellation & timeouts

- **Decision**: A root `context.Context` cancelled on SIGINT/SIGTERM
  (`signal.NotifyContext`) threads through Locate, geo, and ndt7 calls. Each phase
  has its own timeout (connectivity, latency, download, upload) plus an overall
  budget aligned to SC-001b (< 60s).
- **Rationale**: Principle IV + FR-011 (time-bounded, cancellable, no hangs,
  clean terminal restore). `signal.NotifyContext` is stdlib.
- **Alternatives considered**: per-call ad-hoc timers (error-prone, inconsistent);
  no overall budget (risks exceeding SC-001b).

---

## Resolved unknowns summary

| Unknown | Resolution |
|---------|------------|
| CLI framework | stdlib `flag` + manual router (R1) |
| Server registry | M-Lab Locate API v2 over net/http (R2) |
| Measurement protocol | ndt7 via m-lab/ndt7-client-go behind interface (R3) |
| Location method | consent-gated IP-geo + haversine; fallback when declined (R4) |
| Consent/config storage | JSON at os.UserConfigDir()/velox/config.json (R5) |
| Security command | scripts/security.sh: gosec + govulncheck, HIGH+ gate (R6) |
| Output | human default + `--json` schema; stdout/stderr split; exit codes (R7) |
| Cancellation | signal.NotifyContext root ctx + per-phase + overall timeouts (R8) |

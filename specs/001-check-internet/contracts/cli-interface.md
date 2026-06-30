# CLI Contract: velox

**Feature**: 001-check-internet | **Date**: 2026-06-30

The command-line surface velox exposes to users (Constitution Principle II).
This is the testable contract for `internal/cli`.

## Commands & Flags

### `velox --check-internet`

Runs a full speed test (FR-001).

| Flag | Type | Default | Effect |
|------|------|---------|--------|
| `--check-internet` | bool | — | Run the full speed test (connectivity → latency → download → upload). |
| `--json` | bool | false | Emit a single JSON object (`result.schema.json`) to stdout instead of human text (FR-003). |
| `--server <url>` | string | "" | Manually override the test server, bypassing selection (FR-010). |
| `--timeout <dur>` | duration | 60s | Overall budget for the run (SC-001b). |
| `-v, --verbose` | bool | false | Verbose diagnostics to stderr (Constitution Tech Standards). |

Behavior:
- Confirms connectivity first; if offline → report on stderr, exit `1` (FR-001 AS#2).
- If consent is `unset` and a TTY is present → prompt before any IP-geo lookup (FR-004).
- If consent `denied` or non-interactive → skip geo, use fallback, no distance shown (FR-007).
- Reports latency, jitter, download Mbps, upload Mbps, server name, and distance (when available).

### `velox consent`

Manage location consent (FR-005/006).

| Flag | Effect |
|------|--------|
| `--status` | Print current decision (`granted` / `denied` / `unset`) and exit 0. |
| `--reset` | Clear stored consent → next location-dependent run prompts again (FR-006). |
| `--grant` | Non-interactively record consent as granted. |
| `--deny` | Non-interactively record consent as denied. |

### `velox --help` / `velox --version`

MUST work with no network access and MUST NOT trigger a consent prompt (FR-016).
`--help` documents metric units and sampling windows (SC-009).

## Output Streams

- **stdout**: results only (human summary, or JSON with `--json`).
- **stderr**: prompts, progress, diagnostics, errors (Principle II).

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success. |
| `1` | Measurement/network failure (offline, no server, phase failed) (FR-001 AS#2). |
| `2` | Usage error (bad flag/args). |

Distinct network-vs-usage codes per Constitution Principle II.

## Human Output (example shape)

```text
Velox speed test
  Server:    mlab1-gru01 (São Paulo, BR) — 12 km
  Latency:   8.4 ms   (jitter 1.2 ms)
  Download:  243.5 Mbps
  Upload:    38.7 Mbps
```

When consent is declined:

```text
Velox speed test  (location disabled — using fallback server)
  Server:    mlab-fallback
  ...
```

## Cancellation

`Ctrl-C` (SIGINT) cancels the in-flight run via the root context, prints a brief
notice to stderr, restores the terminal, and exits non-zero (FR-011).

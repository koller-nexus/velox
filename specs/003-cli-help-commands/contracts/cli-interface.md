# CLI Contract: Help Commands & Additional Useful Commands

**Feature**: `003-cli-help-commands` | **Date**: 2026-07-01

Defines the command surface, I/O streams, and exit codes added by this feature.
Exit codes follow the existing contract: `0` success, `1` measurement/network
failure, `2` usage error (`internal/cli/app.go`: `ExitOK`/`ExitFailure`/
`ExitUsage`).

## Command surface (after this feature)

```text
velox help                     # overview: every command + one-line summary
velox help <command>           # detailed usage for <command>
velox <command> --help         # same detailed usage as `help <command>`
velox --help                   # top-level overview (backward compatible)
velox                          # bare: top-level overview (backward compatible)

velox --check-internet [flags] # unchanged full speed test
velox consent <--status|--grant|--deny|--reset>   # unchanged
velox version                  # NEW: same output as --version
velox servers [--json]         # NEW: list ~5 nearest candidate servers
velox ping [--json] [--server <url>] [--timeout <dur>]   # NEW: latency only
velox config [--json]          # NEW: local state paths + effective settings
velox --version                # unchanged
```

## Global rules

| Rule | Contract |
|------|----------|
| Help/usage text | Written to **stdout**. |
| Errors & diagnostics | Written to **stderr**, prefixed `velox: ` (existing style). |
| Unknown command / flag | stderr message naming the problem + `run 'velox help'`; exit `2`. |
| `--help`/`-h` on any command | Prints that command's detailed usage to stdout; exit `0`; no side effects (no network, no consent prompt, no state change). |
| `help`, `version`, `config` | MUST NOT perform network I/O, location lookup, or consent prompt (offline-safe). |
| Overview completeness | Every registered command appears in `velox help` (FR-007). |

## Per-command contracts

### `velox help [command]`
- No args → overview: each command's `Name` and `Summary`, aligned columns,
  plus a short footer pointing to `velox help <command>`. Exit `0`.
- Known `command` → its detailed `Usage`. Exit `0`.
- Unknown `command` → stderr error + `velox help` hint. Exit `2`.

### `velox version`
- Prints `version.String()` (e.g. `velox 0.1.0-dev (commit …, built …, go… os/arch)`). Exit `0`. No network. No `--json`.

### `velox servers [--json]`
- Applies the consent gate before any location lookup (same as `--check-internet`).
- Human: aligned table of ~5 nearest servers (machine/site, city+country,
  distance km, a marker on the selected one), plus a note when location was not
  used.
- `--json`: a single document conforming to `servers.schema.json`.
- Exit `0` on success; `1` if server discovery fails (network); `2` on bad flags.

### `velox ping [--json] [--server <url>] [--timeout <dur>]`
- Reuses server selection (consent gate) unless `--server` overrides it.
- Measures latency/jitter via a short ndt7 download sample; no throughput.
- Human: `Latency: X ms (jitter Y ms)` with the server line; `--json` conforms
  to `ping.schema.json`.
- Exit `0` when online and the sample succeeds; `1` when offline or the sample
  fails; `2` on bad flags.

### `velox config [--json]`
- Read-only. Human: config path, config dir, consent decision (+ decidedAt),
  effective geo endpoint, fallback server when set.
- `--json`: conforms to `config.schema.json`.
- Exit `0`. No network. `1` only if the config path cannot be resolved.

## Backward-compatibility assertions (must stay green)

- `velox` (no args) → overview, exit `0` (`TestNoArgsIsUsageOnBareAndOKExit`).
- `velox --version` → version line, exit `0` (`TestVersionNoNetwork`).
- `velox --check-internet [--json|--server|--no-progress]` → unchanged.
- `velox consent <flags>` → unchanged (now also `velox help consent` /
  `velox consent --help`).

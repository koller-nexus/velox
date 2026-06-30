# Changelog

All notable changes to velox are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `velox --check-internet`: full internet speed test — connectivity, latency,
  jitter, download and upload throughput — against the nearest open test server.
- Nearest-server selection via the M-Lab Locate API v2 (open registry, no API
  key) ranked by geographic distance using a bundled metro-coordinate table.
- Consent-gated, IP-based geolocation over HTTPS to compute client→server
  distance; consent is asked once, persisted, and revocable (`velox consent`).
- Safe non-interactive default: when no TTY is present, location is never
  requested and a fallback/auto-discovered server is used.
- `--json` machine-readable output, `-v/--verbose` diagnostics on stderr, and
  meaningful exit codes (0 success, 1 measurement/network failure, 2 usage).
- ndt7 measurement over WebSocket-over-TLS — runs without elevated privileges.
- Security gate (`make security`): gosec + govulncheck, failing CI at HIGH+.
- Cross-compiled static binaries for Linux, macOS, and Windows (amd64 + arm64).

[Unreleased]: https://github.com/koller-nexus/velox/commits/main

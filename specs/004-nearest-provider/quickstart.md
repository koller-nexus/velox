# Quickstart: Nearest Internet Provider

This guide validates the nearest-provider feature end-to-end after
implementation.

## Prerequisites

- Go 1.26.4 installed.
- A local clone of the velox repo.
- No prior config, or config with `consent.decision` set to `granted`.

## Build

```bash
make build
```

Expected outcome: `bin/velox` exists and `./bin/velox --version` prints the
version.

## Scenario 1: Default run shows nearest provider metadata

```bash
./bin/velox consent --reset
./bin/velox consent --grant
./bin/velox --check-internet
```

Expected outcome: the human-readable report includes a line such as:

```text
  Nearest provider: Vivo (São Paulo — Centro) — 2.3 km
```

If your location is outside the catalog coverage, the line is omitted and the
test still completes.

## Scenario 2: Machine-readable output includes nearest provider

```bash
./bin/velox --check-internet --json | jq '.nearestProvider'
```

Expected outcome: valid JSON with a `nearestProvider` object containing
`provider.name`, `pop.label`, and `distanceKm`.

## Scenario 3: Opt-in server selection

```bash
./bin/velox --check-internet --nearest-provider
```

Expected outcome: the report identifies the server used and, if available, shows
that it was selected via the nearest-provider path. The measurement completes
normally.

## Scenario 4: Denied consent falls back silently

```bash
./bin/velox consent --reset
./bin/velox consent --deny
./bin/velox --check-internet --nearest-provider
```

Expected outcome: no location lookup, no "Nearest provider" line, no prompt, and
exit code `0` on success.

## Scenario 5: Non-interactive fallback

```bash
./bin/velox consent --reset
./bin/velox --check-internet --nearest-provider 2>&1 | cat
```

Expected outcome: because stdout is piped, no consent prompt appears, the test
runs on the default server, and exit code is `0` on success.

## Scenario 6: Missing catalog does not break the test

Temporarily move/rename `embed/providers.json` (or the embedded file) and run:

```bash
./bin/velox --check-internet
```

Expected outcome: the test completes successfully; nearest-provider metadata is
omitted. Restore the file afterwards.

## Automated validation

```bash
make all
```

Expected outcome: `gofmt`, `go vet`, `golangci-lint`, and `go test -race ./...`
all pass.

## Notes

- See [data-model.md](../data-model.md) for entity definitions.
- See [contracts/cli-interface.md](cli-interface.md) for flag and output
  contracts.
- See [spec.md](../spec.md) for functional requirements and success criteria.

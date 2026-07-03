# CLI Interface Update: Nearest Provider

This document extends the existing velox CLI contract for the nearest-provider
feature.

## New Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--nearest-provider` | bool | false | When set, velox selects the M-Lab test server nearest to the user's closest ISP POP (requires location consent). |

The flag is valid on the root command (`velox --check-internet --nearest-provider`)
and on the `speedtest` subcommand if one is introduced.

## Config Preference

A new optional field may be added to `config.json`:

```json
{
  "nearestProvider": true
}
```

When `nearestProvider` is `true`, the default speed-test behaviour is equivalent
to passing `--nearest-provider`. The CLI flag overrides the config value.

## Exit Codes

No change. Exit codes remain:

- `0` — success.
- `1` — measurement/network failure.
- `2` — usage error (e.g. invalid flag combination).

## Output

### Human-readable output

When a nearest provider is available, an additional block is rendered after the
server line and before the metrics:

```text
Velox speed test
  Server:           mlab2-gru01 (São Paulo, BR) — 12 km
  Nearest provider: Vivo (São Paulo — Centro) — 2.3 km
  Latency:          7.8 ms   (jitter 1.2 ms)
  Download:         347.2 Mbps
  Upload:           322.5 Mbps
```

When unavailable, the line is omitted (no placeholder).

### Machine-readable output

`MeasurementResult` includes an optional `nearestProvider` object:

```json
{
  "online": true,
  "latencyMs": 7.8,
  "jitterMs": 1.2,
  "downloadMbps": 347.2,
  "uploadMbps": 322.5,
  "server": { "machine": "mlab2-gru01", "city": "São Paulo", "country": "BR" },
  "distanceKm": 12.0,
  "nearestProvider": {
    "provider": { "name": "Vivo" },
    "pop": { "label": "São Paulo — Centro", "city": "São Paulo", "country": "BR", "lat": -23.5505, "lon": -46.6333 },
    "distanceKm": 2.3
  }
}
```

## Non-interactive Behaviour

When stdout is not a terminal, stdin is not a TTY, or consent is unset, the
feature does not prompt. `--nearest-provider` is silently ignored and the test
uses the default server selection.

## Help Text

The `--help` output is updated to include the new flag in the speed-test
options section.

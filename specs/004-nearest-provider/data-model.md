# Data Model: Nearest Internet Provider

## Entities

### Provider

Represents an internet service provider that can appear as "nearest" in the
report.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name, e.g. "Vivo", "Claro", "Oi". |
| `countryCodes` | []string | Optional list of ISO country codes where the provider operates; used to filter the catalog for relevance. |

### POP (Point of Presence)

A physical location associated with a provider. Distance is measured from the
user to POPs.

| Field | Type | Description |
|-------|------|-------------|
| `label` | string | Human-readable place label, e.g. "São Paulo — Centro". |
| `city` | string | City name. |
| `region` | string | State/region code, e.g. "SP". |
| `country` | string | Country name or ISO code. |
| `lat` | float64 | Latitude in decimal degrees. |
| `lon` | float64 | Longitude in decimal degrees. |

### ProviderCatalog

The top-level embedded data set.

| Field | Type | Description |
|-------|------|-------------|
| `version` | int | Schema version of the catalog file. |
| `providers` | map[string]Provider | Providers keyed by a stable short code. |
| `pops` | []POP | All POPs, each referencing a provider via `providerCode`. |

### NearestProviderResult

The runtime outcome computed for the current invocation.

| Field | Type | Description |
|-------|------|-------------|
| `provider` | Provider | The selected provider. |
| `pop` | POP | The selected point of presence. |
| `distanceKm` | float64 | Great-circle distance from user to POP, in kilometres. |
| `selectedServer` | *locate.Server | The M-Lab test server chosen when `--nearest-provider` is active; nil in metadata-only mode. |

### MeasurementResult (extended)

The existing `speedtest.MeasurementResult` gains an optional nearest-provider
field for machine-readable output.

| Field | Type | Description |
|-------|------|-------------|
| `nearestProvider` | *NearestProviderResult | Present when location consent is granted and a nearest provider was found; nil otherwise. |

## Validation Rules

- A POP must have `lat` in `[-90, 90]` and `lon` in `[-180, 180]`.
- A POP must reference a provider code that exists in `providers`.
- Provider names must be non-empty and unique within the catalog.
- Catalog loading must fail gracefully: a malformed or missing catalog is not a
  fatal error; the feature falls back to default behaviour.

## State Transitions

No persistent state transitions are introduced by this feature. The only state
is per-invocation and in-memory:

1. Consent decision is read from existing config.
2. If granted, location is resolved once and cached in memory.
3. Nearest provider is computed from the cached location and the catalog.
4. Result is rendered and discarded at process exit.

## Relationships

```text
User Location (1 per invocation)
    │
    ▼
ProviderCatalog (1 per binary)
    │
    ├── Provider (many)
    │
    └── POP (many) ── belongs to ──▶ Provider
    │
    ▼
NearestProviderResult (0 or 1 per invocation)
    │
    └── may reference ──▶ locate.Server (when target mode active)
```

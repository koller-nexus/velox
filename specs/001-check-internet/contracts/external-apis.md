# External API Contracts

**Feature**: 001-check-internet | **Date**: 2026-06-30

velox depends on two external HTTP services plus the ndt7 measurement protocol.
Each is wrapped behind a local interface so unit tests stay network-free
(Constitution Principle III).

---

## 1. M-Lab Locate API v2 — server discovery (`internal/locate`)

**Request**:

```text
GET https://locate.measurementlab.net/v2/nearest/ndt/ndt7
Accept: application/json
```

No API key (FR-017). Ranked by the request's source IP server-side.

**Response (relevant shape)**:

```json
{
  "results": [
    {
      "machine": "mlab1-gru01.mlab-oti.measurement-lab.org",
      "location": { "city": "São Paulo", "country": "BR" },
      "urls": {
        "wss:///ndt/v7/download": "wss://mlab1-gru01.../ndt/v7/download?access_token=...",
        "wss:///ndt/v7/upload":   "wss://mlab1-gru01.../ndt/v7/upload?access_token=..."
      }
    }
  ]
}
```

**Mapping** → `Server`: `machine`, `location.city/country`, download/upload `wss://` URLs.

**Server coordinates (U1)**: Locate v2 does **not** guarantee per-server
latitude/longitude. velox MUST therefore resolve `Server.lat/lon` from a
**bundled M-Lab site→coordinate table** keyed by the site code embedded in
`machine` (e.g., `mlab1-gru01...` → site `gru01`). The table ships in the binary
(`internal/locate/sites.go` or an embedded JSON), is refreshed per release, and
has no runtime dependency. If a site code is missing from the table, that
candidate is treated as un-rankable (excluded from haversine; falls back to
registry ordering) rather than crashing.

**Failure modes**: timeout/5xx/empty `results` → degrade to fallback server
(FR-009); surface a clear message, do not hang (FR-011).

---

## 2. IP Geolocation — consent-gated (`internal/geo`)

**Only called when `consent.decision == granted` (FR-004).**

**Request (default endpoint, overridable via `geoEndpoint`)**:

```text
GET https://ipwho.is/
```

No API key. **HTTPS mandatory by default** so the public IP is never sent in
cleartext (S1 / FR-004 privacy posture). Endpoint configurable for
privacy/self-hosting (R4); an overridden endpoint SHOULD also be HTTPS.

**Response (relevant shape)**:

```json
{ "success": true, "latitude": -23.55, "longitude": -46.63, "city": "São Paulo", "country": "Brazil" }
```

**Mapping** → `LocationEstimate`: `latitude`→`lat`, `longitude`→`lon`, `city`, `country`, `source`.

**Failure modes**: timeout/error/`success != true` → nil estimate → fallback
ordering/server, no distance shown (FR-009). Never persisted (privacy).

---

## 3. ndt7 measurement protocol (`internal/ndt7`)

Wraps `github.com/m-lab/ndt7-client-go`. Runs over `wss://` (TLS WebSocket) —
**no elevated privileges** (FR-018).

**Local interface (contract under test, implementation network-free via fake)**:

```go
type Client interface {
    // Download measures download goodput (Mbps) and min RTT against the server URL.
    Download(ctx context.Context, url string) (Throughput, error)
    // Upload measures upload goodput (Mbps) against the server URL.
    Upload(ctx context.Context, url string) (Throughput, error)
}

type Throughput struct {
    Mbps      float64
    MinRTTMs  float64 // populated during download; feeds latency/jitter
    JitterMs  float64
}
```

**Rules**:
- All methods honor `ctx` cancellation/timeout (FR-011, R8).
- Goodput reported in Mbps; RTT in ms (SC-009, metric defs in research R3).
- Unit tests inject a fake `Client`; integration tests (`//go:build integration`)
  exercise the real M-Lab path.

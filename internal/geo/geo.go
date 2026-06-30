// Package geo resolves an approximate, consent-gated client location from the
// public IP and ranks candidate servers by geographic distance.
package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DefaultEndpoint is a free, no-API-key IP geolocation service reached over
// HTTPS so the public IP is never sent in cleartext (S1).
const DefaultEndpoint = "https://ipwho.is/"

// LocationEstimate is a transient, city-level location. It is never persisted.
type LocationEstimate struct {
	Lat     float64
	Lon     float64
	City    string
	Country string
	Source  string
}

// Resolver resolves the client's approximate location. Faked in tests.
type Resolver interface {
	Resolve(ctx context.Context) (LocationEstimate, error)
}

// IPResolver resolves location via an HTTPS IP-geolocation endpoint.
type IPResolver struct {
	Endpoint string
	HTTP     *http.Client
}

// NewIPResolver returns a resolver using the given endpoint (empty = default).
func NewIPResolver(endpoint string) *IPResolver {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &IPResolver{
		Endpoint: endpoint,
		HTTP:     &http.Client{Timeout: 8 * time.Second},
	}
}

// ipwhoResponse mirrors the relevant subset of the ipwho.is payload.
type ipwhoResponse struct {
	Success   bool    `json:"success"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
}

// Resolve looks up the client's approximate location. The estimate is held in
// memory only (never written to disk).
func (r *IPResolver) Resolve(ctx context.Context) (LocationEstimate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.Endpoint, nil)
	if err != nil {
		return LocationEstimate{}, fmt.Errorf("build geo request: %w", err)
	}
	resp, err := r.HTTP.Do(req)
	if err != nil {
		return LocationEstimate{}, fmt.Errorf("call geo endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return LocationEstimate{}, fmt.Errorf("geo endpoint: unexpected status %d", resp.StatusCode)
	}

	var gr ipwhoResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return LocationEstimate{}, fmt.Errorf("decode geo response: %w", err)
	}
	if !gr.Success {
		return LocationEstimate{}, fmt.Errorf("geo lookup unsuccessful")
	}
	return LocationEstimate{
		Lat:     gr.Latitude,
		Lon:     gr.Longitude,
		City:    gr.City,
		Country: gr.Country,
		Source:  r.Endpoint,
	}, nil
}

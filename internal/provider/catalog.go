package provider

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed embed/providers.json
var providersJSON []byte

// Load reads the embedded provider catalog. A missing or malformed catalog is
// returned as an empty catalog rather than an error so the feature can fall
// back gracefully.
func Load() Catalog {
	var c Catalog
	if len(providersJSON) == 0 {
		return Catalog{Version: 1, Providers: map[string]Provider{}}
	}
	if err := json.Unmarshal(providersJSON, &c); err != nil {
		return Catalog{Version: 1, Providers: map[string]Provider{}}
	}
	if c.Providers == nil {
		c.Providers = map[string]Provider{}
	}
	c.POPs = validPOPs(c)
	return c
}

// LoadFrom parses catalog data from raw bytes, returning an error only when the
// caller needs to distinguish malformed input (tests).
func LoadFrom(data []byte) (Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(data, &c); err != nil {
		return Catalog{}, fmt.Errorf("parse provider catalog: %w", err)
	}
	if c.Providers == nil {
		c.Providers = map[string]Provider{}
	}
	c.POPs = validPOPs(c)
	return c, nil
}

func validPOPs(c Catalog) []POP {
	out := make([]POP, 0, len(c.POPs))
	for _, p := range c.POPs {
		if p.ProviderCode == "" {
			continue
		}
		if _, ok := c.Providers[p.ProviderCode]; !ok {
			continue
		}
		if p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 {
			continue
		}
		out = append(out, p)
	}
	return out
}

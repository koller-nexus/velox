// Package provider models internet providers and their points of presence,
// and selects the provider POP closest to the user's location.
package provider

// Provider is an internet service provider that can appear as "nearest".
type Provider struct {
	Name         string   `json:"name"`
	CountryCodes []string `json:"countryCodes,omitempty"`
}

// POP is a point of presence belonging to a provider.
type POP struct {
	ProviderCode string  `json:"providerCode"`
	Label        string  `json:"label"`
	City         string  `json:"city"`
	Region       string  `json:"region,omitempty"`
	Country      string  `json:"country"`
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
}

// Catalog is the bundled collection of providers and POPs.
type Catalog struct {
	Version   int                 `json:"version"`
	Providers map[string]Provider `json:"providers"`
	POPs      []POP               `json:"pops"`
}

// Result is the selected nearest provider/POP for the current run.
type Result struct {
	Provider       Provider `json:"provider"`
	POP            POP      `json:"pop"`
	DistanceKm     float64  `json:"distanceKm"`
	SelectedServer *string  `json:"selectedServer,omitempty"`
}

// Empty reports whether the catalog has no providers or POPs.
func (c Catalog) Empty() bool {
	return len(c.Providers) == 0 || len(c.POPs) == 0
}

// ProviderName returns the display name for a provider code, or the code itself
// if unknown.
func (c Catalog) ProviderName(code string) string {
	if p, ok := c.Providers[code]; ok {
		return p.Name
	}
	return code
}

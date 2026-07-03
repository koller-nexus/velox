package provider

import (
	"math"
	"sort"

	"github.com/koller-nexus/velox/internal/geo"
)

// Finder selects the nearest provider POP to a location estimate.
type Finder struct {
	Catalog Catalog
}

// NewFinder returns a Finder using the embedded catalog.
func NewFinder() *Finder {
	return &Finder{Catalog: Load()}
}

// Nearest returns the provider POP with the smallest great-circle distance to
// est. It returns false when the catalog is empty or no POP is available.
func (f *Finder) Nearest(est geo.LocationEstimate) (Result, bool) {
	if f.Catalog.Empty() {
		return Result{}, false
	}

	type ranked struct {
		pop  POP
		dist float64
	}

	var items []ranked
	for _, pop := range f.Catalog.POPs {
		d := geo.HaversineKm(est.Lat, est.Lon, pop.Lat, pop.Lon)
		items = append(items, ranked{pop: pop, dist: d})
	}
	if len(items) == 0 {
		return Result{}, false
	}

	sort.SliceStable(items, func(i, j int) bool {
		if math.Abs(items[i].dist-items[j].dist) > 1e-9 {
			return items[i].dist < items[j].dist
		}
		// Deterministic tie-break: provider code, then POP label.
		if items[i].pop.ProviderCode != items[j].pop.ProviderCode {
			return items[i].pop.ProviderCode < items[j].pop.ProviderCode
		}
		return items[i].pop.Label < items[j].pop.Label
	})

	best := items[0]
	provider := f.Catalog.Providers[best.pop.ProviderCode]
	return Result{
		Provider:   provider,
		POP:        best.pop,
		DistanceKm: best.dist,
	}, true
}

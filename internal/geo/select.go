package geo

import (
	"sort"

	"github.com/koller-nexus/velox/internal/locate"
)

// Selection is the chosen server plus the client->server distance, when known.
type Selection struct {
	Server     locate.Server
	DistanceKm *float64 // nil when no location estimate was available
}

// SelectNearest picks the candidate with the smallest great-circle distance to
// the estimate, considering only candidates with known coordinates (U1). When
// est is nil or no candidate has coordinates, it falls back to the first
// candidate (the registry's proximity ordering) with a nil distance (FR-007/009).
func SelectNearest(servers []locate.Server, est *LocationEstimate) (Selection, bool) {
	if len(servers) == 0 {
		return Selection{}, false
	}
	if est == nil {
		return Selection{Server: servers[0]}, true
	}

	bestIdx := -1
	var bestDist float64
	for i, s := range servers {
		if !s.HasCoords {
			continue
		}
		d := HaversineKm(est.Lat, est.Lon, s.Lat, s.Lon)
		if bestIdx == -1 || d < bestDist {
			bestIdx, bestDist = i, d
		}
	}
	if bestIdx == -1 {
		// No candidate had coordinates; fall back to registry ordering.
		return Selection{Server: servers[0]}, true
	}
	dist := bestDist
	return Selection{Server: servers[bestIdx], DistanceKm: &dist}, true
}

// RankByDistance orders candidates nearest-first when a location estimate is
// available, computing each client->server distance; candidates with known
// coordinates sort ahead of those without (which keep their registry order).
// When est is nil, candidates are returned in their original registry order with
// nil distances. The first element is always the server SelectNearest would pick,
// so callers can mark it as the selected server.
func RankByDistance(servers []locate.Server, est *LocationEstimate) []Selection {
	out := make([]Selection, 0, len(servers))
	if est == nil {
		for _, s := range servers {
			out = append(out, Selection{Server: s})
		}
		return out
	}

	type ranked struct {
		sel      Selection
		dist     float64
		hasCoord bool
		order    int
	}
	items := make([]ranked, 0, len(servers))
	for i, s := range servers {
		if s.HasCoords {
			d := HaversineKm(est.Lat, est.Lon, s.Lat, s.Lon)
			dd := d
			items = append(items, ranked{Selection{Server: s, DistanceKm: &dd}, d, true, i})
		} else {
			items = append(items, ranked{Selection{Server: s}, 0, false, i})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].hasCoord != items[j].hasCoord {
			return items[i].hasCoord // coord'd candidates first
		}
		if items[i].hasCoord {
			return items[i].dist < items[j].dist
		}
		return items[i].order < items[j].order // preserve registry order otherwise
	})
	for _, it := range items {
		out = append(out, it.sel)
	}
	return out
}

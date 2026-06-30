package geo

import "github.com/koller-nexus/velox/internal/locate"

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

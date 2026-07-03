package provider

import (
	"github.com/koller-nexus/velox/internal/geo"
	"github.com/koller-nexus/velox/internal/locate"
)

// SelectServerForProvider picks the M-Lab candidate server that is
// geographically nearest to the given provider POP. It returns the original
// registry's first server when no candidate has coordinates.
func SelectServerForProvider(candidates []locate.Server, nearest Result) (locate.Server, *float64) {
	if len(candidates) == 0 {
		return locate.Server{}, nil
	}

	bestIdx := -1
	var bestDist float64
	for i, s := range candidates {
		if !s.HasCoords {
			continue
		}
		d := geo.HaversineKm(nearest.POP.Lat, nearest.POP.Lon, s.Lat, s.Lon)
		if bestIdx == -1 || d < bestDist {
			bestIdx, bestDist = i, d
		}
	}

	if bestIdx == -1 {
		return candidates[0], nil
	}
	d := bestDist
	return candidates[bestIdx], &d
}

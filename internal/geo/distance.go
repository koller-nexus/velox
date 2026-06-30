package geo

import "math"

const earthRadiusKm = 6371.0

// HaversineKm returns the great-circle distance in kilometres between two
// latitude/longitude points.
func HaversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := rad(lat2 - lat1)
	dLon := rad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func rad(deg float64) float64 { return deg * math.Pi / 180.0 }

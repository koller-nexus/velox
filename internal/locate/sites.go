package locate

import "strings"

// metroCoords maps an M-Lab metro code (the 3-letter IATA prefix of a site
// code, e.g. "gru" in "gru01") to its approximate latitude/longitude. This
// bundled table lets velox compute client->server distance without a network
// call and without per-server coordinates from Locate (U1).
//
// It is intentionally a representative subset of major M-Lab metros, refreshed
// per release. A site whose metro is absent here is treated as un-rankable and
// excluded from distance ranking (it can still be measured against).
var metroCoords = map[string][2]float64{
	"gru": {-23.4356, -46.4731}, // São Paulo
	"gig": {-22.8100, -43.2506}, // Rio de Janeiro
	"for": {-3.7763, -38.5326},  // Fortaleza
	"scl": {-33.3930, -70.7858}, // Santiago
	"eze": {-34.8222, -58.5358}, // Buenos Aires
	"bog": {4.7016, -74.1469},   // Bogotá
	"lim": {-12.0219, -77.1143}, // Lima
	"mia": {25.7959, -80.2870},  // Miami
	"atl": {33.6407, -84.4277},  // Atlanta
	"iad": {38.9531, -77.4565},  // Washington DC
	"jfk": {40.6413, -73.7781},  // New York
	"ord": {41.9742, -87.9073},  // Chicago
	"dfw": {32.8998, -97.0403},  // Dallas
	"den": {39.8561, -104.6737}, // Denver
	"lax": {33.9416, -118.4085}, // Los Angeles
	"sfo": {37.6213, -122.3790}, // San Francisco
	"sea": {47.4502, -122.3088}, // Seattle
	"yyz": {43.6777, -79.6248},  // Toronto
	"lhr": {51.4700, -0.4543},   // London
	"lis": {38.7742, -9.1342},   // Lisbon
	"mad": {40.4983, -3.5676},   // Madrid
	"par": {49.0097, 2.5479},    // Paris
	"ams": {52.3105, 4.7683},    // Amsterdam
	"fra": {50.0379, 8.5622},    // Frankfurt
	"mil": {45.6306, 8.7281},    // Milan
	"arn": {59.6498, 17.9239},   // Stockholm
	"waw": {52.1657, 20.9671},   // Warsaw
	"tnr": {-18.7969, 47.4788},  // Antananarivo
	"jnb": {-26.1367, 28.2411},  // Johannesburg
	"los": {6.5774, 3.3212},     // Lagos
	"nbo": {-1.3192, 36.9278},   // Nairobi
	"bom": {19.0896, 72.8656},   // Mumbai
	"del": {28.5562, 77.1000},   // Delhi
	"sin": {1.3644, 103.9915},   // Singapore
	"nrt": {35.7720, 140.3929},  // Tokyo
	"hnd": {35.5494, 139.7798},  // Tokyo Haneda
	"icn": {37.4602, 126.4407},  // Seoul
	"syd": {-33.9399, 151.1753}, // Sydney
	"akl": {-37.0082, 174.7850}, // Auckland
}

// Coords returns the approximate latitude/longitude for a site code's metro.
func Coords(siteCode string) (lat, lon float64, ok bool) {
	metro := metro(siteCode)
	c, ok := metroCoords[metro]
	if !ok {
		return 0, 0, false
	}
	return c[0], c[1], true
}

// metro returns the leading alphabetic metro prefix of a site code
// ("gru01" -> "gru").
func metro(siteCode string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(siteCode) {
		if r < 'a' || r > 'z' {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

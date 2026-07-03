package provider

import (
	"testing"

	"github.com/koller-nexus/velox/internal/geo"
)

func TestNearest_Basic(t *testing.T) {
	c := Catalog{
		Version: 1,
		Providers: map[string]Provider{
			"vivo":  {Name: "Vivo"},
			"claro": {Name: "Claro"},
		},
		POPs: []POP{
			{ProviderCode: "vivo", Label: "Far", City: "Far", Country: "BR", Lat: 0, Lon: 0},
			{ProviderCode: "claro", Label: "Near", City: "Near", Country: "BR", Lat: -23.55, Lon: -46.63},
		},
	}
	f := Finder{Catalog: c}
	est := geo.LocationEstimate{Lat: -23.55, Lon: -46.63}
	res, ok := f.Nearest(est)
	if !ok {
		t.Fatal("expected a nearest provider")
	}
	if res.Provider.Name != "Claro" {
		t.Errorf("got provider %q, want Claro", res.Provider.Name)
	}
	if res.DistanceKm > 1 {
		t.Errorf("distance too large: %f", res.DistanceKm)
	}
}

func TestNearest_EmptyCatalog(t *testing.T) {
	f := Finder{Catalog: Catalog{Version: 1, Providers: map[string]Provider{}}}
	_, ok := f.Nearest(geo.LocationEstimate{Lat: 0, Lon: 0})
	if ok {
		t.Error("expected no result for empty catalog")
	}
}

func TestNearest_DeterministicTie(t *testing.T) {
	c := Catalog{
		Version: 1,
		Providers: map[string]Provider{
			"vivo":  {Name: "Vivo"},
			"claro": {Name: "Claro"},
		},
		POPs: []POP{
			{ProviderCode: "vivo", Label: "B", City: "X", Country: "BR", Lat: 10, Lon: 10},
			{ProviderCode: "claro", Label: "A", City: "X", Country: "BR", Lat: 10, Lon: 10},
		},
	}
	f := Finder{Catalog: c}
	est := geo.LocationEstimate{Lat: 10, Lon: 10}

	var names []string
	for i := 0; i < 10; i++ {
		res, ok := f.Nearest(est)
		if !ok {
			t.Fatal("expected result")
		}
		names = append(names, res.Provider.Name)
	}
	for i := 1; i < len(names); i++ {
		if names[i] != names[0] {
			t.Fatalf("tie-breaking not deterministic: %v", names)
		}
	}
}

func TestNearest_NoPOPs(t *testing.T) {
	c := Catalog{
		Version:   1,
		Providers: map[string]Provider{"vivo": {Name: "Vivo"}},
		POPs:      []POP{},
	}
	f := Finder{Catalog: c}
	_, ok := f.Nearest(geo.LocationEstimate{Lat: 0, Lon: 0})
	if ok {
		t.Error("expected no result when catalog has no POPs")
	}
}

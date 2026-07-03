package provider

import (
	"testing"

	"github.com/koller-nexus/velox/internal/locate"
)

func TestSelectServerForProvider(t *testing.T) {
	nearest := Result{
		POP: POP{Lat: -23.55, Lon: -46.63},
	}
	candidates := []locate.Server{
		{Machine: "london", Lat: 51.47, Lon: -0.45, HasCoords: true},
		{Machine: "saopaulo", Lat: -23.43, Lon: -46.47, HasCoords: true},
		{Machine: "miami", Lat: 25.8, Lon: -80.29, HasCoords: true},
	}

	srv, dist := SelectServerForProvider(candidates, nearest)
	if srv.Machine != "saopaulo" {
		t.Errorf("got server %q, want saopaulo", srv.Machine)
	}
	if dist == nil {
		t.Fatal("expected distance")
	}
	if *dist > 50 {
		t.Errorf("distance too large: %f", *dist)
	}
}

func TestSelectServerForProvider_NoCoords(t *testing.T) {
	nearest := Result{
		POP: POP{Lat: -23.55, Lon: -46.63},
	}
	candidates := []locate.Server{
		{Machine: "first"},
		{Machine: "second"},
	}

	srv, dist := SelectServerForProvider(candidates, nearest)
	if srv.Machine != "first" {
		t.Errorf("got server %q, want first", srv.Machine)
	}
	if dist != nil {
		t.Error("expected nil distance when no candidate has coordinates")
	}
}

func TestSelectServerForProvider_Empty(t *testing.T) {
	srv, dist := SelectServerForProvider(nil, Result{})
	if srv.Machine != "" || dist != nil {
		t.Errorf("expected empty server and nil distance, got %+v %v", srv, dist)
	}
}

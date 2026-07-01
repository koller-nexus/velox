package geo

import (
	"testing"

	"github.com/koller-nexus/velox/internal/locate"
)

func TestSelectNearestPicksClosest(t *testing.T) {
	servers := []locate.Server{
		{Machine: "far", Lat: 51.47, Lon: -0.45, HasCoords: true},    // London
		{Machine: "near", Lat: -23.43, Lon: -46.47, HasCoords: true}, // São Paulo
	}
	est := &LocationEstimate{Lat: -23.55, Lon: -46.63} // São Paulo client
	sel, ok := SelectNearest(servers, est)
	if !ok {
		t.Fatal("expected a selection")
	}
	if sel.Server.Machine != "near" {
		t.Errorf("picked %q, want near", sel.Server.Machine)
	}
	if sel.DistanceKm == nil || *sel.DistanceKm > 100 {
		t.Errorf("distance should be small and set, got %v", sel.DistanceKm)
	}
}

func TestSelectNearestNoEstimateUsesFirst(t *testing.T) {
	servers := []locate.Server{{Machine: "first"}, {Machine: "second"}}
	sel, ok := SelectNearest(servers, nil)
	if !ok || sel.Server.Machine != "first" {
		t.Errorf("want registry-order first, got %+v", sel)
	}
	if sel.DistanceKm != nil {
		t.Errorf("distance must be nil without estimate")
	}
}

func TestSelectNearestNoCoordsFallsBack(t *testing.T) {
	servers := []locate.Server{{Machine: "a"}, {Machine: "b"}} // no coords
	est := &LocationEstimate{Lat: 1, Lon: 1}
	sel, ok := SelectNearest(servers, est)
	if !ok || sel.Server.Machine != "a" {
		t.Errorf("want fallback to first, got %+v", sel)
	}
	if sel.DistanceKm != nil {
		t.Errorf("distance must be nil when no candidate has coords")
	}
}

func TestSelectNearestEmpty(t *testing.T) {
	if _, ok := SelectNearest(nil, nil); ok {
		t.Error("empty candidates should return ok=false")
	}
}

func TestRankByDistanceOrdersNearestFirst(t *testing.T) {
	servers := []locate.Server{
		{Machine: "far", Lat: 51.47, Lon: -0.45, HasCoords: true},    // London
		{Machine: "near", Lat: -23.43, Lon: -46.47, HasCoords: true}, // São Paulo
	}
	est := &LocationEstimate{Lat: -23.55, Lon: -46.63} // São Paulo client
	ranked := RankByDistance(servers, est)
	if len(ranked) != 2 {
		t.Fatalf("len = %d, want 2", len(ranked))
	}
	if ranked[0].Server.Machine != "near" {
		t.Errorf("nearest first = %q, want near", ranked[0].Server.Machine)
	}
	if ranked[0].DistanceKm == nil || *ranked[0].DistanceKm > 100 {
		t.Errorf("nearest distance should be small and set, got %v", ranked[0].DistanceKm)
	}
	// The head matches what SelectNearest would pick.
	sel, _ := SelectNearest(servers, est)
	if ranked[0].Server.Machine != sel.Server.Machine {
		t.Errorf("head %q != SelectNearest %q", ranked[0].Server.Machine, sel.Server.Machine)
	}
}

func TestRankByDistanceNoEstimateKeepsRegistryOrder(t *testing.T) {
	servers := []locate.Server{{Machine: "first"}, {Machine: "second"}}
	ranked := RankByDistance(servers, nil)
	if len(ranked) != 2 || ranked[0].Server.Machine != "first" || ranked[1].Server.Machine != "second" {
		t.Errorf("registry order not preserved: %+v", ranked)
	}
	if ranked[0].DistanceKm != nil {
		t.Errorf("distance must be nil without estimate")
	}
}

func TestRankByDistanceCoordedBeforeUncoorded(t *testing.T) {
	servers := []locate.Server{
		{Machine: "nocoord"},
		{Machine: "coord", Lat: 1, Lon: 1, HasCoords: true},
	}
	est := &LocationEstimate{Lat: 1, Lon: 1}
	ranked := RankByDistance(servers, est)
	if ranked[0].Server.Machine != "coord" {
		t.Errorf("candidate with coords should rank first, got %q", ranked[0].Server.Machine)
	}
}

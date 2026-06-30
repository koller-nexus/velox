package geo

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveParsesLocation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"latitude":-23.55,"longitude":-46.63,"city":"São Paulo","country":"Brazil"}`))
	}))
	defer srv.Close()

	r := &IPResolver{Endpoint: srv.URL, HTTP: srv.Client()}
	est, err := r.Resolve(context.Background())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if est.Lat != -23.55 || est.Lon != -46.63 {
		t.Errorf("coords = %v,%v want -23.55,-46.63", est.Lat, est.Lon)
	}
	if est.City != "São Paulo" {
		t.Errorf("city = %q", est.City)
	}
}

func TestResolveUnsuccessful(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false}`))
	}))
	defer srv.Close()
	r := &IPResolver{Endpoint: srv.URL, HTTP: srv.Client()}
	if _, err := r.Resolve(context.Background()); err == nil {
		t.Fatal("expected error when success=false")
	}
}

func TestHaversineKnownDistance(t *testing.T) {
	// São Paulo (GRU) to Rio (GIG) ~ 360 km.
	d := HaversineKm(-23.4356, -46.4731, -22.8100, -43.2506)
	if math.Abs(d-360) > 40 {
		t.Errorf("distance = %.1f km, want ~360", d)
	}
}

func TestNewIPResolverDefaultsToHTTPS(t *testing.T) {
	r := NewIPResolver("")
	if r.Endpoint != DefaultEndpoint {
		t.Errorf("endpoint = %q, want default", r.Endpoint)
	}
	if r.Endpoint[:8] != "https://" {
		t.Errorf("default endpoint must be HTTPS, got %q", r.Endpoint)
	}
}

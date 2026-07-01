package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/koller-nexus/velox/internal/config"
	"github.com/koller-nexus/velox/internal/locate"
)

type serversJSON struct {
	LocationUsed bool `json:"locationUsed"`
	Servers      []struct {
		Machine    string   `json:"machine"`
		Site       string   `json:"site"`
		DistanceKm *float64 `json:"distanceKm"`
		Selected   bool     `json:"selected"`
	} `json:"servers"`
}

func TestServersListsNearestWithSingleSelection(t *testing.T) {
	servers := []locate.Server{
		{Machine: "london", SiteCode: "lhr01", Lat: 51.47, Lon: -0.45, HasCoords: true, City: "London", Country: "GB"},
		{Machine: "saopaulo", SiteCode: "gru01", Lat: -23.43, Lon: -46.47, HasCoords: true, City: "Sao Paulo", Country: "BR"},
	}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{servers: servers}, &fakeRunner{}, fakeConsent{decision: config.DecisionGranted})
	if code := a.Run(context.Background(), []string{"servers", "--json"}); code != ExitOK {
		t.Fatalf("exit = %d, stderr=%s", code, errw.String())
	}

	var got serversJSON
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("servers --json invalid: %v\n%s", err, out.String())
	}
	if !got.LocationUsed {
		t.Errorf("location should be used when consent is granted and resolver succeeds")
	}
	if n := len(got.Servers); n == 0 || n > 5 {
		t.Fatalf("want 1..5 servers, got %d", n)
	}
	if !got.Servers[0].Selected {
		t.Errorf("nearest (first) server must be marked selected")
	}
	selected := 0
	for _, s := range got.Servers {
		if s.Selected {
			selected++
		}
	}
	if selected != 1 {
		t.Errorf("exactly one server must be selected, got %d", selected)
	}
	if got.Servers[0].Machine != "saopaulo" {
		t.Errorf("nearest to São Paulo client should be saopaulo, got %q", got.Servers[0].Machine)
	}
	if got.Servers[0].DistanceKm == nil {
		t.Errorf("selected server should carry a distance when location used")
	}
}

func TestServersFallbackWhenConsentDenied(t *testing.T) {
	servers := []locate.Server{{Machine: "a"}, {Machine: "b"}} // no coords
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{servers: servers}, &fakeRunner{}, fakeConsent{decision: config.DecisionDenied})
	if code := a.Run(context.Background(), []string{"servers", "--json"}); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	var got serversJSON
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got.LocationUsed {
		t.Errorf("location must NOT be used when consent denied")
	}
	for _, s := range got.Servers {
		if s.DistanceKm != nil {
			t.Errorf("distance must be null under registry-order fallback, got %v", *s.DistanceKm)
		}
	}
}

func TestServersDiscoveryFailureIsFailure(t *testing.T) {
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{err: context.DeadlineExceeded}, &fakeRunner{}, fakeConsent{})
	if code := a.Run(context.Background(), []string{"servers"}); code != ExitFailure {
		t.Fatalf("exit = %d, want %d", code, ExitFailure)
	}
	if errw.Len() == 0 {
		t.Errorf("discovery failure should report to stderr")
	}
}

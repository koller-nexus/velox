package locate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const sampleResponse = `{
  "results": [
    {
      "machine": "mlab1-gru01.mlab-oti.measurement-lab.org",
      "location": {"city": "São Paulo", "country": "BR"},
      "urls": {
        "wss:///ndt/v7/download": "wss://mlab1-gru01.example/ndt/v7/download?access_token=t",
        "wss:///ndt/v7/upload":   "wss://mlab1-gru01.example/ndt/v7/upload?access_token=t"
      }
    },
    {
      "machine": "mlab2-zzz99.mlab-oti.measurement-lab.org",
      "location": {"city": "Nowhere", "country": "XX"},
      "urls": {
        "wss:///ndt/v7/download": "wss://mlab2-zzz99.example/ndt/v7/download",
        "wss:///ndt/v7/upload":   "wss://mlab2-zzz99.example/ndt/v7/upload"
      }
    }
  ]
}`

func TestNearestParsesServers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleResponse))
	}))
	defer srv.Close()

	c := &Client{URL: srv.URL, HTTP: srv.Client()}
	servers, err := c.Nearest(context.Background())
	if err != nil {
		t.Fatalf("Nearest: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("got %d servers, want 2", len(servers))
	}

	gru := servers[0]
	if gru.SiteCode != "gru01" {
		t.Errorf("SiteCode = %q, want gru01", gru.SiteCode)
	}
	if !gru.HasCoords {
		t.Errorf("gru01 should resolve coords from the bundled table")
	}
	if gru.DownloadURL == "" || gru.UploadURL == "" {
		t.Errorf("download/upload URLs not parsed: %+v", gru)
	}

	// Unknown metro -> no coords, but still a valid candidate.
	if servers[1].HasCoords {
		t.Errorf("zzz99 should have no coords")
	}
}

func TestNearestNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	c := &Client{URL: srv.URL, HTTP: srv.Client()}
	if _, err := c.Nearest(context.Background()); err == nil {
		t.Fatal("expected error on non-200")
	}
}

func TestSiteCode(t *testing.T) {
	cases := map[string]string{
		"mlab1-gru01.mlab-oti.measurement-lab.org": "gru01",
		"mlab3-lhr05.foo.bar":                      "lhr05",
		"weird":                                    "weird",
	}
	for in, want := range cases {
		if got := SiteCode(in); got != want {
			t.Errorf("SiteCode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCoordsMetroLookup(t *testing.T) {
	if _, _, ok := Coords("gru01"); !ok {
		t.Error("gru01 metro should be found")
	}
	if _, _, ok := Coords("zzz99"); ok {
		t.Error("zzz99 metro should not be found")
	}
}

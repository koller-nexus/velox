package provider

import (
	"testing"
)

func TestLoadFrom_Valid(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"providers": {"vivo": {"name": "Vivo"}},
		"pops": [{"providerCode": "vivo", "label": "São Paulo", "city": "São Paulo", "country": "BR", "lat": -23.55, "lon": -46.63}]
	}`)
	c, err := LoadFrom(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Version != 1 {
		t.Errorf("version = %d, want 1", c.Version)
	}
	if len(c.POPs) != 1 {
		t.Errorf("got %d POPs, want 1", len(c.POPs))
	}
}

func TestLoadFrom_InvalidJSON(t *testing.T) {
	_, err := LoadFrom([]byte(`{not json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestLoadFrom_MissingProvider(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"providers": {"vivo": {"name": "Vivo"}},
		"pops": [
			{"providerCode": "vivo", "label": "SP", "city": "São Paulo", "country": "BR", "lat": -23.55, "lon": -46.63},
			{"providerCode": "claro", "label": "RJ", "city": "Rio", "country": "BR", "lat": -22.9, "lon": -43.17}
		]
	}`)
	c, err := LoadFrom(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.POPs) != 1 {
		t.Errorf("got %d POPs, want 1 (unknown provider filtered)", len(c.POPs))
	}
}

func TestLoadFrom_InvalidCoordinates(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"providers": {"vivo": {"name": "Vivo"}},
		"pops": [
			{"providerCode": "vivo", "label": "Bad", "city": "X", "country": "BR", "lat": 999, "lon": -46.63},
			{"providerCode": "vivo", "label": "OK", "city": "Y", "country": "BR", "lat": -23.55, "lon": -46.63}
		]
	}`)
	c, err := LoadFrom(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.POPs) != 1 || c.POPs[0].Label != "OK" {
		t.Errorf("invalid coordinate POP not filtered: %+v", c.POPs)
	}
}

func TestLoad_EmbeddedNotEmpty(t *testing.T) {
	c := Load()
	if c.Empty() {
		t.Fatal("embedded catalog should not be empty")
	}
	if _, ok := c.Providers["vivo"]; !ok {
		t.Error("embedded catalog missing Vivo provider")
	}
}

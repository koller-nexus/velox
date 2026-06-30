package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadFromMissingReturnsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")
	c, err := loadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Consent.Decision != DecisionUnset {
		t.Errorf("decision = %q, want %q", c.Consent.Decision, DecisionUnset)
	}
	if c.SchemaVersion != SchemaVersion {
		t.Errorf("schemaVersion = %d, want %d", c.SchemaVersion, SchemaVersion)
	}
}

func TestLoadFromCorruptReturnsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := loadFrom(path)
	if err != nil {
		t.Fatalf("corrupt file should not error, got: %v", err)
	}
	if c.Consent.Decision != DecisionUnset {
		t.Errorf("decision = %q, want unset on corrupt file", c.Consent.Decision)
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	now := time.Now().UTC().Truncate(time.Second)
	want := Config{
		SchemaVersion: SchemaVersion,
		Consent:       Consent{Decision: DecisionGranted, DecidedAt: &now},
		GeoEndpoint:   "https://example.test/",
	}
	if err := saveTo(path, want); err != nil {
		t.Fatalf("saveTo: %v", err)
	}
	got, err := loadFrom(path)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}
	if got.Consent.Decision != DecisionGranted {
		t.Errorf("decision = %q, want granted", got.Consent.Decision)
	}
	if got.GeoEndpoint != want.GeoEndpoint {
		t.Errorf("geoEndpoint = %q, want %q", got.GeoEndpoint, want.GeoEndpoint)
	}
}

func TestSaveIsAtomicNoTempLeft(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := saveTo(path, Default()); err != nil {
		t.Fatalf("saveTo: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

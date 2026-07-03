// Package config persists velox user configuration (consent decision and
// optional overrides) as JSON under the OS user config directory.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// SchemaVersion is the current config schema version.
const SchemaVersion = 1

// Decision is the user's location-consent decision.
type Decision string

const (
	// DecisionUnset means the user has not yet decided.
	DecisionUnset Decision = "unset"
	// DecisionGranted means the user allowed location use.
	DecisionGranted Decision = "granted"
	// DecisionDenied means the user denied location use.
	DecisionDenied Decision = "denied"
)

// Consent is the stored consent record (see contracts/consent.schema.json).
type Consent struct {
	Decision  Decision   `json:"decision"`
	DecidedAt *time.Time `json:"decidedAt,omitempty"`
}

// FallbackServer is an optional user-specified fallback test server.
type FallbackServer struct {
	Machine     string `json:"machine,omitempty"`
	DownloadURL string `json:"downloadURL,omitempty"`
	UploadURL   string `json:"uploadURL,omitempty"`
}

// Config is the persisted velox configuration.
type Config struct {
	SchemaVersion   int             `json:"schemaVersion"`
	Consent         Consent         `json:"consent"`
	GeoEndpoint     string          `json:"geoEndpoint,omitempty"`
	NearestProvider bool            `json:"nearestProvider,omitempty"`
	FallbackServer  *FallbackServer `json:"fallbackServer,omitempty"`
}

// Default returns a fresh config with an unset consent decision.
func Default() Config {
	return Config{
		SchemaVersion: SchemaVersion,
		Consent:       Consent{Decision: DecisionUnset},
	}
}

// Path returns the absolute path to the velox config file.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(dir, "velox", "config.json"), nil
}

// Load reads the config from disk. A missing or corrupt file is treated as a
// fresh default config (never an error), so velox never crashes on a bad store.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	return loadFrom(path)
}

func loadFrom(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Default(), nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		// Corrupt file: fall back to defaults rather than failing.
		return Default(), nil
	}
	if c.SchemaVersion == 0 {
		c.SchemaVersion = SchemaVersion
	}
	if c.Consent.Decision == "" {
		c.Consent.Decision = DecisionUnset
	}
	return c, nil
}

// Save writes the config to disk atomically (temp file + rename).
func Save(c Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	return saveTo(path, c)
}

func saveTo(path string, c Config) error {
	if c.SchemaVersion == 0 {
		c.SchemaVersion = SchemaVersion
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if rename succeeded
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

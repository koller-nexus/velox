package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// config must never touch the network; a locator that errors when used proves it.
func TestConfigJSONIsValidAndOffline(t *testing.T) {
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{err: errors.New("network must not be used")}, &fakeRunner{}, fakeConsent{})
	if code := a.Run(context.Background(), []string{"config", "--json"}); code != ExitOK {
		t.Fatalf("exit = %d, stderr=%s", code, errw.String())
	}
	var v map[string]any
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatalf("config --json not valid JSON: %v\n%s", err, out.String())
	}
	for _, key := range []string{"configPath", "configDir", "consent"} {
		if _, ok := v[key]; !ok {
			t.Errorf("config --json missing %q", key)
		}
	}
}

func TestConfigHumanReadable(t *testing.T) {
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{err: errors.New("no network")}, &fakeRunner{}, fakeConsent{})
	if code := a.Run(context.Background(), []string{"config"}); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "Config file:") {
		t.Errorf("human config output missing 'Config file:': %q", out.String())
	}
}

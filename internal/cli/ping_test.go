package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestPingReportsLatencyOnlyJSON(t *testing.T) {
	var out, errw bytes.Buffer
	// Locator errors -> fallback server; ping then uses Runner.Latency (fake).
	a := newApp(&out, &errw, fakeLocator{err: context.Canceled}, &fakeRunner{}, fakeConsent{})
	if code := a.Run(context.Background(), []string{"ping", "--json"}); code != ExitOK {
		t.Fatalf("exit = %d, stderr=%s", code, errw.String())
	}

	var res struct {
		Online    bool    `json:"online"`
		LatencyMs float64 `json:"latencyMs"`
		JitterMs  float64 `json:"jitterMs"`
	}
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatalf("ping --json invalid: %v\n%s", err, out.String())
	}
	if !res.Online || res.LatencyMs != 9.9 || res.JitterMs != 1.1 {
		t.Errorf("unexpected ping result: %+v", res)
	}
	if strings.ContainsRune(out.String(), '\x1b') {
		t.Errorf("stdout must not contain escape sequences: %q", out.String())
	}
}

func TestPingServerOverrideHonored(t *testing.T) {
	run := &fakeRunner{}
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{err: context.Canceled}, run, fakeConsent{})
	if code := a.Run(context.Background(), []string{"ping", "--server", "wss://h/ndt/v7/download"}); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if run.gotServer.DownloadURL != "wss://h/ndt/v7/download" {
		t.Errorf("--server override not applied: %+v", run.gotServer)
	}
}

func TestPingHumanShowsLatencyNotThroughput(t *testing.T) {
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{err: context.Canceled}, &fakeRunner{}, fakeConsent{})
	if code := a.Run(context.Background(), []string{"ping"}); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	s := out.String()
	if !strings.Contains(s, "Latency:") {
		t.Errorf("ping human output should show Latency: %q", s)
	}
	if strings.Contains(s, "Download") || strings.Contains(s, "Upload") {
		t.Errorf("ping must not report throughput: %q", s)
	}
}

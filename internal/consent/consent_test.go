package consent

import (
	"bytes"
	"testing"
	"time"

	"github.com/koller-nexus/velox/internal/config"
)

// memStore builds a Store backed by an in-memory config plus a save spy.
func memStore(initial config.Config) (*Store, *config.Config, *int) {
	cur := initial
	saves := 0
	s := &Store{
		load: func() (config.Config, error) { return cur, nil },
		save: func(c config.Config) error { cur = c; saves++; return nil },
		now:  func() time.Time { return time.Unix(0, 0).UTC() },
	}
	return s, &cur, &saves
}

func TestResolveStoredGrantedNoPrompt(t *testing.T) {
	s, _, saves := memStore(config.Config{Consent: config.Consent{Decision: config.DecisionGranted}})
	var errw bytes.Buffer
	// nil files => non-interactive, but stored decision short-circuits first.
	d, err := s.Resolve(nil, nil, &errw)
	if err != nil {
		t.Fatal(err)
	}
	if d != config.DecisionGranted {
		t.Errorf("decision = %q, want granted", d)
	}
	if errw.Len() != 0 {
		t.Errorf("should not prompt when decision stored; wrote %q", errw.String())
	}
	if *saves != 0 {
		t.Errorf("stored decision should not re-save")
	}
}

func TestResolveUnsetNonInteractiveDeniesWithoutPersist(t *testing.T) {
	s, _, saves := memStore(config.Config{Consent: config.Consent{Decision: config.DecisionUnset}})
	var errw bytes.Buffer
	d, err := s.Resolve(nil, nil, &errw) // nil files => non-interactive
	if err != nil {
		t.Fatal(err)
	}
	if d != config.DecisionDenied {
		t.Errorf("decision = %q, want denied (non-interactive default)", d)
	}
	if *saves != 0 {
		t.Errorf("non-interactive default must NOT persist (SC-004)")
	}
}

func TestSetAndReset(t *testing.T) {
	s, cur, _ := memStore(config.Default())
	if err := s.Set(config.DecisionGranted); err != nil {
		t.Fatal(err)
	}
	if cur.Consent.Decision != config.DecisionGranted {
		t.Errorf("Set didn't persist granted")
	}
	if cur.Consent.DecidedAt == nil {
		t.Errorf("Set should stamp DecidedAt")
	}
	if err := s.Reset(); err != nil {
		t.Fatal(err)
	}
	if cur.Consent.Decision != config.DecisionUnset {
		t.Errorf("Reset should clear to unset")
	}
}

func TestPromptYesNo(t *testing.T) {
	cases := map[string]config.Decision{
		"y\n":     config.DecisionGranted,
		"yes\n":   config.DecisionGranted,
		"n\n":     config.DecisionDenied,
		"\n":      config.DecisionDenied,
		"garbage": config.DecisionDenied,
	}
	for input, want := range cases {
		r, w, err := osPipe(t)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.WriteString(input)
		_ = w.Close()
		var errw bytes.Buffer
		if got := prompt(r, &errw); got != want {
			t.Errorf("prompt(%q) = %q, want %q", input, got, want)
		}
		_ = r.Close()
	}
}

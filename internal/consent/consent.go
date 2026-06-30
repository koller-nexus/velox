// Package consent manages the user's location-consent decision: prompting,
// persistence, reset, and the safe non-interactive default.
package consent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/koller-nexus/velox/internal/config"
)

// Store loads and persists the consent decision via the config package.
type Store struct {
	load func() (config.Config, error)
	save func(config.Config) error
	now  func() time.Time
}

// NewStore returns a Store backed by the on-disk config file.
func NewStore() *Store {
	return &Store{load: config.Load, save: config.Save, now: time.Now}
}

// Decision returns the currently stored decision.
func (s *Store) Decision() (config.Decision, error) {
	c, err := s.load()
	if err != nil {
		return config.DecisionUnset, err
	}
	return c.Consent.Decision, nil
}

// Set persists a decision with the current timestamp.
func (s *Store) Set(d config.Decision) error {
	c, err := s.load()
	if err != nil {
		return err
	}
	now := s.now().UTC()
	c.Consent = config.Consent{Decision: d, DecidedAt: &now}
	return s.save(c)
}

// Reset clears any stored decision back to unset (FR-006).
func (s *Store) Reset() error {
	c, err := s.load()
	if err != nil {
		return err
	}
	c.Consent = config.Consent{Decision: config.DecisionUnset}
	return s.save(c)
}

// Resolve returns the effective consent decision for the current run, prompting
// the user when the stored decision is unset and the session is interactive.
//
//   - granted/denied stored  -> returned as-is, no prompt (FR-005).
//   - unset + interactive    -> prompt; the answer is persisted (FR-004).
//   - unset + non-interactive-> treated as denied for this run, NOT persisted
//     (FR-007, SC-004).
//
// in/out/errw are injectable for testing; pass os.Stdin/os.Stdout/os.Stderr in
// production.
func (s *Store) Resolve(in, out *os.File, errw io.Writer) (config.Decision, error) {
	d, err := s.Decision()
	if err != nil {
		return config.DecisionUnset, err
	}
	if d == config.DecisionGranted || d == config.DecisionDenied {
		return d, nil
	}
	if !isInteractive(in, out) {
		// No way to ask; default to denied for this run without persisting.
		return config.DecisionDenied, nil
	}
	answer := prompt(in, errw)
	if err := s.Set(answer); err != nil {
		return answer, err
	}
	return answer, nil
}

// prompt asks the user to approve or decline location use. It writes the prompt
// to errw (stderr) so it never pollutes stdout results (Constitution II).
func prompt(in *os.File, errw io.Writer) config.Decision {
	fmt.Fprintln(errw, "velox can pick the nearest test server based on your approximate location.")
	fmt.Fprintln(errw, "This sends your public IP address to a geolocation service over HTTPS.")
	fmt.Fprint(errw, "Allow location lookup? [y/N]: ")

	reader := bufio.NewReader(in)
	line, _ := reader.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return config.DecisionGranted
	default:
		return config.DecisionDenied
	}
}

// Package progress renders a single-line animated loading indicator on a
// terminal. It is standard-library only and safe to use when the target stream
// is not an interactive terminal: in that case the indicator is disabled and
// every method is a no-op that writes nothing, so piped/redirected output and
// machine-readable results are never corrupted.
package progress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// spinnerFrames is the animation glyph cycle (Braille dots).
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const (
	// repaintInterval keeps the indicator updating at ~10 fps (>= 4/s per SC-001).
	repaintInterval = 100 * time.Millisecond
	hideCursor      = "\x1b[?25l"
	showCursor      = "\x1b[?25h"
	clearLine       = "\r\x1b[K" // carriage return + erase to end of line
)

// Indicator is an animated, single-line loading indicator. The zero value is
// not usable; construct one with New. A disabled Indicator (non-terminal
// target) is a valid no-op.
type Indicator struct {
	w       io.Writer
	enabled bool

	mu    sync.Mutex
	label string
	start time.Time

	done     chan struct{}
	wg       sync.WaitGroup
	started  bool
	stopOnce sync.Once
}

// New returns an Indicator that writes to w. It animates only when f is an
// interactive terminal (see IsTerminal); otherwise it is a disabled no-op.
func New(w io.Writer, f *os.File) *Indicator {
	return &Indicator{w: w, enabled: IsTerminal(f)}
}

// newForTest builds an enabled Indicator writing to w, bypassing terminal
// detection. Used by tests to exercise the animation without a real TTY.
func newForTest(w io.Writer) *Indicator {
	return &Indicator{w: w, enabled: true}
}

// IsTerminal reports whether f can display the animated indicator: f must be a
// character device, TERM must not be empty or "dumb", and NO_COLOR must be
// unset. A nil file is never a terminal.
func IsTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	isChar := fi.Mode()&os.ModeCharDevice != 0
	return interactive(isChar, os.Getenv("TERM"), os.Getenv("NO_COLOR"))
}

// interactive is the pure terminal-eligibility rule, split out for testing.
func interactive(isCharDevice bool, term, noColor string) bool {
	if noColor != "" || term == "" || term == "dumb" {
		return false
	}
	return isCharDevice
}

// Start begins the animation. It is a no-op when disabled or already started.
func (i *Indicator) Start() {
	if !i.enabled || i.started {
		return
	}
	i.started = true
	i.start = time.Now()
	i.done = make(chan struct{})
	i.write(hideCursor)
	i.wg.Add(1)
	go i.loop()
}

func (i *Indicator) loop() {
	defer i.wg.Done()
	ticker := time.NewTicker(repaintInterval)
	defer ticker.Stop()
	n := 0
	i.paint(n) // paint immediately so the indicator shows within <1s (SC-001)
	for {
		select {
		case <-i.done:
			return
		case <-ticker.C:
			n++
			i.paint(n)
		}
	}
}

func (i *Indicator) paint(n int) {
	i.mu.Lock()
	label := i.label
	elapsed := time.Since(i.start)
	i.mu.Unlock()
	glyph := spinnerFrames[n%len(spinnerFrames)]
	i.write(clearLine + frame(glyph, label, elapsed))
}

// write emits s to the target stream, ignoring write errors (a failed write to
// a progress stream is not actionable and must never disrupt the measurement).
func (i *Indicator) write(s string) {
	_, _ = io.WriteString(i.w, s)
}

// SetPhase updates the label shown by the indicator. It is safe to call before
// Start and after Stop, and is a no-op when disabled.
func (i *Indicator) SetPhase(label string) {
	if !i.enabled {
		return
	}
	i.mu.Lock()
	i.label = label
	i.mu.Unlock()
}

// Stop halts the animation, clears the line, and restores the cursor. It is
// idempotent and a no-op when disabled.
func (i *Indicator) Stop() {
	if !i.enabled {
		return
	}
	i.stopOnce.Do(func() {
		if i.done != nil {
			close(i.done)
			i.wg.Wait()
		}
		i.write(clearLine + showCursor)
	})
}

// frame renders the indicator's single line, e.g. "⠋ measuring download… 12s".
func frame(glyph, label string, elapsed time.Duration) string {
	secs := int(elapsed.Seconds())
	if label == "" {
		return fmt.Sprintf("%s %ds", glyph, secs)
	}
	return fmt.Sprintf("%s %s %ds", glyph, label, secs)
}

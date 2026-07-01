package progress

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestFrame(t *testing.T) {
	tests := []struct {
		name    string
		glyph   string
		label   string
		elapsed time.Duration
		want    string
	}{
		{"zero seconds with label", "⠋", "measuring download…", 0, "⠋ measuring download… 0s"},
		{"five seconds", "⠙", "measuring upload…", 5 * time.Second, "⠙ measuring upload… 5s"},
		{"past a minute counts total seconds", "⠹", "measuring download…", 65 * time.Second, "⠹ measuring download… 65s"},
		{"empty label omits the label", "⠋", "", 3 * time.Second, "⠋ 3s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := frame(tt.glyph, tt.label, tt.elapsed); got != tt.want {
				t.Errorf("frame() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInteractive(t *testing.T) {
	tests := []struct {
		name    string
		isChar  bool
		term    string
		noColor string
		want    bool
	}{
		{"char device, normal term", true, "xterm-256color", "", true},
		{"dumb terminal suppressed", true, "dumb", "", false},
		{"empty TERM suppressed", true, "", "", false},
		{"NO_COLOR suppressed", true, "xterm", "1", false},
		{"not a char device", false, "xterm", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := interactive(tt.isChar, tt.term, tt.noColor); got != tt.want {
				t.Errorf("interactive(%v, %q, %q) = %v, want %v", tt.isChar, tt.term, tt.noColor, got, tt.want)
			}
		})
	}
}

// A disabled indicator (non-terminal target) must write nothing at all so that
// piped/redirected output and JSON results stay clean (FR-004/SC-003).
func TestDisabledWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	ind := New(&buf, nil) // nil file -> disabled
	ind.Start()
	ind.SetPhase("measuring download…")
	ind.Stop()
	if buf.Len() != 0 {
		t.Errorf("disabled indicator wrote %d bytes: %q", buf.Len(), buf.String())
	}
}

// An enabled indicator must, on Stop, clear its line and restore the cursor,
// leaving no dangling animation; Stop must be idempotent (FR-006/FR-007/SC-005).
func TestEnabledStopRestoresCursorAndIsIdempotent(t *testing.T) {
	var buf bytes.Buffer
	ind := newForTest(&buf)
	ind.SetPhase("measuring download…")
	ind.Start()
	ind.Stop()

	out := buf.String()
	if !strings.Contains(out, hideCursor) {
		t.Errorf("expected cursor-hide sequence in output: %q", out)
	}
	if !strings.Contains(out, "measuring download…") {
		t.Errorf("expected phase label in output: %q", out)
	}
	if !strings.HasSuffix(out, clearLine+showCursor) {
		t.Errorf("output must end by clearing the line and restoring the cursor: %q", out)
	}

	before := buf.Len()
	ind.Stop() // second Stop must not panic nor write again
	if buf.Len() != before {
		t.Errorf("second Stop wrote additional bytes: %q", buf.String()[before:])
	}
}

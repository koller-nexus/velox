package cli

import (
	"testing"

	"github.com/koller-nexus/velox/internal/speedtest"
)

func TestPhaseLabel(t *testing.T) {
	tests := []struct {
		phase speedtest.Phase
		want  string
	}{
		{speedtest.PhaseConnectivity, "checking connectivity…"},
		{speedtest.PhaseDownload, "measuring download…"},
		{speedtest.PhaseUpload, "measuring upload…"},
	}
	for _, tt := range tests {
		if got := phaseLabel(tt.phase); got != tt.want {
			t.Errorf("phaseLabel(%q) = %q, want %q", tt.phase, got, tt.want)
		}
	}
}

func TestIndicatorEnabled(t *testing.T) {
	tests := []struct {
		name                       string
		isTTY, noProgress, verbose bool
		want                       bool
	}{
		{"tty, no opt-outs", true, false, false, true},
		{"not a tty", false, false, false, false},
		{"no-progress disables on tty", true, true, false, false},
		{"verbose disables on tty", true, false, true, false},
		{"both opt-outs", true, true, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indicatorEnabled(tt.isTTY, tt.noProgress, tt.verbose); got != tt.want {
				t.Errorf("indicatorEnabled(%v,%v,%v) = %v, want %v",
					tt.isTTY, tt.noProgress, tt.verbose, got, tt.want)
			}
		})
	}
}

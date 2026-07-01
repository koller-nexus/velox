package cli

import (
	"github.com/koller-nexus/velox/internal/progress"
	"github.com/koller-nexus/velox/internal/speedtest"
)

// phaseLabel maps a measurement phase to the label shown by the loading
// indicator. Latency is measured within the download phase and is not shown
// separately.
func phaseLabel(p speedtest.Phase) string {
	switch p {
	case speedtest.PhaseConnectivity:
		return "checking connectivity…"
	case speedtest.PhaseDownload:
		return "measuring download…"
	case speedtest.PhaseUpload:
		return "measuring upload…"
	default:
		return string(p)
	}
}

// phaseReporter adapts speedtest phase events to the loading indicator.
// It implements speedtest.Reporter.
type phaseReporter struct{ ind *progress.Indicator }

func (r phaseReporter) Phase(p speedtest.Phase) {
	r.ind.SetPhase(phaseLabel(p))
}

// indicatorEnabled reports whether the loading indicator should animate: only
// on an interactive terminal, and never under --no-progress or --verbose (in
// verbose mode the text diagnostics narrate progress instead — FR-009/FR-011).
func indicatorEnabled(isTTY, noProgress, verbose bool) bool {
	return isTTY && !noProgress && !verbose
}

// newIndicator builds the loading indicator for a run, honoring the interactive
// opt-outs. When disabled it returns a no-op indicator that writes nothing.
func (a *App) newIndicator(noProgress, verbose bool) *progress.Indicator {
	if !indicatorEnabled(progress.IsTerminal(a.StderrF), noProgress, verbose) {
		return progress.New(a.Stderr, nil)
	}
	return progress.New(a.Stderr, a.StderrF)
}

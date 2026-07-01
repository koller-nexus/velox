package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/koller-nexus/velox/internal/config"
)

const configUsage = `velox config — show where velox stores its settings and their values.

USAGE:
  velox config [--json]

Prints the config file path, the consent decision, and effective settings.
Read-only; performs no network access.
`

type consentView struct {
	Decision  string     `json:"decision"`
	DecidedAt *time.Time `json:"decidedAt,omitempty"`
}

type fallbackServerView struct {
	Machine     string `json:"machine,omitempty"`
	DownloadURL string `json:"downloadURL,omitempty"`
	UploadURL   string `json:"uploadURL,omitempty"`
}

type configView struct {
	ConfigPath     string              `json:"configPath"`
	ConfigDir      string              `json:"configDir"`
	Consent        consentView         `json:"consent"`
	GeoEndpoint    string              `json:"geoEndpoint,omitempty"`
	FallbackServer *fallbackServerView `json:"fallbackServer,omitempty"`
}

// configCommand implements `velox config` (FR-012): a read-only, offline view of
// where velox stores its state and the effective settings.
func (a *App) configCommand() *Command {
	return &Command{
		Name:    "config",
		Summary: "Show where velox stores its settings and their values",
		Usage:   configUsage,
		Run: func(_ context.Context, args []string) int {
			fs := flag.NewFlagSet("config", flag.ContinueOnError)
			asJSON := fs.Bool("json", false, "emit machine-readable JSON")
			if code, handled := a.parseCommandFlags(fs, configUsage, args); handled {
				return code
			}

			path, err := config.Path()
			if err != nil {
				fmt.Fprintf(a.Stderr, "velox: %v\n", err)
				return ExitFailure
			}
			cfg := config.Default()
			if a.LoadConfig != nil {
				if c, lerr := a.LoadConfig(); lerr == nil {
					cfg = c
				}
			}
			view := configView{
				ConfigPath:  path,
				ConfigDir:   filepath.Dir(path),
				Consent:     consentView{Decision: string(cfg.Consent.Decision), DecidedAt: cfg.Consent.DecidedAt},
				GeoEndpoint: cfg.GeoEndpoint,
			}
			if cfg.FallbackServer != nil {
				view.FallbackServer = &fallbackServerView{
					Machine:     cfg.FallbackServer.Machine,
					DownloadURL: cfg.FallbackServer.DownloadURL,
					UploadURL:   cfg.FallbackServer.UploadURL,
				}
			}

			if *asJSON {
				enc := json.NewEncoder(a.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(view); err != nil {
					fmt.Fprintf(a.Stderr, "velox: render error: %v\n", err)
					return ExitFailure
				}
				return ExitOK
			}
			renderConfigHuman(a.Stdout, view)
			return ExitOK
		},
	}
}

func renderConfigHuman(w io.Writer, v configView) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "Config file:\t%s\n", v.ConfigPath)
	fmt.Fprintf(tw, "Config dir:\t%s\n", v.ConfigDir)
	decided := "—"
	if v.Consent.DecidedAt != nil {
		decided = v.Consent.DecidedAt.Format(time.RFC3339)
	}
	fmt.Fprintf(tw, "Consent:\t%s (decided %s)\n", v.Consent.Decision, decided)
	geo := v.GeoEndpoint
	if geo == "" {
		geo = "(default)"
	}
	fmt.Fprintf(tw, "Geo endpoint:\t%s\n", geo)
	if v.FallbackServer != nil {
		fmt.Fprintf(tw, "Fallback server:\t%s\n", v.FallbackServer.Machine)
	}
	_ = tw.Flush()
}

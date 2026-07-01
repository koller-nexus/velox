package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/koller-nexus/velox/internal/config"
	"github.com/koller-nexus/velox/internal/geo"
)

const serversUsage = `velox servers — list the nearest test servers velox would use.

USAGE:
  velox servers [--json] [--timeout <dur>]

Shows the server velox would select plus the nearest alternatives (about 5),
without running a measurement. Uses your approximate location only if you have
granted consent; otherwise it falls back to registry proximity order.
`

const maxServerList = 5

type serverEntry struct {
	Machine    string   `json:"machine"`
	Site       string   `json:"site"`
	City       string   `json:"city,omitempty"`
	Country    string   `json:"country,omitempty"`
	DistanceKm *float64 `json:"distanceKm"`
	Selected   bool     `json:"selected"`
}

type serverListing struct {
	LocationUsed bool          `json:"locationUsed"`
	Servers      []serverEntry `json:"servers"`
}

// serversCommand implements `velox servers` (FR-011): list the ~5 nearest
// candidate servers velox would pick, honoring the consent gate.
func (a *App) serversCommand() *Command {
	return &Command{
		Name:    "servers",
		Summary: "List the nearest test servers velox would use",
		Usage:   serversUsage,
		Run: func(ctx context.Context, args []string) int {
			fs := flag.NewFlagSet("servers", flag.ContinueOnError)
			asJSON := fs.Bool("json", false, "emit machine-readable JSON")
			timeout := fs.Duration("timeout", 10*time.Second, "overall budget")
			if code, handled := a.parseCommandFlags(fs, serversUsage, args); handled {
				return code
			}

			ctx, cancel := context.WithTimeout(ctx, *timeout)
			defer cancel()

			listing, err := a.listServers(ctx)
			if err != nil {
				fmt.Fprintf(a.Stderr, "velox: %v\n", err)
				return ExitFailure
			}
			if *asJSON {
				enc := json.NewEncoder(a.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(listing); err != nil {
					fmt.Fprintf(a.Stderr, "velox: render error: %v\n", err)
					return ExitFailure
				}
				return ExitOK
			}
			renderServersHuman(a.Stdout, listing)
			return ExitOK
		},
	}
}

func (a *App) listServers(ctx context.Context) (serverListing, error) {
	candidates, err := a.Locator.Nearest(ctx)
	if err != nil {
		return serverListing{}, fmt.Errorf("discover servers: %w", err)
	}
	if len(candidates) == 0 {
		return serverListing{}, errors.New("no test servers available")
	}

	var est *geo.LocationEstimate
	locationUsed := false
	if a.resolveConsent() == config.DecisionGranted {
		if e, gerr := a.NewResolver(a.geoEndpoint()).Resolve(ctx); gerr == nil {
			est = &e
			locationUsed = true
		}
	}

	ranked := geo.RankByDistance(candidates, est)
	if len(ranked) > maxServerList {
		ranked = ranked[:maxServerList]
	}
	out := serverListing{LocationUsed: locationUsed, Servers: make([]serverEntry, 0, len(ranked))}
	for i, sel := range ranked {
		out.Servers = append(out.Servers, serverEntry{
			Machine:    sel.Server.Machine,
			Site:       sel.Server.SiteCode,
			City:       sel.Server.City,
			Country:    sel.Server.Country,
			DistanceKm: sel.DistanceKm,
			Selected:   i == 0,
		})
	}
	return out, nil
}

func renderServersHuman(w io.Writer, l serverListing) {
	if len(l.Servers) == 0 {
		fmt.Fprintln(w, "No servers found.")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  \tSERVER\tLOCATION\tDISTANCE")
	for _, s := range l.Servers {
		marker := " "
		if s.Selected {
			marker = "*"
		}
		loc := "—"
		switch {
		case s.City != "" && s.Country != "":
			loc = s.City + ", " + s.Country
		case s.Country != "":
			loc = s.Country
		case s.City != "":
			loc = s.City
		}
		dist := "—"
		if s.DistanceKm != nil {
			dist = fmt.Sprintf("%.0f km", *s.DistanceKm)
		}
		name := s.Machine
		if s.Site != "" {
			name = s.Site
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", marker, name, loc, dist)
	}
	_ = tw.Flush()
	if l.LocationUsed {
		fmt.Fprintln(w, "\n('*' = server velox would select)")
	} else {
		fmt.Fprintln(w, "\n(location not used — ranked by registry proximity; '*' = default server)")
	}
}

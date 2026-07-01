package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/koller-nexus/velox/internal/version"
)

const versionUsage = `velox version — print the velox version.

USAGE:
  velox version

Prints the same version line as 'velox --version'. Requires no network.
`

// versionCommand implements `velox version` (FR-010): the subcommand form of
// --version. Plain text only; offline.
func (a *App) versionCommand() *Command {
	return &Command{
		Name:    "version",
		Summary: "Print the velox version",
		Usage:   versionUsage,
		Run: func(_ context.Context, args []string) int {
			fs := flag.NewFlagSet("version", flag.ContinueOnError)
			if code, handled := a.parseCommandFlags(fs, versionUsage, args); handled {
				return code
			}
			fmt.Fprintln(a.Stdout, version.String())
			return ExitOK
		},
	}
}

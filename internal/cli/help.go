package cli

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"
)

// usage text for the top-level overview. It performs no network or consent work
// (FR-008); the command list is generated from the registry (FR-007).
const overviewIntro = `velox — measure your internet speed against the nearest open test server.

USAGE:
  velox --check-internet [flags]   Run a full speed test.
  velox <command> [flags]
`

const overviewOutro = `
FLAGS (velox --check-internet):
  --json            Emit machine-readable JSON.
  --server <url>    Override the test server (ndt7 service URL).
  --timeout <dur>   Overall run budget (default 60s).
  --no-progress     Disable the loading indicator.
  -v, --verbose     Verbose diagnostics on stderr.

Run 'velox help <command>' for details on a specific command.
`

const helpUsage = `velox help — show help for velox or a specific command.

USAGE:
  velox help [command]

With no command, prints an overview of every command. With a command name,
prints that command's detailed help (identical to 'velox <command> --help').
Performs no network access.
`

// printOverview writes the top-level help: intro, the command list built from
// the registry (so it always matches the real command set — FR-007), and the
// speed-test flags. Output goes to stdout; no network or consent work (FR-008).
func (a *App) printOverview() {
	var b strings.Builder
	b.WriteString(overviewIntro)
	b.WriteString("\nCOMMANDS:\n")
	tw := tabwriter.NewWriter(&b, 0, 0, 3, ' ', 0)
	for _, c := range a.commands() {
		fmt.Fprintf(tw, "  %s\t%s\n", c.Name, c.Summary)
	}
	_ = tw.Flush()
	b.WriteString(overviewOutro)
	fmt.Fprint(a.Stdout, b.String())
}

// helpCommand implements `velox help [command]` (FR-001/FR-002). It is offline.
func (a *App) helpCommand() *Command {
	return &Command{
		Name:    "help",
		Summary: "Show help for velox or a specific command",
		Usage:   helpUsage,
		Run: func(_ context.Context, args []string) int {
			if len(args) == 0 {
				a.printOverview()
				return ExitOK
			}
			name := args[0]
			if name == "-h" || name == "--help" {
				fmt.Fprint(a.Stdout, helpUsage)
				return ExitOK
			}
			if cmd := a.lookup(name); cmd != nil {
				fmt.Fprint(a.Stdout, cmd.Usage)
				return ExitOK
			}
			fmt.Fprintf(a.Stderr, "velox: unknown command %q; run 'velox help'\n", name)
			return ExitUsage
		},
	}
}

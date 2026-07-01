package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
)

// Command is a registry entry: an invocable velox subcommand. The registry is
// the single source of truth for dispatch and for the `velox help` overview, so
// every command stays discoverable (FR-007).
type Command struct {
	Name    string
	Summary string // one-line description shown in the overview
	Usage   string // detailed help printed by `help <name>` and `<name> --help`
	Run     func(ctx context.Context, args []string) int
}

// commands returns the registry in display order.
func (a *App) commands() []*Command {
	return []*Command{
		a.helpCommand(),
		a.versionCommand(),
		a.serversCommand(),
		a.pingCommand(),
		a.configCommand(),
		a.consentCommand(),
	}
}

// lookup returns the command with the given name, or nil.
func (a *App) lookup(name string) *Command {
	for _, c := range a.commands() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// dispatch runs the subcommand named by args[0]. An unknown name is a usage
// error that points the user at `velox help` (FR-004).
func (a *App) dispatch(ctx context.Context, args []string) int {
	name := args[0]
	cmd := a.lookup(name)
	if cmd == nil {
		fmt.Fprintf(a.Stderr, "velox: unknown command %q; run 'velox help'\n", name)
		return ExitUsage
	}
	return cmd.Run(ctx, args[1:])
}

// parseCommandFlags parses a subcommand's flags with uniform help/error
// handling. When handled is true the caller must return code: either help was
// requested (--help/-h → Usage to stdout, ExitOK) or an unknown flag was given
// (→ stderr error + `velox help` hint, ExitUsage) (FR-003/FR-004/FR-006).
func (a *App) parseCommandFlags(fs *flag.FlagSet, usage string, args []string) (code int, handled bool) {
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(a.Stdout, usage)
			return ExitOK, true
		}
		fmt.Fprintf(a.Stderr, "velox %s: %v; run 'velox help %s'\n", fs.Name(), err, fs.Name())
		return ExitUsage, true
	}
	return 0, false
}

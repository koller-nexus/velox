// Command velox measures internet speed against the nearest open test server.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/koller-nexus/velox/internal/cli"
)

func main() {
	// Root context cancelled on Ctrl-C / SIGTERM so in-flight network work is
	// torn down cleanly (FR-011).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := cli.NewApp()
	os.Exit(app.Run(ctx, os.Args[1:]))
}

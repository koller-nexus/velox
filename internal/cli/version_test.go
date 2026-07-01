package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestVersionSubcommandMatchesFlag(t *testing.T) {
	var o1, e1, o2, e2 bytes.Buffer
	a1 := newApp(&o1, &e1, fakeLocator{err: context.Canceled}, &fakeRunner{}, fakeConsent{})
	a2 := newApp(&o2, &e2, fakeLocator{err: context.Canceled}, &fakeRunner{}, fakeConsent{})

	if c := a1.Run(context.Background(), []string{"version"}); c != ExitOK {
		t.Fatalf("version exit = %d", c)
	}
	if c := a2.Run(context.Background(), []string{"--version"}); c != ExitOK {
		t.Fatalf("--version exit = %d", c)
	}
	if o1.String() != o2.String() {
		t.Errorf("version subcommand != --version flag: %q vs %q", o1.String(), o2.String())
	}
	if !strings.HasPrefix(o1.String(), "velox ") {
		t.Errorf("version output = %q", o1.String())
	}
}

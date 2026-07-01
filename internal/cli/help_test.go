package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHelpOverviewListsEveryCommand(t *testing.T) {
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{}, &fakeRunner{}, fakeConsent{})
	if code := a.Run(context.Background(), []string{"help"}); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	got := out.String()
	if !strings.Contains(got, "USAGE") {
		t.Errorf("overview missing USAGE section")
	}
	for _, c := range a.commands() {
		if !strings.Contains(got, c.Name) {
			t.Errorf("overview missing command %q", c.Name)
		}
		if !strings.Contains(got, c.Summary) {
			t.Errorf("overview missing summary for %q", c.Name)
		}
	}
}

func TestHelpForCommandPrintsUsage(t *testing.T) {
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{}, &fakeRunner{}, fakeConsent{})
	if code := a.Run(context.Background(), []string{"help", "servers"}); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if out.String() != serversUsage {
		t.Errorf("help servers should print serversUsage, got %q", out.String())
	}
}

func TestHelpUnknownCommandIsUsageError(t *testing.T) {
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{}, &fakeRunner{}, fakeConsent{})
	if code := a.Run(context.Background(), []string{"help", "nope"}); code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(errw.String(), "velox help") {
		t.Errorf("stderr should suggest 'velox help': %q", errw.String())
	}
}

// Every command's --help prints the same usage as `help <command>`, exits 0,
// and touches no network (FR-003/SC-002). The fakeLocator here would error if
// used, proving the help path is side-effect free.
func TestCommandHelpFlagMatchesHelpCommand(t *testing.T) {
	for _, name := range []string{"version", "servers", "ping", "config", "consent"} {
		t.Run(name, func(t *testing.T) {
			var o1, e1, o2, e2 bytes.Buffer
			a1 := newApp(&o1, &e1, fakeLocator{err: context.Canceled}, &fakeRunner{}, fakeConsent{})
			a2 := newApp(&o2, &e2, fakeLocator{err: context.Canceled}, &fakeRunner{}, fakeConsent{})
			if c := a1.Run(context.Background(), []string{name, "--help"}); c != ExitOK {
				t.Fatalf("%s --help exit = %d", name, c)
			}
			if c := a2.Run(context.Background(), []string{"help", name}); c != ExitOK {
				t.Fatalf("help %s exit = %d", name, c)
			}
			if o1.String() != o2.String() {
				t.Errorf("%q: '--help' != 'help %s'\n%q\nvs\n%q", name, name, o1.String(), o2.String())
			}
			if o1.Len() == 0 {
				t.Errorf("%q --help produced no usage text", name)
			}
		})
	}
}

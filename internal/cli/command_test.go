package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestDispatchUnknownCommand(t *testing.T) {
	var out, errw bytes.Buffer
	a := newApp(&out, &errw, fakeLocator{}, &fakeRunner{}, fakeConsent{})
	code := a.Run(context.Background(), []string{"frobnicate"})
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(errw.String(), "unknown command") || !strings.Contains(errw.String(), "velox help") {
		t.Errorf("stderr should name the problem and suggest 'velox help': %q", errw.String())
	}
	if out.Len() != 0 {
		t.Errorf("nothing should go to stdout on a usage error: %q", out.String())
	}
}

func TestBackwardCompatibleRootRouting(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
		out  string
	}{
		{"bare", nil, ExitOK, "USAGE"},
		{"help flag", []string{"--help"}, ExitOK, "USAGE"},
		{"version flag", []string{"--version"}, ExitOK, "velox "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errw bytes.Buffer
			a := newApp(&out, &errw, fakeLocator{}, &fakeRunner{}, fakeConsent{})
			if code := a.Run(context.Background(), tc.args); code != tc.want {
				t.Fatalf("exit = %d, want %d", code, tc.want)
			}
			if !strings.Contains(out.String(), tc.out) {
				t.Errorf("stdout = %q, want contains %q", out.String(), tc.out)
			}
		})
	}
}

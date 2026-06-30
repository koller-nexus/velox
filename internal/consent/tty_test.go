package consent

import (
	"os"
	"path/filepath"
	"testing"
)

// osPipe returns a connected read/write file pair for feeding prompt input.
func osPipe(t *testing.T) (*os.File, *os.File, error) {
	t.Helper()
	return os.Pipe()
}

func TestIsCharDeviceFalseForRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if isCharDevice(f) {
		t.Error("regular file should not be a char device")
	}
}

func TestIsCharDeviceFalseForNil(t *testing.T) {
	if isCharDevice(nil) {
		t.Error("nil file should not be a char device")
	}
}

func TestIsInteractiveFalseForPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if isInteractive(r, w) {
		t.Error("pipes are not interactive terminals")
	}
}

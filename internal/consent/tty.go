package consent

import "os"

// IsTerminal reports whether f refers to a character device (terminal).
func IsTerminal(f *os.File) bool {
	return isCharDevice(f)
}

// isInteractive reports whether both stdin and stdout are terminals, so a
// consent prompt can be shown and answered. Used to decide the non-interactive
// default (decline) per FR-007 / SC-004.
func isInteractive(in, out *os.File) bool {
	return isCharDevice(in) && isCharDevice(out)
}

func isCharDevice(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

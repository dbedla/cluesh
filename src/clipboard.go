// Package cluesh — clipboard: copy the final command via platform tools.
package cluesh

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// ShouldCopy is the clipboard gate for put_cmd_in_clipboard. The mode is
// pre-validated by LoadConfig; read-only copies only commands that do not
// modify any file (ReadOnly).
func ShouldCopy(mode string, readOnly bool) bool {
	switch mode {
	case "always":
		return true
	case "read-only":
		return readOnly
	}
	return false // never
}

// CopyToClipboard shells out to the platform clipboard tool, feeding text via
// stdin. Linux prefers wl-copy (Wayland) and falls back to xclip (X11); the
// tools themselves keep the selection alive after cluesh exits.
func CopyToClipboard(text string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name = "pbcopy"
	case "windows":
		name = "clip.exe"
	default:
		if _, err := exec.LookPath("wl-copy"); err == nil {
			name = "wl-copy"
		} else if _, err := exec.LookPath("xclip"); err == nil {
			name = "xclip"
			args = []string{"-selection", "clipboard"}
		} else {
			return errors.New("no clipboard tool found — install wl-copy or xclip")
		}
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

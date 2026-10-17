package main

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// /dev/null is a character device, which the old ModeCharDevice heuristic
// wrongly reported as a terminal; the isatty-based check must not.
func TestIsTerminalCharDevice(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /dev/null on windows")
	}
	f, err := os.Open("/dev/null")
	if err != nil {
		t.Skip("no /dev/null available:", err)
	}
	defer f.Close()
	assert.False(t, isTerminal(f), "/dev/null must not be detected as a terminal")
}

func TestExplainMessage(t *testing.T) {
	got := explainMessage("ls -l")
	assert.True(t, strings.HasPrefix(got, "Explain this command; do not generate a new one:\n"),
		"mode-selecting prefix must lead the message, got %q", got)
	assert.True(t, strings.HasSuffix(got, "ls -l"),
		"command must arrive intact after the prefix, got %q", got)
}

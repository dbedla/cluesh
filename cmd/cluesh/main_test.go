package main

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	defer func() {
		err = f.Close()
		assert.NoError(t, err)
	}()

	assert.False(t, isTerminal(f), "/dev/null must not be detected as a terminal")

}

func TestExplainFlowInputLimit(t *testing.T) {
	const limit = 16
	// over the limit via argument path (shared len check)
	_, err := promptForExplainFlow(strings.Repeat("a", limit+1), limit)
	assert.Error(t, err, "input over the limit must be refused")
	assert.Contains(t, err.Error(), "input_limit_in_bytes")

	// over the limit via stdin (LimitReader path; os.Stdin is a file, not a
	// terminal, so the stdin branch is taken)
	f, ferr := os.CreateTemp(t.TempDir(), "oversized*")
	require.NoError(t, ferr)
	_, werr := f.WriteString(strings.Repeat("a", limit+1))
	require.NoError(t, werr)
	require.NoError(t, f.Close())
	f, ferr = os.Open(f.Name())
	require.NoError(t, ferr)

	defer func() {
		err = f.Close()
		assert.NoError(t, err)
	}()

	orig := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = orig }()

	_, err = promptForExplainFlow("", limit)
	assert.Error(t, err, "stdin input over the limit must be refused")
	assert.Contains(t, err.Error(), "input_limit_in_bytes")
}

func TestDemandInputLimit(t *testing.T) {
	const limit = 16
	assert.NoError(t, checkDemandLimit(strings.Repeat("a", limit), limit))
	assert.NoError(t, checkDemandLimit("", limit))

	err := checkDemandLimit(strings.Repeat("a", limit+1), limit)
	require.Error(t, err, "demand over the limit must be refused")
	assert.Contains(t, err.Error(), "input_limit_in_bytes")
}

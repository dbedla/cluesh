package cluesh

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintLayout(t *testing.T) {
	// Print owns ALL layout (newlines, indentation) — this exact string is
	// the contract; the colored modes only recolor the same bytes.
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	Print(NoMode{}, Command{
		FinalCommand: "ls -la",
		SubCommands: []SubCommand{{
			Command:   "ls",
			Arguments: []Argument{{Argument: "-l", Description: "long listing"}},
		}},
		Notes: "note",
	})
	err = w.Close()
	assert.NoError(t, err)
	os.Stdout = old

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "\nls -la\n\n\tls\n\t\t -l\n\t\t  long listing\n\nnote\n\n", string(out))
}

func TestConsoleByName(t *testing.T) {
	// dark / light / none each map to their mode
	assert.IsType(t, &DarkMode{}, ConsoleByName("dark"))
	assert.IsType(t, &LightMode{}, ConsoleByName("light"))
	assert.IsType(t, &NoMode{}, ConsoleByName("none"))

	// unknown value falls back to NoMode, no error
	assert.IsType(t, &NoMode{}, ConsoleByName("nonsense"))
	assert.IsType(t, &NoMode{}, ConsoleByName(""))
}

func TestConsoleByNameNoColor(t *testing.T) {
	// NO_COLOR (https://no-color.org) wins over every config value
	t.Setenv("NO_COLOR", "1")
	assert.IsType(t, &NoMode{}, ConsoleByName("dark"))
	assert.IsType(t, &NoMode{}, ConsoleByName("light"))
	assert.IsType(t, &NoMode{}, ConsoleByName("none"))
	assert.IsType(t, &NoMode{}, ConsoleByName("nonsense"))
}

func TestConsoleByNameNoColorEmpty(t *testing.T) {
	// per no-color.org: empty NO_COLOR is "not set" — config still applies
	t.Setenv("NO_COLOR", "")
	assert.IsType(t, &DarkMode{}, ConsoleByName("dark"))
	assert.IsType(t, &NoMode{}, ConsoleByName("none"))
}

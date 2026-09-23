package cluesh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

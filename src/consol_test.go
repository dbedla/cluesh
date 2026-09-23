package cluesh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsolByName(t *testing.T) {
	// dark / light / none each map to their mode
	assert.IsType(t, &DarkMode{}, ConsolByName("dark"))
	assert.IsType(t, &LightMode{}, ConsolByName("light"))
	assert.IsType(t, &NoMode{}, ConsolByName("none"))

	// unknown value falls back to NoMode, no error
	assert.IsType(t, &NoMode{}, ConsolByName("nonsense"))
	assert.IsType(t, &NoMode{}, ConsolByName(""))
}

func TestConsolByNameNoColor(t *testing.T) {
	// NO_COLOR (https://no-color.org) wins over every config value
	t.Setenv("NO_COLOR", "1")
	assert.IsType(t, &NoMode{}, ConsolByName("dark"))
	assert.IsType(t, &NoMode{}, ConsolByName("light"))
	assert.IsType(t, &NoMode{}, ConsolByName("none"))
	assert.IsType(t, &NoMode{}, ConsolByName("nonsense"))
}

func TestConsolByNameNoColorEmpty(t *testing.T) {
	// per no-color.org: empty NO_COLOR is "not set" — config still applies
	t.Setenv("NO_COLOR", "")
	assert.IsType(t, &DarkMode{}, ConsolByName("dark"))
	assert.IsType(t, &NoMode{}, ConsolByName("none"))
}
package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExplainMessage(t *testing.T) {
	got := explainMessage("ls -l")
	assert.True(t, strings.HasPrefix(got, "Explain this command; do not generate a new one:\n"),
		"mode-selecting prefix must lead the message, got %q", got)
	assert.True(t, strings.HasSuffix(got, "ls -l"),
		"command must arrive intact after the prefix, got %q", got)
}

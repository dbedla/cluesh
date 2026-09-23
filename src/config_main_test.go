package cluesh

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/dbedla/rellm/pkg/rellm"
	"github.com/stretchr/testify/assert"
)

func TestBaseDirInjection(t *testing.T) {
	baseDir := t.TempDir()

	cfg, sysPrompt, err := LoadConfig(baseDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, cfg.DefaultModelTag)
	assert.NotEmpty(t, sysPrompt)
	assert.Equal(t, DefaultTemperature, cfg.Temperature)
	assert.Equal(t, DefaultReasoningEffort, cfg.ReasoningEffort)

	assert.True(t, strings.HasPrefix(ConfigPath(baseDir), baseDir), "config path outside baseDir")
	assert.True(t, strings.HasPrefix(SysPromptPath(baseDir), baseDir), "sysprompt path outside baseDir")
	assert.True(t, strings.HasPrefix(ConversationPath(baseDir), baseDir), "conversation path outside baseDir")

	assert.FileExists(t, ConfigPath(baseDir))
	assert.FileExists(t, SysPromptPath(baseDir))
	// conversation.jsonl is deliberately absent: rellm's FilesystemConversation
	// creates it lazily on first Append — missing file = empty conversation.
}

func TestLoadConfigRejectsBadReasoningEffort(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"unknown value", "bogus"},
		{"empty value", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			_, _, err := LoadConfig(baseDir)
			assert.NoError(t, err)

			// replace the generated "reasoning_effort": "low" with a bad value
			path := ConfigPath(baseDir)
			data, err := os.ReadFile(path)
			assert.NoError(t, err)
			out := strings.Replace(string(data),
				fmt.Sprintf("%q: %q", "reasoning_effort", DefaultReasoningEffort),
				fmt.Sprintf("%q: %q", "reasoning_effort", tt.value), 1)
			assert.NoError(t, os.WriteFile(path, []byte(out), 0o600))

			_, _, err = LoadConfig(baseDir)
			assert.Error(t, err)
		})
	}
}

func TestReasoningEffortMapping(t *testing.T) {
	for s, want := range map[string]rellm.ReasoningEffort{
		"none":   rellm.ReasoningEffortNone,
		"low":    rellm.ReasoningEffortLow,
		"medium": rellm.ReasoningEffortMedium,
		"high":   rellm.ReasoningEffortHigh,
	} {
		got, err := ReasoningEffort(s)
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	}
	_, err := ReasoningEffort("bogus")
	assert.Error(t, err)
	_, err = ReasoningEffort("")
	assert.Error(t, err)
}
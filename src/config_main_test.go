package cluesh

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseDirInjection(t *testing.T) {
	baseDir := t.TempDir()

	cfg, sysPrompt, err := LoadConfig(baseDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, cfg.DefaultModelTag)
	assert.NotEmpty(t, sysPrompt)

	assert.True(t, strings.HasPrefix(ConfigPath(baseDir), baseDir), "config path outside baseDir")
	assert.True(t, strings.HasPrefix(SysPromptPath(baseDir), baseDir), "sysprompt path outside baseDir")
	assert.True(t, strings.HasPrefix(ConversationPath(baseDir), baseDir), "conversation path outside baseDir")

	assert.FileExists(t, ConfigPath(baseDir))
	assert.FileExists(t, SysPromptPath(baseDir))
	// conversation.jsonl is deliberately absent: rellm's FilesystemConversation
	// creates it lazily on first Append — missing file = empty conversation.
}

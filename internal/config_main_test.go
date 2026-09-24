package cluesh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestLoadConfigTimeout(t *testing.T) {
	// write a config with the given timeout value and try to load it
	load := func(baseDir string, minutes int) error {
		_, _, err := LoadConfig(baseDir) // first run: generate defaults
		require.NoError(t, err)
		path := filepath.Join(baseDir, ConfigFileName)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		out := strings.Replace(string(data),
			`"execution_timeout_minutes": 5`,
			fmt.Sprintf(`"execution_timeout_minutes": %d`, minutes), 1)
		require.NoError(t, os.WriteFile(path, []byte(out), 0o600))
		_, _, err = LoadConfig(baseDir)
		return err
	}

	// zero would expire the context instantly — rejected
	assert.Error(t, load(t.TempDir(), 0))
	// no ceiling: a user willing to wait 123 minutes may
	assert.NoError(t, load(t.TempDir(), 123))
}

func TestIsFirstRun(t *testing.T) {
	dir := t.TempDir()

	missing := filepath.Join(dir, "nope")
	if !IsFirstRun(missing) {
		t.Fatal("missing dir should be first run")
	}

	empty := filepath.Join(dir, "empty")
	err := os.MkdirAll(empty, 0o755)
	assert.NoError(t, err)
	if !IsFirstRun(empty) {
		t.Fatal("empty dir should be first run")
	}

	if err := os.WriteFile(filepath.Join(empty, "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if IsFirstRun(empty) {
		t.Fatal("dir with a file should not be first run")
	}
}

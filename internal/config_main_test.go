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

	cfg, sysPrompts, err := LoadConfig(baseDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, cfg.DefaultModelTag)
	assert.NotEmpty(t, sysPrompts.Cmd)
	assert.NotEmpty(t, sysPrompts.Explain)

	assert.True(t, strings.HasPrefix(ConfigPath(baseDir), baseDir), "config path outside baseDir")
	assert.True(t, strings.HasPrefix(SysPromptCmdPath(baseDir), baseDir), "cmd sysprompt path outside baseDir")
	assert.True(t, strings.HasPrefix(SysPromptExplainPath(baseDir), baseDir), "explain sysprompt path outside baseDir")
	assert.True(t, strings.HasPrefix(ConversationPath(baseDir), baseDir), "conversation path outside baseDir")

	assert.FileExists(t, ConfigPath(baseDir))
	assert.FileExists(t, SysPromptCmdPath(baseDir))
	assert.FileExists(t, SysPromptExplainPath(baseDir))
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

// Guards the shell contract of the built-in sys prompt: generation must stay
// bash-by-default, and the explain extension must carry its mode rules —
// so a prompt edit can not silently drop either.
func TestSysPromptContent(t *testing.T) {
	assert.Contains(t, ProgramSysPrompt, "Bash is the default output shell")
	assert.Contains(t, ProgramSysPrompt, "copy-paste ready bash command")
	assert.NotContains(t, ProgramSysPrompt, "Explanation")

	assert.Contains(t, ExplainSysPrompt, "Explanation Mode")
	assert.Contains(t, ExplainSysPrompt, "Do not generate a new command")
}

// A pre-existing single sysprompt.md must survive the two-file split: it is
// renamed to deprecated-sysprompt.md (kept, but unused), and both prompt
// files are generated with defaults.
func TestLegacySysPromptMigrated(t *testing.T) {
	baseDir := t.TempDir()
	legacy := filepath.Join(baseDir, LegacySysPromptFileName)
	require.NoError(t, os.WriteFile(legacy, []byte("custom prompt"), 0o600))

	_, _, err := LoadConfig(baseDir)
	require.NoError(t, err)

	assert.NoFileExists(t, legacy)
	data, err := os.ReadFile(filepath.Join(baseDir, DeprecatedPrefix+LegacySysPromptFileName))
	require.NoError(t, err)
	assert.Equal(t, "custom prompt", string(data))
	assert.FileExists(t, SysPromptCmdPath(baseDir))
	assert.FileExists(t, SysPromptExplainPath(baseDir))
}

// Refusal case: a legacy sysprompt.md AND a deprecated-sysprompt.md (from an
// earlier migration) both exist. The rename must be refused — not silently
// overwrite the earlier deprecated prompt (os.Rename clobbers on Unix) —
// with an error naming both files, and nothing on disk may change.
func TestLegacySysPromptRenameRefusedWhenDeprecatedExists(t *testing.T) {
	baseDir := t.TempDir()
	legacy := filepath.Join(baseDir, LegacySysPromptFileName)
	dep := filepath.Join(baseDir, DeprecatedPrefix+LegacySysPromptFileName)
	require.NoError(t, os.WriteFile(legacy, []byte("second custom"), 0o600))
	require.NoError(t, os.WriteFile(dep, []byte("first custom"), 0o600))

	_, _, err := LoadConfig(baseDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), legacy, "error must name the legacy file")
	assert.Contains(t, err.Error(), dep, "error must name the file that would be overwritten")

	// both prompts survive untouched
	data, rErr := os.ReadFile(legacy)
	require.NoError(t, rErr)
	assert.Equal(t, "second custom", string(data))
	data, rErr = os.ReadFile(dep)
	require.NoError(t, rErr)
	assert.Equal(t, "first custom", string(data))
}

// Corner case: both new prompt files already exist, yet a legacy sysprompt.md
// is still around — it must be deprecated in that state too.
func TestLegacySysPromptDeprecatedWithNewFilesPresent(t *testing.T) {
	baseDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, LegacySysPromptFileName), []byte("custom"), 0o600))
	require.NoError(t, os.WriteFile(SysPromptCmdPath(baseDir), []byte("cmd"), 0o600))
	require.NoError(t, os.WriteFile(SysPromptExplainPath(baseDir), []byte("explain"), 0o600))

	_, _, err := LoadConfig(baseDir)
	require.NoError(t, err)

	assert.NoFileExists(t, filepath.Join(baseDir, LegacySysPromptFileName))
	data, err := os.ReadFile(filepath.Join(baseDir, DeprecatedPrefix+LegacySysPromptFileName))
	require.NoError(t, err)
	assert.Equal(t, "custom", string(data))
	// the existing prompt files are untouched
	data, err = os.ReadFile(SysPromptCmdPath(baseDir))
	require.NoError(t, err)
	assert.Equal(t, "cmd", string(data))
}

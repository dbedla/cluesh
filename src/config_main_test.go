package cluesh

import (
	"os"
	"strings"
	"testing"
)

func TestBaseDirInjection(t *testing.T) {
	baseDir := t.TempDir()

	cfg, sysPrompt, err := LoadConfig(baseDir)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.DefaultModelTag == "" || sysPrompt == "" {
		t.Fatal("LoadConfig returned empty defaults")
	}

	for name, got := range map[string]string{
		"config":       ConfigPath(baseDir),
		"sysprompt":    SysPromptPath(baseDir),
		"conversation": ConversationPath(baseDir),
	} {
		if !strings.HasPrefix(got, baseDir) {
			t.Errorf("%s path %q outside baseDir %q", name, got, baseDir)
		}
	}
	for name, path := range map[string]string{
		"config":    ConfigPath(baseDir),
		"sysprompt": SysPromptPath(baseDir),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s file not created in baseDir: %v", name, err)
		}
	}
	// conversation.jsonl is deliberately absent: rellm's FilesystemConversation
	// creates it lazily on first Append — missing file = empty conversation.
}
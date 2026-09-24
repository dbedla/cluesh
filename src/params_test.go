package cluesh

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Params
	}{
		{"continue flag first", []string{"-c", "do thing"}, Params{Continue: true, Prompt: "do thing"}},
		{"continue flag last", []string{"do thing", "-c"}, Params{Continue: true, Prompt: "do thing"}},
		{"no flag", []string{"do thing"}, Params{Prompt: "do thing"}},
		{"llm-list no prompt", []string{"--llm-list"}, Params{LLMList: true}},
		{"llm-list with prompt", []string{"--llm-list", "do thing"}, Params{LLMList: true, Prompt: "do thing"}},
		{"llm tag before prompt", []string{"--llm", "or-gemma", "do thing"}, Params{LLMTag: "or-gemma", Prompt: "do thing"}},
		{"llm tag after prompt", []string{"do thing", "--llm", "or-gemma"}, Params{LLMTag: "or-gemma", Prompt: "do thing"}},
		{"help no prompt", []string{"--help"}, Params{Help: true}},
		{"help with prompt", []string{"-h", "do thing"}, Params{Help: true, Prompt: "do thing"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseParams(tt.args)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseParamsError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"missing positional", []string{"-c"}},
		{"extra positional", []string{"a", "b"}},
		{"unknown flag", []string{"-x"}},
		{"llm missing value", []string{"--llm"}},
		{"llm tag but no prompt", []string{"--llm", "or-gemma"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseParams(tt.args)
			assert.Error(t, err)
		})
	}
}

func TestStartFresh(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conversation.jsonl")

	// missing file is already fresh
	assert.NoError(t, StartFresh(path))

	assert.NoError(t, os.WriteFile(path, []byte("{}\n"), 0o600))
	assert.NoError(t, StartFresh(path))
	assert.NoFileExists(t, path)
}
func TestGetHelp(t *testing.T) {
	baseDir := t.TempDir()
	out := GetHelp(baseDir)

	for _, want := range []string{
		"usage: cluesh", "-c, --continue", "--llm-list", "--llm", "-h, --help",
		"config.json", "sysprompt.md", "conversation.jsonl", "openrouter.json",
		"api_key_env", baseDir,
	} {
		assert.Contains(t, out, want)
	}
	assert.NotContains(t, out, "api_key_file")
}

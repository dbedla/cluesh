package cluesh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    Params
		wantErr bool
	}{
		{"continue flag first", []string{"-c", "do thing"}, Params{Continue: true, Prompt: "do thing"}, false},
		{"continue flag last", []string{"do thing", "-c"}, Params{Continue: true, Prompt: "do thing"}, false},
		{"no flag", []string{"do thing"}, Params{Prompt: "do thing"}, false},
		{"missing positional", []string{"-c"}, Params{}, true},
		{"extra positional", []string{"a", "b"}, Params{}, true},
		{"unknown flag", []string{"-x"}, Params{}, true},
		{"llm-list no prompt", []string{"--llm-list"}, Params{LLMList: true}, false},
		{"llm-list with prompt", []string{"--llm-list", "do thing"}, Params{LLMList: true, Prompt: "do thing"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseParams(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseParams(%q) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got != tt.want {
				t.Errorf("ParseParams(%q) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestStartFresh(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conversation.jsonl")

	// missing file is already fresh
	if err := StartFresh(path); err != nil {
		t.Fatalf("StartFresh on missing file: %v", err)
	}

	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := StartFresh(path); err != nil {
		t.Fatalf("StartFresh on existing file: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("conversation file still exists after StartFresh")
	}
}
package cluesh

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ProgramSysPrompt is the built-in sys prompt. It is written into a freshly
// generated config and cluesh always reads the sys prompt from config after
// that — this constant is the single source of the default.
const ProgramSysPrompt = `
# Domain Expertise
Bash, Zsh, Shell Scripting (sh/csh/tcsh), PowerShell, and general command-line interfaces.

# Task

Your task is to process one of two types of input provided by the user:

1. **Command Generation**: If the user provides a natural language description, generate a single, optimized, one-line, copy-paste ready command that fulfills the request.
2. **Command Explanation**: If the user provides an existing command, analyze it and explain its functionality by deconstructing its components. Do not evaluate correctness just explain all arguments.

# Rules

- **Optimization**: Prefer these utilities where possible: find, grep, cut, sort, uniq, xargs, wc, head, cat, less, tail, ls, tree.
- **Simplicity**: Avoid control structures (loops, if/else) unless absolutely necessary for the task.
- **Placeholders**: Use the shortest possible placeholders in any examples provided.
- **Deconstruction (Crucial)**: Regardless of whether you are generating or explaining a command, you must populate the "sub_commands" field to break down the logic:
    - Every component of the pipeline must be listed in sequential order.
    - Each entry must include the base command and an explanation of ALL its arguments (e.g., "ls -la dir" -> one entry with command ls and arguments -l, -a, dir).
    - Never leave the "sub_commands" field empty.

# Output

Return only JSON matching the given schema.

`

// ConfigDir is the single place where all cluesh configuration lives:
// config.json (main), provider files, tools.json, last conversation.
const ConfigDir = ".cluesh"

// ConfigFileName is the main configuration file inside ConfigDir.
const ConfigFileName = "config.json"

// SysPromptFileName is the markdown sys prompt file inside ConfigDir.
// The prompt lives in its own file because JSON has no raw multiline strings —
// no \n escapes to fight with when editing it.
const SysPromptFileName = "sysprompt.md"

// ConversationFileName is the persisted conversation file inside ConfigDir.
// JSON lines (rellm FilesystemConversation); missing file = empty conversation.
const ConversationFileName = "conversation.jsonl"

// MainConfig is the main configuration of the program.
type MainConfig struct {
	DefaultModelTag         string `json:"default_model_tag"`         // required, e.g. "or-glm53flash"
	PutCmdInClipboard       string `json:"put_cmd_in_clipboard"`      // always | never | read-only
	Colors                  string `json:"colors"`                    // dark | light | none
	ExecutionTimeoutMinutes int    `json:"execution_timeout_minutes"` // agent timeout, minutes — must be >=1, no upper limit
}

// Sampling parameters (temperature, reasoning) are per model, configured in
// the provider files — see Model.

// configTemplate is what gets written on first run: the main config plus a
// generated _options block listing every allowed value, so users can see
// what to change without reading the source.
type configTemplate struct {
	Options map[string][]string `json:"_options"`
	MainConfig
}

// configOptions returns the allowed values for each enum-like config field.
func configOptions() map[string][]string {
	return map[string][]string{
		"put_cmd_in_clipboard": {"always", "never", "read-only"},
		"colors":               {"dark", "light", "none"},
	}
}

// DefaultMainConfig returns the config written on first run.
func DefaultMainConfig() MainConfig {
	return MainConfig{
		DefaultModelTag:         "or-glm53flash",
		PutCmdInClipboard:       "always",
		Colors:                  "dark",
		ExecutionTimeoutMinutes: 5,
	}
}

// DefaultBaseDir returns the standard cluesh base directory ($HOME/.cluesh).
// The only place the home directory is resolved; tests pass t.TempDir() instead.
func DefaultBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ConfigDir), nil
}

// ConfigPath returns the path of the main config file inside baseDir.
func ConfigPath(baseDir string) string {
	return filepath.Join(baseDir, ConfigFileName)
}

// SysPromptPath returns the path of the sys prompt file inside baseDir.
func SysPromptPath(baseDir string) string {
	return filepath.Join(baseDir, SysPromptFileName)
}

// ConversationPath returns the path of the conversation file inside baseDir.
func ConversationPath(baseDir string) string {
	return filepath.Join(baseDir, ConversationFileName)
}

// IsFirstRun reports whether baseDir is missing or empty — nothing has been
// configured yet. A dir that can't be read counts as first run; the real
// error surfaces later from EnsureConfig.
func IsFirstRun(baseDir string) bool {
	entries, err := os.ReadDir(baseDir)
	return err != nil || len(entries) == 0
}

// EnsureConfig makes sure the base dir with config.json and sysprompt.md exists.
// Missing files are generated with default settings; created reports them.
func EnsureConfig(baseDir string) (created []string, err error) {
	if mkErr := os.MkdirAll(baseDir, 0o755); mkErr != nil {
		return nil, fmt.Errorf("cannot create config dir %s: %w", baseDir, mkErr)
	}

	cfgPath := ConfigPath(baseDir)
	if _, statErr := os.Stat(cfgPath); errors.Is(statErr, fs.ErrNotExist) {
		if wErr := writeJSON(cfgPath, configTemplate{Options: configOptions(), MainConfig: DefaultMainConfig()}); wErr != nil {
			return nil, wErr
		}
		created = append(created, cfgPath)
	} else if statErr != nil {
		return nil, fmt.Errorf("cannot access %s: %w", cfgPath, statErr)
	}

	spPath := SysPromptPath(baseDir)
	if _, statErr := os.Stat(spPath); errors.Is(statErr, fs.ErrNotExist) {
		if wErr := os.WriteFile(spPath, []byte(ProgramSysPrompt), 0o600); wErr != nil {
			return nil, fmt.Errorf("cannot write %s: %w", spPath, wErr)
		}
		created = append(created, spPath)
	} else if statErr != nil {
		return nil, fmt.Errorf("cannot access %s: %w", spPath, statErr)
	}
	return created, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}
	return nil
}

// LoadConfig reads and validates the main config and the sys prompt file.
// Missing files are generated on first run with a notice.
func LoadConfig(baseDir string) (MainConfig, string, error) {
	created, err := EnsureConfig(baseDir)
	for _, p := range created {
		fmt.Printf("created %s\n", p)
	}
	if err != nil {
		return MainConfig{}, "", err
	}

	cfgPath := ConfigPath(baseDir)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return MainConfig{}, "", fmt.Errorf("cannot read %s: %w", cfgPath, err)
	}

	var cfg MainConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return MainConfig{}, "", fmt.Errorf("invalid json in %s: %w", cfgPath, err)
	}

	switch cfg.PutCmdInClipboard {
	case "always", "never", "read-only":
	default:
		return MainConfig{}, "", fmt.Errorf("%s: put_cmd_in_clipboard must be one of always|never|read-only, got %q", cfgPath, cfg.PutCmdInClipboard)
	}
	switch cfg.Colors {
	case "dark", "light", "none":
	default:
		return MainConfig{}, "", fmt.Errorf("%s: colors must be one of dark|light|none, got %q", cfgPath, cfg.Colors)
	}
	if cfg.DefaultModelTag == "" {
		return MainConfig{}, "", errors.New("default_model_tag is missing in " + cfgPath + " — run cluesh --llm-list to see configured models")
	}
	if cfg.ExecutionTimeoutMinutes < 1 {
		return MainConfig{}, "", fmt.Errorf("%s: execution_timeout_minutes must be >=1, got %d", cfgPath, cfg.ExecutionTimeoutMinutes)
	}

	spPath := SysPromptPath(baseDir)
	sysPrompt, err := os.ReadFile(spPath)
	if err != nil {
		return MainConfig{}, "", fmt.Errorf("cannot read sys prompt %s: %w", spPath, err)
	}

	return cfg, string(sysPrompt), nil
}

package cluesh

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dbedla/rellm/pkg/rellm"
)

// ProgramSysPrompt is the built-in sys prompt. It is written into a freshly
// generated config and cluesh always reads the sys prompt from config after
// that — this constant is the single source of the default.
const ProgramSysPrompt = `# Task

Provide one final bash command (or bash command combination) which fulfills the requested demand. You are a bash expert.
The answer must be a oneline copy-paste ready bash command.

# Rules

- Prefer these commands where possible: find, grep, cut, sort, uniq, xargs, wc, head, cat, less, tail, wc, ls, tree.
- No loops and ifs unless absolutely necessary.
- Placeholder in example should be as short as possible.
- sub_commands must list EVERY command of the pipeline in order, including the first, each with ALL its arguments explained — e.g. "ls -la dir" → one sub_commands entry "ls" with arguments -l, -a, dir; never leave sub_commands empty.

# Output

Return only JSON matching the given schema.`

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
	DefaultModelTag        string  `json:"default_model_tag"`         // required, e.g. "or-glm53flash"
	PutCmdInClipboard      string  `json:"put_cmd_in_clipboard"`      // always | never | read-only
	Colors                 string  `json:"colors"`                    // dark | light | none
	ExecutionTimeoutMinutes int    `json:"execution_timeout_minutes"` // agent Ask() timeout, minutes — must be >0 and <=60
	ReasoningEffort        string  `json:"reasoning_effort"`          // none | low | medium | high
	Temperature            float64 `json:"temperature"`               // passed through to the provider unvalidated
}

// configTemplate is what gets written on first run: the main config plus a
// generated _options block listing every allowed value, so users can see
// what to change without reading the source.
type configTemplate struct {
	Options map[string][]string `json:"_options"`
	MainConfig
}

// DefaultReasoningEffort and DefaultTemperature are the prompt-parameter
// values written into a freshly generated config.
const (
	DefaultReasoningEffort = "low"
	DefaultTemperature     = 0.4
)

// configOptions returns the allowed values for each enum-like config field.
func configOptions() map[string][]string {
	return map[string][]string{
		"put_cmd_in_clipboard":   {"always", "never", "read-only"},
		"colors":                 {"dark", "light", "none"},
		"reasoning_effort":       {"none", "low", "medium", "high"},
	}
}

// DefaultMainConfig returns the config written on first run.
func DefaultMainConfig() MainConfig {
	return MainConfig{
		DefaultModelTag:         "or-glm53flash",
		PutCmdInClipboard:       "always",
		Colors:                  "dark",
		ExecutionTimeoutMinutes: 5,
		ReasoningEffort:         DefaultReasoningEffort,
		Temperature:             DefaultTemperature,
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

// EnsureConfig makes sure the base dir with config.json and sysprompt.md exists.
// Missing files are generated with default settings; created reports them.
func EnsureConfig(baseDir string) (created []string, err error) {
	if mkErr := os.MkdirAll(baseDir, 0o755); mkErr != nil {
		return nil, fmt.Errorf("cannot create config dir %s: %w", baseDir, mkErr)
	}

	cfgPath := ConfigPath(baseDir)
	if _, statErr := os.Stat(cfgPath); os.IsNotExist(statErr) {
		if wErr := writeJSON(cfgPath, configTemplate{Options: configOptions(), MainConfig: DefaultMainConfig()}); wErr != nil {
			return nil, wErr
		}
		created = append(created, cfgPath)
	} else if statErr != nil {
		return nil, fmt.Errorf("cannot access %s: %w", cfgPath, statErr)
	}

	spPath := SysPromptPath(baseDir)
	if _, statErr := os.Stat(spPath); os.IsNotExist(statErr) {
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
	if _, err := ReasoningEffort(cfg.ReasoningEffort); err != nil {
		return MainConfig{}, "", fmt.Errorf("%s: %w", cfgPath, err)
	}


	spPath := SysPromptPath(baseDir)
	sysPrompt, err := os.ReadFile(spPath)
	if err != nil {
		return MainConfig{}, "", fmt.Errorf("cannot read sys prompt %s: %w", spPath, err)
	}

	return cfg, string(sysPrompt), nil
}

// GenerateTemplateConfig regenerates missing default config files
// (--generate-template-config). Existing files are left untouched.
func GenerateTemplateConfig(baseDir string) ([]string, error) {
	return EnsureConfig(baseDir)
}

// ReasoningEffort maps the config string to rellm's enum. Exact match only:
// any other value (including empty) is an error. LoadConfig calls it, so an
// unknown or missing reasoning_effort fails config load with the exact set.
func ReasoningEffort(s string) (rellm.ReasoningEffort, error) {
	switch s {
	case "none":
		return rellm.ReasoningEffortNone, nil
	case "low":
		return rellm.ReasoningEffortLow, nil
	case "medium":
		return rellm.ReasoningEffortMedium, nil
	case "high":
		return rellm.ReasoningEffortHigh, nil
	}
	return "", fmt.Errorf("reasoning_effort must be one of none|low|medium|high, got %q", s)
}

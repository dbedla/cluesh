package cluesh

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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

// MainConfig is the main configuration of the program.
type MainConfig struct {
	DefaultModelTag   string `json:"default_model_tag"`    // required, e.g. "or-glm53flash"
	PutCmdInClipboard string `json:"put_cmd_in_clipboard"` // always | never | read-only
	Colors            string `json:"colors"`               // dark | light | none
}

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
		DefaultModelTag:   "or-glm53flash",
		PutCmdInClipboard: "always",
		Colors:            "dark",
	}
}

// ConfigPath returns the absolute path of the main config file (~/.cluesh/config.json).
func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return dir + string(os.PathSeparator) + ConfigFileName, nil
}

// SysPromptPath returns the absolute path of the sys prompt file (~/.cluesh/sysprompt.txt).
func SysPromptPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return dir + string(os.PathSeparator) + SysPromptFileName, nil
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return home + string(os.PathSeparator) + ConfigDir, nil
}

// EnsureConfig makes sure ~/.cluesh/ with config.json and sysprompt.txt exists.
// Missing files are generated with default settings; created reports them.
func EnsureConfig() (created []string, err error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
		return nil, fmt.Errorf("cannot create config dir %s: %w", dir, mkErr)
	}

	cfgPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	if _, statErr := os.Stat(cfgPath); os.IsNotExist(statErr) {
		if wErr := writeJSON(cfgPath, configTemplate{Options: configOptions(), MainConfig: DefaultMainConfig()}); wErr != nil {
			return nil, wErr
		}
		created = append(created, cfgPath)
	} else if statErr != nil {
		return nil, fmt.Errorf("cannot access %s: %w", cfgPath, statErr)
	}

	spPath, err := SysPromptPath()
	if err != nil {
		return nil, err
	}
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
func LoadConfig() (MainConfig, string, error) {
	created, err := EnsureConfig()
	for _, p := range created {
		fmt.Printf("created %s\n", p)
	}
	if err != nil {
		return MainConfig{}, "", err
	}

	cfgPath, err := ConfigPath()
	if err != nil {
		return MainConfig{}, "", err
	}
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

	spPath, err := SysPromptPath()
	if err != nil {
		return MainConfig{}, "", err
	}
	sysPrompt, err := os.ReadFile(spPath)
	if err != nil {
		return MainConfig{}, "", fmt.Errorf("cannot read sys prompt %s: %w", spPath, err)
	}

	return cfg, string(sysPrompt), nil
}

// GenerateTemplateConfig regenerates missing default config files
// (--generate-template-config). Existing files are left untouched.
func GenerateTemplateConfig() ([]string, error) {
	return EnsureConfig()
}

package cluesh

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ProgramSysPrompt is the built-in sys prompt for generation mode (no -e),
// written into sysprompt-cmd.md on first run. cluesh always reads the sys
// prompts from the config files after that — these constants are the single
// source of the defaults. The explain-mode counterpart is ExplainSysPrompt.
const ProgramSysPrompt = `
# Domain Expertise
Bash, Zsh, Shell Scripting (sh/csh/tcsh), PowerShell, and general command-line interfaces.

# Task

Generate one final command which fulfills the requested demand.

**Bash is the default output shell.** Only when the user explicitly asks for a
different shell (e.g. Zsh or PowerShell), produce the command for that shell
instead and note the deviation in "notes".

# Rules

- The answer must be a one-line, copy-paste ready bash command (unless the
  user explicitly asked for another shell, see above).
- **Optimization**: Prefer these utilities where possible: find, grep, cut, sort, uniq, xargs, wc, head, cat, less, tail, ls, tree.
- **Simplicity**: Avoid control structures (loops, if/else) unless absolutely necessary for the task.
- **Placeholders**: Use the shortest possible placeholders in any examples provided.
- **Deconstruction (Crucial)**: You must populate the "sub_commands" field to break down the logic:
    - Every component of the pipeline must be listed in sequential order.
    - Each entry must include the base command and an explanation of ALL its arguments (e.g., "ls -la dir" -> one entry with command ls and arguments -l, -a, dir).
    - Never leave the "sub_commands" field empty.

# Output

Return only JSON matching the given schema.

`

// ExplainSysPrompt is the built-in sys prompt for explain mode (-e), kept
// in its own file (sysprompt-explain.md) so the two modes can be edited
// independently.
const ExplainSysPrompt = `
# Explanation Mode

The user input is an existing command, NOT a demand. Explain it:

- Do not generate a new command. Copy the input command as given into
  "final_command", unchanged.
- Deconstruct the command: list EVERY subcommand of the pipeline in order,
  each with ALL its arguments explained — e.g., "ls -la dir" -> one entry
  with command ls and arguments -l, -a, dir. Never leave "sub_commands" empty.
- Do not evaluate correctness or suggest improvements, only explain what
  each part does.
- Set "read_only" to true when the command changes no file, false otherwise.
- In "notes", give a short explanation of what the command does overall —
  it is the only prose the user sees for commands without arguments.

# Output

Return only JSON matching the given schema.

`

// ConfigDir is the single place where all cluesh configuration lives:
// config.json (main), provider files, tools.json, last conversation.
const ConfigDir = ".cluesh"

// ConfigFileName is the main configuration file inside ConfigDir.
const ConfigFileName = "config.json"

// The sys prompts live in their own files because JSON has no raw multiline
// strings — no \n escapes to fight with when editing them.
const (
	// SysPromptCmdFileName is the generation-mode (-e absent) sys prompt file.
	SysPromptCmdFileName = "sysprompt-cmd.md"
	// SysPromptExplainFileName is the explain-mode (-e) sys prompt file.
	SysPromptExplainFileName = "sysprompt-explain.md"
	// LegacySysPromptFileName is the single-prompt file from before the two
	// prompt files existed; it is migrated to sysprompt-cmd.md on startup.
	LegacySysPromptFileName = "sysprompt.md"
)

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

// SysPromptCmdPath returns the path of the generation sys prompt file.
func SysPromptCmdPath(baseDir string) string {
	return filepath.Join(baseDir, SysPromptCmdFileName)
}

// SysPromptExplainPath returns the path of the explain sys prompt file.
func SysPromptExplainPath(baseDir string) string {
	return filepath.Join(baseDir, SysPromptExplainFileName)
}

// SysPrompts carries both mode prompts, read from their files.
type SysPrompts struct {
	Cmd     string // generation mode (no -e)
	Explain string // explanation mode (-e)
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

// EnsureConfig makes sure the base dir with config.json, sysprompt-cmd.md
// and sysprompt-explain.md exists. Missing files are generated with default
// settings; created reports them. A legacy sysprompt.md is migrated to
// sysprompt-cmd.md instead of being overwritten, so customizations survive.
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

	cmdPath := SysPromptCmdPath(baseDir)
	if _, statErr := os.Stat(cmdPath); errors.Is(statErr, fs.ErrNotExist) {
		legacyPath := filepath.Join(baseDir, LegacySysPromptFileName)
		if _, legErr := os.Stat(legacyPath); legErr == nil {
			if rErr := os.Rename(legacyPath, cmdPath); rErr != nil {
				return nil, fmt.Errorf("cannot migrate %s to %s: %w", legacyPath, cmdPath, rErr)
			}
		} else if wErr := os.WriteFile(cmdPath, []byte(ProgramSysPrompt), 0o600); wErr != nil {
			return nil, fmt.Errorf("cannot write %s: %w", cmdPath, wErr)
		}
		created = append(created, cmdPath)
	} else if statErr != nil {
		return nil, fmt.Errorf("cannot access %s: %w", cmdPath, statErr)
	}

	expPath := SysPromptExplainPath(baseDir)
	if _, statErr := os.Stat(expPath); errors.Is(statErr, fs.ErrNotExist) {
		if wErr := os.WriteFile(expPath, []byte(ExplainSysPrompt), 0o600); wErr != nil {
			return nil, fmt.Errorf("cannot write %s: %w", expPath, wErr)
		}
		created = append(created, expPath)
	} else if statErr != nil {
		return nil, fmt.Errorf("cannot access %s: %w", expPath, statErr)
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

// LoadConfig reads and validates the main config and both sys prompt files.
// Missing files are generated on first run with a notice.
func LoadConfig(baseDir string) (MainConfig, SysPrompts, error) {
	created, err := EnsureConfig(baseDir)
	for _, p := range created {
		fmt.Printf("created %s\n", p)
	}
	if err != nil {
		return MainConfig{}, SysPrompts{}, err
	}

	cfgPath := ConfigPath(baseDir)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return MainConfig{}, SysPrompts{}, fmt.Errorf("cannot read %s: %w", cfgPath, err)
	}

	var cfg MainConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return MainConfig{}, SysPrompts{}, fmt.Errorf("invalid json in %s: %w", cfgPath, err)
	}

	switch cfg.PutCmdInClipboard {
	case "always", "never", "read-only":
	default:
		return MainConfig{}, SysPrompts{}, fmt.Errorf("%s: put_cmd_in_clipboard must be one of always|never|read-only, got %q", cfgPath, cfg.PutCmdInClipboard)
	}
	switch cfg.Colors {
	case "dark", "light", "none":
	default:
		return MainConfig{}, SysPrompts{}, fmt.Errorf("%s: colors must be one of dark|light|none, got %q", cfgPath, cfg.Colors)
	}
	if cfg.DefaultModelTag == "" {
		return MainConfig{}, SysPrompts{}, errors.New("default_model_tag is missing in " + cfgPath + " — run cluesh --llm-list to see configured models")
	}
	if cfg.ExecutionTimeoutMinutes < 1 {
		return MainConfig{}, SysPrompts{}, fmt.Errorf("%s: execution_timeout_minutes must be >=1, got %d", cfgPath, cfg.ExecutionTimeoutMinutes)
	}

	readPrompt := func(path string) (string, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("cannot read sys prompt %s: %w", path, err)
		}
		return string(data), nil
	}
	cmdPrompt, err := readPrompt(SysPromptCmdPath(baseDir))
	if err != nil {
		return MainConfig{}, SysPrompts{}, err
	}
	explainPrompt, err := readPrompt(SysPromptExplainPath(baseDir))
	if err != nil {
		return MainConfig{}, SysPrompts{}, err
	}

	return cfg, SysPrompts{Cmd: cmdPrompt, Explain: explainPrompt}, nil
}

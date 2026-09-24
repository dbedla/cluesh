// Package cluesh — parameter parsing, help output and conversation-file handling.
package cluesh

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	flag "github.com/spf13/pflag"
)

// Params is the parsed command line.
type Params struct {
	Continue bool   // -c: continue last conversation
	LLMList  bool   // --llm-list: list configured models and exit
	LLMTag   string // --llm: model tag to use for this run
	Help     bool   // --help: show help and exit
	Prompt   string // the positional demand
}

// newFlagSet registers every flag with its full help description. The
// descriptions are the help content — ParseParams and GetHelp share this
// one definition.
func newFlagSet() (*flag.FlagSet, *Params) {
	var p Params
	fs := flag.NewFlagSet("cluesh", flag.ContinueOnError)
	fs.BoolVarP(&p.Continue, "continue", "c", false, "continue last conversation")
	fs.BoolVar(&p.LLMList, "llm-list", false, "list configured models and exit")
	fs.StringVarP(&p.LLMTag, "llm", "", "", "use model with tag <tag> this run (default_model_tag if unset)")
	fs.BoolVarP(&p.Help, "help", "h", false, "show help and exit")
	return fs, &p
}

// ParseParams parses args (without the program name) into Params. Flag order
// is free (pflag interspersed parsing): exactly one positional — the demand —
// is required, e.g. `cluesh -c "demand"` ≡ `cluesh "demand" -c`.
func ParseParams(args []string) (Params, error) {
	fs, p := newFlagSet()
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return Params{}, err
	}

	positional := fs.Args()
	// --llm-list and --help are informational runs: the demand is optional.
	if len(positional) > 1 || (len(positional) == 0 && !p.LLMList && !p.Help) {
		return Params{}, errors.New("usage: cluesh [-c] \"<demand>\" — exactly one prompt argument required (see --help)")
	}
	if len(positional) == 1 {
		p.Prompt = positional[0]
	}
	return *p, nil
}

// shortenHome replaces the user's home directory prefix with "~" so paths in
// help output don't leak usernames.
func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	return strings.Replace(path, home, "~", 1)
}

// GetHelp returns usage, the generated flag list (pflag PrintDefaults), the
// config location and key setup. baseDir is resolved by the caller so help
// works even when the config is missing or broken.
func GetHelp(baseDir string) string {
	var help strings.Builder
	fs, _ := newFlagSet()
	help.WriteString("cluesh — natural-language demand in, one copy-paste-ready bash command out\n")
	help.WriteString("\nusage: cluesh [-c] [--llm <tag>] \"<demand>\"\n\nFlags:\n")
	fs.SetOutput(&help)
	fs.PrintDefaults()

	help.WriteString("\nhttps://github.com/dbedla/cluesh — #agent-of-rellm\n")
	help.WriteString("https://github.com/dbedla/rellm — LLM communication framework\n")
	help.WriteString("Detailed info: README.md\n")

	help.WriteString("\nConfig location: " + shortenHome(baseDir) + "\n")
	files := []struct{ name, desc string }{
		{ConfigFileName, "main settings (default model, clipboard mode, colors, timeout)"},
		{SysPromptFileName, "system prompt"},
		{ProvidersDir + "/", "one JSON file per provider (openrouter.json, openai.json, lmstudio.json)"},
		{ConversationFileName, "persisted conversation (created on first ask)"},
	}
	for _, f := range files {
		help.WriteString(fmt.Sprintf("  %-19s %s\n", f.name, f.desc))
	}
	help.WriteString(`
API key setup (per provider file, e.g. providers/openrouter.json):
  "api_key_env": "OPENROUTER_API_KEY"   key from environment (recommended)
  "api_key": "sk-..."                   inline in the provider file (last resort)
  both set? the environment variable wins
`)

	return help.String()
}

// StartFresh deletes the conversation file so the next run starts empty.
// A missing file is already fresh; rellm recreates the file on first Append.
func StartFresh(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

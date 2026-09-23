// Package cluesh — parameter parsing, help output and conversation-file handling.
package cluesh

import (
	"errors"
	"fmt"
	"io"
	"os"

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
// descriptions are the help content — ParseParams and PrintHelp share this
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
	if len(positional) != 1 && !(p.LLMList && len(positional) == 0) && !(p.Help && len(positional) == 0) {
		return Params{}, errors.New("usage: cluesh [-c] \"<demand>\" — exactly one prompt argument required (see --help)")
	}
	if len(positional) == 1 {
		p.Prompt = positional[0]
	}
	return *p, nil
}

// PrintHelp writes usage, the generated flag list (pflag PrintDefaults), the
// config location and key setup. baseDir is resolved by the caller — help
// must work even when the config is missing or broken.
func PrintHelp(w io.Writer, baseDir string) {
	fs, _ := newFlagSet()
	fmt.Fprintf(w, "cluesh — natural-language demand in, one copy-paste-ready bash command out\n")
	fmt.Fprintf(w, "usage: cluesh [-c] [--llm <tag>] \"<demand>\"\n\nFlags:\n")
	fs.SetOutput(w)
	fs.PrintDefaults()

	fmt.Fprintf(w, "\nConfig location: %s\n", baseDir)
	fmt.Fprintf(w, "  %s   main settings (default model, clipboard mode, colors, timeout)\n", ConfigFileName)
	fmt.Fprintf(w, "  %s   system prompt\n", SysPromptFileName)
	fmt.Fprintf(w, "  %s   one JSON file per provider (openrouter.json, openai.json, lmstudio.json)\n", ProvidersDir+"/")
	fmt.Fprintf(w, "  %s   persisted conversation (created on first ask)\n", ConversationFileName)

	fmt.Fprintf(w, "\nAPI key setup (per provider file, e.g. providers/openrouter.json):\n")
	fmt.Fprintf(w, `  "api_key_env": "OPENROUTER_API_KEY"   key from environment (recommended)
  "api_key": "sk-..."                   inline in the provider file (last resort)
`)
}

// StartFresh deletes the conversation file so the next run starts empty.
// A missing file is already fresh; rellm recreates the file on first Append.
func StartFresh(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
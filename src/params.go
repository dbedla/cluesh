// Package cluesh — parameter parsing and conversation-file handling.
package cluesh

import (
	"errors"
	"io"
	"os"

	flag "github.com/spf13/pflag"
)

// Params is the parsed command line.
type Params struct {
	Continue bool   // -c: continue last conversation
	Prompt   string // the positional demand
}

// ParseParams parses args (without the program name) into Params. Flag order
// is free (pflag interspersed parsing): exactly one positional — the demand —
// is required, e.g. `cluesh -c "demand"` ≡ `cluesh "demand" -c`.
func ParseParams(args []string) (Params, error) {
	var p Params

	fs := flag.NewFlagSet("cluesh", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVarP(&p.Continue, "continue", "c", false, "continue last conversation")

	if err := fs.Parse(args); err != nil {
		return Params{}, err
	}

	positional := fs.Args()
	if len(positional) != 1 {
		return Params{}, errors.New("usage: cluesh [-c] \"<demand>\" — exactly one prompt argument required")
	}
	p.Prompt = positional[0]
	return p, nil
}

// StartFresh deletes the conversation file so the next run starts empty.
// A missing file is already fresh; rellm recreates the file on first Append.
func StartFresh(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
package cluesh

import (
	"fmt"

	"github.com/fatih/color"
)

// Print renders a Command. ALL layout (newlines, indentation) lives here —
// the Consol implementations are pure writers: same bytes, only colored or
// not. That way no mode can disagree about layout.
func Print(format Consol, c Command) {
	format.FinalComand("\n%s\n\n", c.FinalCommand)
	for _, sc := range c.SubCommands {
		format.SubCommand("\t%s\n", sc.Command)
		for _, arg := range sc.Arguments {
			format.Argument("\t\t- %s\n", arg.Argument)
			format.ArgumentDescription("\t\t  %s\n", arg.Description)
		}
		fmt.Println()
	}
	if c.Notes != "" {
		format.Notes("%s\n", c.Notes)
	}
	fmt.Println()
}

// Consol is the output writer contract: format + args, no newline handling.
type Consol interface {
	FinalComand(format string, a ...interface{})
	SubCommand(format string, a ...interface{})
	Argument(format string, a ...interface{})
	ArgumentDescription(format string, a ...interface{})
	Notes(format string, a ...interface{})
}

// colored returns a writer using fatih/color's Color.Printf, which — unlike
// the package-level color.Red etc. — does NOT auto-append a newline. Print
// owns every newline, so modes never add or double one.
func colored(c *color.Color) func(format string, a ...interface{}) {
	return func(format string, a ...interface{}) { c.Printf(format, a...) }
}

// DarkMode — colors tuned for dark terminals.
type DarkMode struct{}

var _ Consol = &DarkMode{}

func (_ DarkMode) FinalComand(format string, a ...interface{}) {
	colored(color.New(color.FgHiRed))(format, a...)
}
func (_ DarkMode) SubCommand(format string, a ...interface{}) {
	colored(color.New(color.FgHiCyan))(format, a...)
}
func (_ DarkMode) Argument(format string, a ...interface{}) {
	colored(color.New(color.FgHiYellow))(format, a...)
}
func (_ DarkMode) ArgumentDescription(format string, a ...interface{}) {
	colored(color.New(color.FgHiWhite))(format, a...)
}
func (_ DarkMode) Notes(format string, a ...interface{}) {
	colored(color.New(color.FgHiBlue))(format, a...)
}

// LightMode — colors tuned for light terminals.
type LightMode struct{}

var _ Consol = &LightMode{}

func (_ LightMode) FinalComand(format string, a ...interface{}) {
	colored(color.New(color.FgRed))(format, a...)
}
func (_ LightMode) SubCommand(format string, a ...interface{}) {
	colored(color.New(color.FgCyan))(format, a...)
}
func (_ LightMode) Argument(format string, a ...interface{}) {
	colored(color.New(color.FgYellow))(format, a...)
}
func (_ LightMode) ArgumentDescription(format string, a ...interface{}) {
	colored(color.New(color.FgBlack))(format, a...)
}
func (_ LightMode) Notes(format string, a ...interface{}) {
	colored(color.New(color.FgGreen))(format, a...)
}

// NoMode — plain text, byte-identical layout to the colored modes.
type NoMode struct{}

var _ Consol = &NoMode{}

func (_ NoMode) FinalComand(format string, a ...interface{})         { fmt.Printf(format, a...) }
func (_ NoMode) SubCommand(format string, a ...interface{})          { fmt.Printf(format, a...) }
func (_ NoMode) Argument(format string, a ...interface{})            { fmt.Printf(format, a...) }
func (_ NoMode) ArgumentDescription(format string, a ...interface{}) { fmt.Printf(format, a...) }
func (_ NoMode) Notes(format string, a ...interface{})               { fmt.Printf(format, a...) }

// ConsolByName maps the colors config value (dark | light | none) to a
// Consol implementation. Config validation guarantees a valid value.
func ConsolByName(name string) Consol {
	switch name {
	case "dark":
		return &DarkMode{}
	case "light":
		return &LightMode{}
	default: // "none" and NO_COLOR-friendly fallback
		return &NoMode{}
	}
}

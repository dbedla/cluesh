package cluesh

import (
	"github.com/fatih/color"
)

func Print(format Consol, c Command) {

}

type Consol interface {
	FinalComand(format string, a ...interface{})
	SubCommand(format string, a ...interface{})
	Argument(format string, a ...interface{})
	ArgumentDescription(format string, a ...interface{})
	Notes(format string, a ...interface{})
}

type DarkMode struct {
}

var _ Consol = &DarkMode{}

func (_ DarkMode) FinalComand(format string, a ...interface{}) {
	color.Magenta(format, a)
}
func (_ DarkMode) SubCommand(format string, a ...interface{}) {
	color.Cyan(format, a)
}
func (_ DarkMode) Argument(format string, a ...interface{}) {
	color.Yellow(format, a)
}
func (_ DarkMode) ArgumentDescription(format string, a ...interface{}) {
	color.White(format, a)
}
func (_ DarkMode) Notes(format string, a ...interface{}) {
	color.Green(format, a)
}

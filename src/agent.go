// Package cluesh — agent wiring: natural-language bash demand in, one
// copy-paste-ready command out, as schema-constrained JSON.
package cluesh

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dbedla/rellm/pkg/rellm"
	"github.com/invopop/jsonschema"
)

// Command is the agent's structured output: one final command plus the
// explanation of every subcommand and argument it consists of.
type Command struct {
	FinalCommand string       `json:"final_command" jsonschema:"description=The final bash command, oneline, copy-paste ready"`
	SubCommands  []SubCommand `json:"sub_commands"  jsonschema:"description=EVERY command of the pipeline, in order, including the first, each with ALL its arguments explained, never empty"`
	Notes        string       `json:"notes"        jsonschema:"description=Short notes, e.g. caveats or variants, empty if none"`
	RedOnly      bool         `json:"read_only"      jsonschema:"description=true when the command does not change any file, false when it modifies anything"`
}

// SubCommand is one command within the pipeline.
type SubCommand struct {
	Command   string     `json:"command"   jsonschema:"description=Name of the command, e.g. find"`
	Arguments []Argument `json:"arguments" jsonschema:"description=Arguments of the command"`
}

// Argument is one argument with its explanation.
type Argument struct {
	Argument    string `json:"argument"    jsonschema:"description=The argument as written, e.g. -type f"`
	Description string `json:"description" jsonschema:"description=What the argument does"`
}

// DefaultMaxAgentSteps caps the tool-calling loop when a toolset is injected.
const DefaultMaxAgentSteps = 5

// AgentConfig carries everything needed to build the agent. Conversation and
// Toolset may be nil; MaxSteps and SysPrompt fall back to defaults when zero.
type AgentConfig struct {
	Provider     rellm.Provider
	Conversation rellm.Conversation // nil → new in-memory conversation
	Toolset      rellm.Toolset      // nil → no tools
	MaxSteps     uint64             // zero → DefaultMaxAgentSteps
	SysPrompt    string             // empty → defaultSysPrompt (+ embedded schema)
}

// CommandTextFormat returns the JSON-schema text format generated from Command.
func CommandTextFormat() rellm.TextFormat {
	return rellm.TextFormat{
		Type:   "json_schema",
		Name:   "command",
		Strict: true,
		Schema: (&jsonschema.Reflector{DoNotReference: true}).Reflect(&Command{}),
	}
}

// NewAgent builds the agent from the given provider, conversation and optional
// toolset, configured for structured output into Command.
func NewAgent(cfg AgentConfig) (*rellm.Agent, error) {
	textFormat := CommandTextFormat()

	steps := cfg.MaxSteps
	if steps == 0 {
		return nil, errors.New("no agent steps defined")
	}

	builder := rellm.NewAgentBuilder().
		WithAgentName("cluesh").
		WithProvider(cfg.Provider).
		WithMaxAgentSteps(steps).
		WithConversation(cfg.Conversation).
		WithSystemMessage(cfg.SysPrompt).
		WithTextFormat(textFormat).
		WithImageGenerationKeepInTheLoop().
		WithUnknownConversationElementKeepInTheLoop()

	if cfg.Toolset != nil {
		builder = builder.WithToolset(cfg.Toolset, rellm.ParallelToolCallsDisable)
	}

	return builder.Build()
}

// ParseAgentResult unmarshals the agent's final message into Command.
func ParseAgentResult(report rellm.Report) (Command, error) {
	var cmd Command
	if err := json.Unmarshal([]byte(report.Message), &cmd); err != nil {
		return Command{}, fmt.Errorf("agent output not valid JSON: %w", err)
	}
	return cmd, nil
}

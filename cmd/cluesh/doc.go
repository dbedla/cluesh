// Command cluesh is an AI agent that turns a natural-language demand into
// one copy-paste-ready bash command, explained flag by flag — powered by
// LLM models from external providers (OpenRouter, OpenAI, LM Studio).
//
// # Usage
//
//	cluesh [-c] [--llm <tag>] "<demand>"      generate a command
//	cluesh [-e ["<command>"]]                   explain one: argument, $EDITOR or stdin
//
//	cluesh "list all png files larger than 1MB"
//	cluesh -c "now only the ones modified this week"   continue last conversation
//	cluesh --llm lms-gemma "count lines in *.go"       use a specific model
//	cluesh -e "ls -l"                                  explain this simple command
//	cluesh -e                                           paste/edit a command in $EDITOR
//	cluesh -e < mycmd.sh                                explain a command from a file
//	cluesh --llm-list                                  list configured models
//	cluesh --help
//
// Flags:
//
//	-c, --continue     continue last conversation
//	-e, --explain      explain a command: as argument ("ls -l"), in $EDITOR
//	                   (no argument, terminal), or from stdin (file/pipe)
//	-h, --help         show help and exit
//	    --llm string   use model with tag <tag> this run (default_model_tag if unset)
//	    --llm-list     list configured models and exit
//
// # Passing commands safely
//
// The shell expands $(), backticks and globs inside double quotes before
// cluesh even sees the text — sometimes running a command in the process.
// cluesh never executes anything itself; the shell acts before it.
//
// Generating: pass the demand in single quotes, which the shell never
// touches. A demand that starts with '-' needs the -- end-of-flags marker:
//
//	cluesh 'delete files under $(pwd)/tmp'   $() reaches the LLM as text
//	cluesh -- '-v flag: what does it do?'
//
// Explaining: use -e — the command comes from a file, a pipe or $EDITOR,
// so the shell does no expansion at all:
//
//	cluesh -e "ls -l"                short and simple: as argument
//	cluesh -e < mycmd.sh              command from a file
//	cluesh -e                          opens $EDITOR: paste, save, quit
//
// cluesh echoes `info: explaining: <command>` before the call — if the
// shell expanded something, you'll see it there.
//
// # Install
//
//	go install github.com/dbedla/cluesh/cmd/cluesh@latest
//
// # Configuration
//
// Configuration lives under ~/.cluesh:
//
//	config.json         main settings (default model, clipboard mode, colors, timeout)
//	sysprompt.md        system prompt
//	providers/          one JSON file per provider (openrouter.json, openai.json, lmstudio.json)
//	conversation.jsonl  persisted conversation (created on first ask)
//
// Every missing file is generated with defaults on the first run; existing
// files are never overwritten. Each provider file references an API key via
// an environment variable (e.g. OPENROUTER_API_KEY); lmstudio.json talks to
// a local endpoint and needs no key.
//
// cluesh never executes commands itself; the only side effect is the
// clipboard.
//
// # Links
//
// - GitHub, https://github.com/dbedla/cluesh
// - Tutorial, https://github.com/dbedla/cluesh/blob/main/doc/tutorial.md
// - Homepage, https://rellm.dev/agents/cluesh
// - rellm, https://github.com/dbedla/rellm
package main

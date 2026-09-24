// Command cluesh is an AI agent that turns a natural-language demand into
// one copy-paste-ready bash command, explained flag by flag — powered by
// LLM models from external providers (OpenRouter, OpenAI, LM Studio).
//
// # Usage
//
//	cluesh [-c] [--llm <tag>] "<demand>"
//
//	cluesh "list all png files larger than 1MB"
//	cluesh -c "now only the ones modified this week"   continue last conversation
//	cluesh --llm lms-gemma "count lines in *.go"       use a specific model
//	cluesh --llm-list                                  list configured models
//	cluesh --help
//
// Flags:
//
//	-c, --continue     continue last conversation
//	-h, --help         show help and exit
//	    --llm string   use model with tag <tag> this run (default_model_tag if unset)
//	    --llm-list     list configured models and exit
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
// GitHub:       https://github.com/dbedla/cluesh
// Tutorial:     https://github.com/dbedla/cluesh/blob/main/doc/tutorial.md
// Homepage:     https://rellm.dev/agents/cluesh
// rellm:        https://rellm.dev (https://github.com/dbedla/rellm)
package main

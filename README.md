# cluesh

Natural-language demand in, one copy-paste-ready bash command out.

```
$ cluesh "find all go files with more than 100 lines"
=========

find . -name '*.go' -exec sh -c 'lines=$(wc -l < "$1"); [ "$lines" -gt 100 ] && echo "$1"' _ {} \;

	find
		 .
		  search current directory recursively
		 -name
		  match files by name pattern
		 '*.go'
		  only Go source files
		 -exec
		  run a shell check for each file
		 sh -c 'lines=$(wc -l < "$1"); [ "$lines" -gt 100 ] && echo "$1"'
		  count lines with wc -l and print file if more than 100
		 _
		  placeholder for $0 in sh -c
		 {}
		  current file path passed as $1
		 \;
		  terminate the -exec command

Uses a small sh -c wrapper since find's -exec cannot do numeric comparisons directly. Alternative: find . -name '*.go' | xargs wc -l | awk '$1>100 && $2!="total"{print $2}' — but the find -exec version handles filenames with spaces correctly.

info: command in clipboard

```

cluesh sends your demand to a cheap LLM (OpenRouter, OpenAI or a local LM Studio
endpoint) and prints a colored explanation of every subcommand and flag of the
resulting command, plus the final one-liner — ready to paste.

## Install

```
go install github.com/dbedla/cluesh.git@latest
```

## First run

The first run creates the configuration under `~/.cluesh/`:

```
config.json          main settings (default model, clipboard mode, colors, timeout)
sysprompt.md         system prompt
providers/           one JSON file per provider (openrouter.json, openai.json, lmstudio.json)
conversation.jsonl   persisted conversation (created on the first ask)
```

Every missing file is generated with default settings and reported:
`created /home/you/.cluesh/config.json`. Existing files are never overwritten.

## Setup

Only step required: give the default provider (`openrouter.json`) an API key.
Either export an environment variable (recommended):

```
export OPENROUTER_API_KEY="sk-or-..."
```

or put the key inline in `providers/openrouter.json`:

```json
{
  "api_key": "sk-or-...",
  "models": [
    {
      "tag": "or-glm53flash",
      "name": "z-ai/glm-5.3-flash",
      "temperature": 0.4,
      "reasoning": "low"
    }
  ]
}
```

`temperature` and `reasoning` (`none`|`low`|`medium`|`high`) are optional per
model; unset fields are not sent at all, since some models and providers reject
them (e.g. OpenAI reasoning models reject `temperature` for model `gpt-5.6-luna`).

Each model gets a short `tag`; `default_model_tag` in `config.json` selects the
one used when no flag overrides it. Add more models per provider file
(`openai.json`, `lmstudio.json` — local models need no key).

Main settings in `config.json`:

| field                     | values / meaning                              | default |
|---------------------------|-----------------------------------------------|---------|
| `default_model_tag`       | model tag used when `--llm` is not given      | `or-glm53flash` |
| `put_cmd_in_clipboard`    | `always` \| `never` \| `read-only`            | `always` |
| `colors`                  | `dark` \| `light` \| `none` (`NO_COLOR` wins) | `dark` |
| `execution_timeout_minutes` | agent timeout per run.                      | `5` |

## Usage

```
cluesh [-c] [--llm <tag>] "<demand>"
```

Examples:

```
cluesh "list all png files larger than 1MB"
cluesh -c "now only the ones modified this week"      continue last conversation
cluesh --llm lms-gemma "count lines in *.go"          use a specific model this run
cluesh --llm-list                                     show all configured models
cluesh --help
```

## Full help

```
cluesh — natural-language demand in, one copy-paste-ready bash command out
usage: cluesh [-c] [--llm <tag>] "<demand>"

Flags:
  -c, --continue     continue last conversation
  -h, --help         show help and exit
      --llm string   use model with tag <tag> this run (default_model_tag if unset)
      --llm-list     list configured models and exit

Config location: /home/you/.cluesh
  config.json   main settings (default model, clipboard mode, colors, timeout)
  sysprompt.md   system prompt
  providers/   one JSON file per provider (openrouter.json, openai.json, lmstudio.json)
  conversation.jsonl   persisted conversation (created on first ask)

API key setup (per provider file, e.g. providers/openrouter.json):
  "api_key_env": "OPENROUTER_API_KEY"   key from environment (recommended)
  "api_key": "sk-..."                   inline in the provider file (last resort)
```

## Models list

`cluesh --llm-list` groups every configured model by provider file; the model
matching `default_model_tag` is marked:

```
openrouter.json
	or-glm53flash	z-ai/glm-5.3-flash	(default)
openai.json
	oai-luna	gpt-5.6-luna
lmstudio.json
	lms-gemma	google/gemma-4-26b-a4b
```

## Conversation

Without flags every run starts from a fresh conversation. With `-c` the
previous conversation (from `~/.cluesh/conversation.jsonl`) is loaded and
extended, so you can iterate:

```
cluesh "list files modified this week"
cluesh -c "sort that by size"
```

## Output & clipboard

The command explanation is printed to the console (ANSI colors, `dark`/`light`/`none`
via `colors` in `config.json`, `NO_COLOR` respected). The final command is copied to the
clipboard by shell-out (`wl-copy`/`xclip` on Linux, `pbcopy` on macOS, `clip.exe` on
Windows) — install one of them, or set `put_cmd_in_clipboard`:

| value        | behavior                                              |
|--------------|-------------------------------------------------------|
| `always`     | copy every final command (default)                    |
| `read-only`  | copy only commands that do not modify any file        |
| `never`      | never copy                                            |

cluesh never executes commands itself.

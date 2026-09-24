# Install and configure cluesh

This walkthrough installs cluesh, creates its default configuration, configures an OpenRouter model, runs the first request, and continues the conversation.

## 1. Check the Go binary directory

cluesh installs into the Go binary directory. Check its location:

```console
$  go env | grep GOBIN
GOBIN='~/go/bin'
```

Make sure this directory appears in your shell's `PATH`.

## 2. Install cluesh

Install the latest version from GitHub:

```console
$  go install github.com/dbedla/cluesh/cmd/cluesh@latest
```

Confirm that your shell can find it:

```console
$  which cluesh
~/go/bin/cluesh
```

## 3. Create the default configuration

Run cluesh without a demand. On its first run, cluesh creates the configuration directory and provider files:

```console
$  cluesh
created ~/.cluesh/config.json
created ~/.cluesh/sysprompt.md
created ~/.cluesh/providers/openrouter.json
created ~/.cluesh/providers/openai.json
created ~/.cluesh/providers/lmstudio.json

First run: default configuration created. Finish configuration:
  - set your API key in a provider file, e.g. providers/openrouter.json:
      "api_key_env": "OPENROUTER_API_KEY"
  - review config.json and pick your default model (default_model_tag)
Then run: cluesh "<demand>"
Details: cluesh --help / https://github.com/dbedla/cluesh/blob/main/README.md
```

Inspect the generated directory:

```console
$  tree ~/.cluesh
~/.cluesh
├── config.json
├── providers
│   ├── lmstudio.json
│   ├── openai.json
│   └── openrouter.json
└── sysprompt.md

2 directories, 5 files
```

The files have separate responsibilities:

- `config.json` controls the default model, color mode, clipboard behavior, and execution timeout.
- `providers/openrouter.json` configures OpenRouter credentials and models.
- `providers/openai.json` configures OpenAI credentials and models.
- `providers/lmstudio.json` configures a local LM Studio endpoint.
- `sysprompt.md` contains the system prompt used by cluesh.

## 4. Choose the default model

Inspect the main configuration:

```console
$  cat ~/.cluesh/config.json
{
  "_options": {
    "colors": [
      "dark",
      "light",
      "none"
    ],
    "put_cmd_in_clipboard": [
      "always",
      "never",
      "read-only"
    ]
  },
  "default_model_tag": "or-glm53flash",
  "put_cmd_in_clipboard": "always",
  "colors": "dark",
  "execution_timeout_minutes": 5
}
```

Set `default_model_tag` to a model tag defined in one of the provider files. Here, cluesh selects `or-glm53flash` by default.

You can override this setting for one request with `--llm`.

The remaining settings control:

- `put_cmd_in_clipboard`: whether cluesh copies generated commands to the clipboard.
- `colors`: the terminal color scheme.
- `execution_timeout_minutes`: the request timeout.

## 5. Configure an OpenRouter model

Inspect the OpenRouter provider file:

```console
$  cat ~/.cluesh/providers/openrouter.json
{
  "api_key_env": "OPENROUTER_API_KEY",
  "api_key": "",
  "models": [
    {
      "tag": "or-glm53flash",
      "name": "z-ai/glm-5.3-flash",
      "temperature": 0.4,
      "reasoning": "low"
    },

    {
      "tag": "or-glm53",
      "name": "z-ai/glm-5.3",
      "temperature": 0.4,
      "reasoning": "low"
    }

  ]
}
```

Each model needs a unique `tag` across all provider files. The tag connects a provider model to `default_model_tag` and the `--llm` option.

You can configure credentials in either of two ways:

- Set `api_key_env` to the name of an environment variable containing the key.
- Put the key directly in `api_key`.

When both fields have values, `api_key_env` takes precedence.

For this configuration, export the OpenRouter key before running cluesh:

```console
export OPENROUTER_API_KEY="sk-or-..."
```

## 6. Run the first request

Ask cluesh to produce a command:

```console
$  cluesh "list all files and present output in order of creation date"

Model: z-ai/glm-5.3-flash
temperature=0.4 reasoning=low
=======

ls -lt --time=birth

    ls
         -l
          use long listing format
         -t
          sort by time, newest first
         --time=birth
          sort by file creation time (birth time)

On systems/filesystems without birth time support, use 'ls -lt' (modification time) instead: ls -lt --time=birth || ls -lt

info: tokens: in 192 (cached 0), out 112 (reasoning 0), total 304, cost $0.00008480
LLM claim: <no file modification>
info: command in clipboard
```

cluesh prints:

- The selected model and model settings.
- One generated command.
- An explanation of each command component.
- Relevant portability or safety notes.
- Token usage and cost.
- The model's file-modification claim.
- Clipboard status.

cluesh generates and explains the command but does not execute it.

## 7. Inspect the saved conversation

After the first successful request, cluesh creates `conversation.jsonl`:

```console
$  tree ~/.cluesh
~/.cluesh
├── config.json
├── conversation.jsonl
├── providers
│   ├── lmstudio.json
│   ├── openai.json
│   └── openrouter.json
└── sysprompt.md

2 directories, 6 files
```

`conversation.jsonl` stores the most recent conversation. A normal cluesh run starts a new conversation and replaces this file.

Use `-c` to continue the saved conversation. cluesh then loads the previous context and appends the new exchange.

## 8. Continue the conversation

Ask cluesh to reverse the previous sort order:

```console
$  cluesh -c "revers order"

Model: z-ai/glm-5.3-flash
temperature=0.4 reasoning=low
==

ls -lrt --time=birth

    ls
         -l
          use long listing format
         -t
          sort by time, newest first
         -r
          reverse the sort order (oldest first)
         --time=birth
          sort by file creation time (birth time)

On systems/filesystems without birth time support, use 'ls -lrt' (modification time) instead: ls -lrt --time=birth || ls -lrt

info: tokens: in 307 (cached 0), out 135 (reasoning 8), total 442, cost $0.00003272
LLM claim: <no file modification>
info: command in clipboard
```

Because `-c` loads the previous exchange, cluesh understands that "revers order" refers to the earlier command and adds `-r` without requiring the full request again.

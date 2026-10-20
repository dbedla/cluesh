# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `-e` / `--explain` flag: explain an existing command. Three input paths:
  as argument (`cluesh -e "ls -l"`), in `$EDITOR` (no argument, terminal;
  fallback `vi`), or from stdin (`cluesh -e < mycmd.sh`, `cat f | cluesh -e`).
  The shell does no expansion in the editor/stdin paths; `info: explaining:
  <command>` is echoed before the LLM call as a drift alarm
- explain mode always produces a prose explanation: the explain sys prompt now
  instructs the model to put a one-sentence summary of the command in `notes`
  (the only prose visible for commands without arguments, e.g. `pwd`), and the
  `notes` schema description describes the field's role in both modes
- two sys prompt files: `sysprompt-cmd.md` (generation) and `sysprompt-explain.md`
  (explanation), chosen by the mode — missing files are generated with defaults;
  a pre-existing `sysprompt.md` is kept but renamed to `deprecated-sysprompt.md`,
  so customized prompts are preserved yet no longer used
- the default sys prompt states the shell contract explicitly: bash is the
  default output; another shell only when the user asks for one (named in
  "notes"); other shells listed are for understanding input, not output
- "Passing commands safely" section in README and `doc.go`: single quotes prevent
  shell expansion of `$()`, backticks and globs; `--` marker for demands
  starting with `-`
- `input_limit_in_bytes` in `config.json` (default 65536 bytes): caps the
  input on every path — generation demand and `-e` command alike (argument,
  `$EDITOR`, stdin) — so a misdirected file is refused before it is read
  into memory or sent to the provider; configs written before the field
  existed get it added automatically (existing settings kept, reported with
  `updated …`), and a value below 1 is rejected

## [v0.1.1] - 2026-09-24

### Added

- `all` make target — runs the full pipeline: nuke, list, fmt, build, test, lint, cyclo, coverage
- `list` make target — prints all available make targets
- install and configuration tutorial (`doc/tutorial.md`)
- package documentation (`cmd/cluesh/doc.go`)

### Fixed

- `go-fmt` now formats the actual source directories (`./internal ./cmd` instead of the
  non-existent `./src`)

### Changed

- `go-nuke` simplified — removes the whole `./output` directory instead of individual files
- README updates

## [v0.1.0] - 2026-09-24

Initial release.

- natural-language demand in, one copy-paste-ready bash command out, explained
  flag by flag and subcommand by subcommand
- LLM providers: OpenRouter, OpenAI, LM Studio (local endpoint)
- structured explanation output (plain text and image form), colored console output
- token, cost and reasoning info per request
- command copied to clipboard
- configuration via config file, environment variables and CLI flags with priority rules
- `--help` usage text
- Makefile targets: build, test, lint (vet + golangci-lint), cyclomatic complexity,
  coverage, fmt, nuke

[v0.1.1]: https://github.com/dbedla/cluesh/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/dbedla/cluesh/releases/tag/v0.1.0

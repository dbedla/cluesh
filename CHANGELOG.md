# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

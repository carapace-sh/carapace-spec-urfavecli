# AGENTS.md

Guide for AI agents working in this repository.

## Project Overview

`carapace-spec-urfavecli` is a Go **library** (no `main` package, no binary) that generates [carapace-spec](https://github.com/carapace-sh/carapace-spec) YAML from [urfave/cli/v3](https://github.com/urfave/cli) applications. It scrapes an existing `*cli.Command` command/flag tree and emits a `command.Command` spec consumable by the carapace completion engine.

Consumers call `spec.Register(app)` to inject a hidden `_carapace spec` subcommand into their urfave/cli app, which prints the generated spec to stdout.

## Essential Commands

All commands run from the repo root. The Go module targets `go 1.24`.

```bash
go build -v ./...                          # build (CI step)
go test -v -coverprofile=profile.cov ./... # test with coverage (CI step)
gofmt -d -s .                              # format check (CI fails on diff; use -w to fix)
go vet ./...                               # vet
go install honnef.co/go/tools/cmd/staticcheck@latest && staticcheck ./...  # staticcheck (CI step)
```

CI (`.github/workflows/go.yml`) runs: build, test, gofmt `-d -s` check, goveralls coverage, and staticcheck. All must pass.

## Architecture

The codebase is two source files in a single `package spec` (plus `spec_test.go`):

- **`spec.go`** — Public API and the scraping engine.
  - `Register(app *cli.Command)` appends a hidden `_carapace` command (with a `spec` subcommand) to `app.Commands`. The subcommand's action marshals `Command(app)` to YAML via `gopkg.in/yaml.v2` and prints it.
  - `Command(app *cli.Command) command.Command` is the entry point for programmatic use. It delegates directly to `scrape`.
  - `scrape(c *cli.Command) command.Command` recursively walks the urfave/cli command tree, building a `command.Command` (from `github.com/carapace-sh/carapace-spec/pkg/command`) for each node.

- **`flag.go`** — A `flag` wrapper struct around `cli.Flag` that extracts metadata via interface/type assertions. See "Flag wrapper pattern" below.

### Data flow

```
*cli.Command ──Command()──> scrape()──> command.Command ──yaml.Marshal──> YAML stdout
                              │                │
                              ├── cmd.Flags     ├── Flags (via AddFlag)
                              ├── cmd.Commands  ├── Commands (recursive scrape)
                              └── cmd.Usage     └── Completion.Flag (file completion)
```

## Key Conventions & Gotchas

### Flag wrapper pattern (`flag.go`)

`cli.Flag` is an interface; not all methods an agent might want (e.g. `TakesValue`, `GetUsage`, `GetDefaultText`) are on the base interface. Some live on `cli.DocGenerationFlag`, which is asserted at call time:

```go
if docFlag, ok := f.Flag.(cli.DocGenerationFlag); ok {
    return docFlag.TakesValue()
}
return false // non-conforming flags silently report no value/usage
```

`TakesFile()` uses a **type switch** over the concrete flag types (`*cli.GenericFlag`, `*cli.StringFlag`, `*cli.StringSliceFlag`) and reads each one's `TakesFile` bool field. New urfave/cli flag types that carry a `TakesFile` field must be added here explicitly — there is no interface to satisfy. (v3 removed `PathFlag`; use `StringFlag` with `TakesFile: true` for path completion.)

### `Longhand` and `Shorthand` fields expect bare names

`command.Flag.Longhand` and `.Shorthand` must be the **bare** flag name without dash prefixes (e.g. `"output"`, not `"--output"`; `"o"`, not `"-o"`). carapace-spec's `Flag.format()` adds the `--`/`-` prefixes itself when generating map keys and display strings. Setting `Longhand: "--output"` would produce keys like `----output=` (double-prefixed). The sibling kingpin generator still has this bug; the kong generator does it correctly.

### Default values and the `%q` quoting trap

`flag.Default()` populates the `Default` field on `command.Flag` (carapace-spec v1.8.0+), which triggers extended YAML notation and is consumed by pflag's `Value.Set()` at registration time. The value comes from `DocGenerationFlag.GetDefaultText()` first (the user's explicit `DefaultText`), falling back to `DocGenerationFlag.GetValue()` (the stringified `Value` field) when `DefaultText` is empty.

**Critical gotcha**: `StringFlag` and `StringSliceFlag` wrap their defaults in display quoting (`fmt.Sprintf("%q", val)` or `strconv.Quote` per element) via `GetValue`. Passing quoted strings to pflag's `Value.Set()` would embed literal quote characters in the value. `flag.Default()` calls `strconv.Unquote()` on the result for these types to strip the wrapping quotes. For `StringSliceFlag`, each comma-separated element is unquoted individually. Other flag types (`IntFlag`, `DurationFlag`, `Float64Flag`, etc.) don't apply `%q` and are passed through as-is.

### Hidden commands are dropped

`scrape` skips subcommands where `subcmd.Hidden` is true (`spec.go:65`). The injected `_carapace` command itself is hidden, so it never appears in its own output. Hidden parent flags, however, are **not** filtered — only subcommands.

### Usage strings are truncated at first newline

`flag.Usage()` splits the `GetUsage()` output on the first `\n` and keeps only the first line. Multi-line usage text from urfave/cli flags is silently reduced to its first line in the spec.

### `command.Command` is only partially populated

This library fills `Name`, `Aliases`, `Description`, `Group` (from `cli.Command.Category`), `Hidden`, `Flags`, `Commands`, and `Completion.Flag` (only for file-taking flags, using the literal `$files` macro). It does **not** populate `PersistentFlags`, `ExclusiveFlags`, `Run`, `Completion.Positional*`, `Documentation`, or `Examples` — those carapace-spec fields have no urfave/cli equivalent and are left zero/empty.

### yaml.v2, not yaml.v3

Marshalling uses `gopkg.in/yaml.v2` directly (imported in `spec.go`), even though `carapace-spec` itself uses `yaml.v3` internally. Don't "fix" this — the v2 marshal output is what consumers expect here.

### urfave/cli v3 `Name` field and aliases

In urfave/cli v3, the `Name` field on flags can contain comma-separated aliases (e.g. `"output, o"`), but `FlagNames()` strips everything after the first comma/whitespace. Shorthand aliases should be set via the `Aliases` field, not embedded in `Name`. The `flag.Shorthand()` method scans `Names()` for a single-character entry, which only works when aliases are set correctly via the `Aliases` field.

### v3 API changes from v2

- `cli.App` is removed; use `*cli.Command` as the root command.
- `cli.Context` is removed; callbacks use `(context.Context, *cli.Command)` instead.
- `cli.Command.Subcommands` is renamed to `cli.Command.Commands`.
- `cli.PathFlag` is removed; use `cli.StringFlag` with `TakesFile: true`.
- `StringSliceFlag.Value` expects `[]string` directly (not `cli.NewStringSlice`).
- `GetDefaultText()` only returns the explicit `DefaultText` field; use `GetValue()` to get the stringified actual default value.

## Testing

Tests are in `spec_test.go` (table-driven + targeted tests, standard Go style). Coverage includes:

- `TestDefaultValues` — default extraction across flag types (string, int, string-with-TakesFile, duration, bool, no-default)
- `TestStringSliceFlagDefault` — slice flag defaults with unquoting
- `TestShorthand` — shorthand extraction from `Aliases` field
- `TestFlagKeyFormat` — verifies no `----` double-prefix in flag keys
- `TestTakesFile` — file completion wiring for `TakesFile` flags
- `TestUsageTruncation` — multi-line usage truncated at first newline
- `TestHiddenCommandsFiltered` — hidden subcommands excluded from spec
- `TestSubcommandRecursion` — nested command trees scraped correctly
- `TestExtendedYAMLNotation` — extended YAML `default:` notation emitted correctly
- `TestAliasesAndGroup` — aliases and category mapped to spec fields

Run with `go test -v ./...`.

## Dependencies

- `github.com/urfave/cli/v3 v3.10.1` — the framework being scraped.
- `github.com/carapace-sh/carapace-spec v1.8.0` — target spec format; its `pkg/command` package defines `Command`, `Flag`, `FlagSet`, and `AddFlag`. The `Flag.Default` field (added in v1.8.0) carries flag defaults into the spec's extended YAML notation, consumed by pflag's `Value.Set()` at registration time. Source lives in a sibling repo at `../carapace-spec` in typical carapace-sh dev layouts.
- `gopkg.in/yaml.v2 v2.4.0` — used only for marshalling the final spec.

Dependabot keeps `carapace-spec` and GitHub Actions versions current; most commits in history are dependency bumps merged via PR.

# Slingr CLI

A Go command-line client for Slingr runtime APIs. The invoked executable name
selects an independent configuration, so one installed binary can be used for
multiple apps through symlinks such as `solutions` and `hq`.

## Install

Requires Go 1.26 or newer.

```sh
mkdir -p "$HOME/.local/bin"
GOBIN="$HOME/.local/bin" go install github.com/espinosajuanma/slingr/cmd/slingr@latest
ln -sfn "$HOME/.local/bin/slingr" "$HOME/.local/bin/solutions"
ln -sfn "$HOME/.local/bin/slingr" "$HOME/.local/bin/hq"
```

Make sure `$HOME/.local/bin` is on your `PATH` (for example, add
`export PATH="$HOME/.local/bin:$PATH"` to `~/.bashrc`). Each alias gets its own
settings and login token.

## Project structure

- `cmd/slingr` — executable entry point and dependency wiring.
- `internal/cli` — command dispatch, validation, and request orchestration;
  command handlers and manual metadata live together in an extensible command
  tree. API and configuration persistence are injected through small interfaces.
- `internal/api` — HTTP transport, Slingr URL construction, and API errors.
- `internal/config` — JSON configuration storage and alias-specific paths.
- `internal/terminal` — hidden password input.
- `internal/*/*_test.go` — package-focused tests.
- `legacy/slingr.bash` — archived Bash implementation for reference.

## Configuration

Running the CLI with no command displays help. A command that needs app settings
starts a setup wizard if app, environment, or email is missing. Existing values
are shown as defaults; pressing Enter keeps the current value.

Set the app, environment, and email for each alias:

```sh
solutions config app my-solutions-app
solutions config env dev
solutions config email juan@example.com
```

`config KEY VALUE` stores any key and value; `config KEY` prints that key's
value, and `config` lists all settings. Multiple value arguments are joined
with spaces. Configuration is stored as JSON in
`~/.config/<executable-name>/config.json` (or `$XDG_CONFIG_HOME/<name>`).
`CONFIG_DIR` overrides the directory, and `EXE` overrides the executable name
used for configuration.

## Commands

- `help [COMMAND [SUBCOMMAND...]]` — display the command index or a man-page-style page.
- `config [KEY [VALUE...]]` — list, read, or set configuration.
- `current` — print the configured app and environment, such as `solutions-dev`.
- `version` — display the CLI version.
- `path` — display the configuration file path.
- `app`, `env`, `email` — aliases for reading or setting those config keys.
- `login [PASSWORD]` — log in; prompts without echo when the password is
  omitted.
- `whoami` — fetch the currently authenticated user.
- `query` / `q ENTITY [--field VALUE ...]` — query records.
- `aggregate ENTITY JSON_OPERATIONS` — run an aggregate query using an ordered
  JSON array of operations such as `match`, `group`, and `count`.
- `upload FILE_PATH` — upload a local file and print its metadata, including
  `fileId`.
- `download FILE_ID OUTPUT_PATH` — download a file; use `-` as the output path
  to write the file bytes to standard output. Record files can be downloaded
  with `download ENTITY RECORD_ID FILE_ID OUTPUT_PATH`.
- `get ENTITY RECORD_ID` — fetch a record.
- `new` / `add ENTITY [--field VALUE ...]` — create a record.
- `remove` / `rm ENTITY RECORD_ID` — delete a record.
- `edit ENTITY RECORD_ID [--field VALUE ...]` — fetch, merge changes, and
  update a record.
- `action ENTITY ACTION_NAME [RECORD_ID] [--field VALUE ...]` — run an entity
  or record action.
- `logout` — log out and remove the current app/environment token.
- `token` — print the current token.

Nested fields use dotted names, for example
`--daySettings.date '2026-10-14'`. A flag value that is valid JSON is decoded as
JSON, supporting strings, numbers, booleans, null, arrays, and objects. Other
values are kept as plain text. To explicitly pass a JSON string, include its
JSON quotes, e.g. `--status='"ready"'`.

Use `solutions help` for the command index or `solutions help edit` for the
full manual page for a command.

```sh
solutions query holidays --country Argentina
solutions new holidays --country Argentina --daySettings.date 2026-10-14
solutions edit holidays d987485b-488e-43c9-b5d4-77ba5dd6950b --active true
solutions action holidays approve --reason 'manual review'
solutions action holidays approve d987485b-488e-43c9-b5d4-77ba5dd6950b --approved true
```

The first action form calls the entity action; supplying a record ID calls the
action for that record. API requests use the configured app's runtime API at
`https://<app>.slingrs.io/<env>/runtime/api`.

Aggregate query bodies are ordered arrays of operations, as described by
Slingr's [aggregate query API](https://platform-docs.slingr.io/dev-reference/rest-apis/apps-api/#aggregate-queries):

```sh
solutions aggregate contacts '[{"project":"company,numberOfSkills"},{"group":{"by":"company","totalSkills":"sum(numberOfSkills)"}}]'
```

Files are uploaded through `POST /files` and downloaded through `GET /files/{id}`
or the record-files endpoint. See the [files handling reference](https://platform-docs.slingr.io/dev-reference/rest-apis/apps-api/#files-handling).

## Bash completion

Register completion for each alias in `.bashrc`:

```bash
complete -C solutions solutions
complete -C hq hq
```

Completion offers the supported top-level commands.

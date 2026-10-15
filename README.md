[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Active](https://img.shields.io/badge/Status-Active-green)](https://guide.unitvectorylabs.com/bestpractices/status/#active) 

# ghscanscope

A local-first workbench for syncing, triaging, and finding patterns in GitHub code-scanning alerts across repositories.

## Key Features

- Sync an entire organization or a single repository's open code-scanning alerts.
- Detect GitHub repositories from local checkouts, including subdirectories and worktrees.
- Configure organization folders with a `.ghscanscope` property file.
- Share findings across working directories in SQLite (the default), or use PostgreSQL for a shared server cache.
- Manage schema upgrades and downgrades with the `migrate` command.
- Read color-coded CLI output and follow sync progress; set `NO_COLOR` for plain text.
- Export cached repository findings as plain text or JSON, filtered by priority and scanner.
- Compare findings by scanner and rule with search, filters, and GitHub links.

## Quick Start

1. Install:

   ```sh
   go install github.com/UnitVectorY-Labs/ghscanscope@latest
   ```

2. Authenticate with `gh auth login`, or set `GITHUB_TOKEN` or `GH_TOKEN`.

3. Sync and browse:

   ```sh
   ghscanscope sync --org UnitVectorY-Labs
   ghscanscope web
   ```

Open `http://127.0.0.1:8080`. The database defaults to
`~/.local/share/ghscanscope/ghscanscope.db`, with its directory created automatically.

SQLite needs no server or new configuration. To use PostgreSQL, pass a connection
URL with `--db` or set `GHSCAN_SCOPE_DB` for all commands:

```sh
export GHSCAN_SCOPE_DB='postgresql://scanner@db.example.com/ghscanscope?sslmode=verify-full'
ghscanscope migrate up
ghscanscope sync --org UnitVectorY-Labs
ghscanscope web
```

See [Databases](docs/DATABASE.md) for choosing a backend, migrations, and operations.

From a GitHub checkout, `ghscanscope sync` refreshes just that repository using
its `origin` remote. From an organization folder such as `~/github`, set
`org=UnitVectorY-Labs` in `.ghscanscope` and run `ghscanscope sync` there.
Command-line flags and environment variables override these defaults.

Inspect cached findings from a checkout, or select a repository explicitly:

```sh
ghscanscope findings
ghscanscope findings --repo UnitVectorY-Labs/ghscanscope --level critical,high --json
```

The default text output is a short summary. Filters or `--verbose` (also `--details`) show finding
messages, file and line locations, rules, revisions, and scanner metadata.
`--ref refs/heads/main` selects a cached Git ref.
`--json` always includes full finding records. Reading the cache needs no GitHub
credentials; run `sync` to refresh it.

Human-readable output uses color by default. Define `NO_COLOR` (even with an
empty value) to disable it. JSON is always uncolored. Sync shows live progress
on stderr, with counted bars for caching and activity while downloading.

See [Installation](docs/INSTALL.md), [Usage](docs/USAGE.md), and
[Examples](docs/EXAMPLES.md) for the full reference. The
[documentation home](docs/README.md) lists the main capabilities.

## Exploring findings

The alerts page starts with **Critical & high** findings grouped by scanner and
rule, ordered by priority and then the number of affected repositories. Use the
priority filter to broaden that scope. Scanner, repository, priority, and
other filters can be combined; active filters are visible and individually
removable. Search matches scanner text, finding messages, rules, tags, and paths.
Filter and sort URLs can be bookmarked.

Open an issue group to compare affected repositories and inspect individual
occurrences. Dedicated finding pages show the scanner message, location,
reported revision, and scan metadata, with source and alert links to GitHub.
Back links return to the filtered view you came from. Aggregate counts show
**matching / total before filters**, including priority breakdowns. Listing
totals cover all cached open findings; detail-page totals cover the
selected repository or scanner rule. Repository search matches names and
descriptions. Zero matching alerts does not establish scan coverage.

The light theme uses white surfaces with blue and orange accents. The header
theme toggle switches to dark mode; the preference is remembered in your browser.
Scanner badges use a muted palette and identify which tool reported each finding.

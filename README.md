[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Active](https://img.shields.io/badge/Status-Active-green)](https://guide.unitvectorylabs.com/bestpractices/status/#active) 

# ghscanscope

A local-first workbench for syncing, triaging, and finding patterns in GitHub code-scanning alerts across repositories.

## Key Features

- Sync an entire organization or a single repository's open code-scanning alerts.
- Detect GitHub repositories from local checkouts, including subdirectories and worktrees.
- Configure organization folders with a `.ghscanscope` property file.
- Share findings across working directories in a local SQLite database.
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

From a GitHub checkout, `ghscanscope sync` refreshes just that repository using
its `origin` remote. From an organization folder such as `~/github`, set
`org=UnitVectorY-Labs` in `.ghscanscope` and run `ghscanscope sync` there.
Command-line flags and environment variables override these defaults.

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
totals cover all locally stored open findings; detail-page totals cover the
selected repository or scanner rule. Repository search matches names and
descriptions. Zero matching alerts does not establish scan coverage.

The light theme uses white surfaces with blue and orange accents. The header
theme toggle switches to dark mode; the preference is remembered in your browser.
Scanner badges use a muted palette and identify which tool reported each finding.

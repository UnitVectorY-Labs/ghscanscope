[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Active](https://img.shields.io/badge/Status-Active-green)](https://guide.unitvectorylabs.com/bestpractices/status/#active) 

# ghscanscope

A local-first workbench for syncing, triaging, and finding patterns in GitHub code-scanning alerts across repositories.

## Usage

```sh
go build -o ghscanscope .
./ghscanscope sync --org UnitVectorY-Labs
./ghscanscope web
```

The SQLite database defaults to `.ghscanscope.db` and the web UI to
`http://127.0.0.1:8080`. Authentication is read from `GITHUB_TOKEN`, `GH_TOKEN`,
or the active `gh auth login` credential. Configuration can also be supplied by
`GHSCAN_SCOPE_ORG`, `GHSCAN_SCOPE_REPO`, `GHSCAN_SCOPE_DB`, and
`GHSCAN_SCOPE_ADDR`.

`sync --repo OWNER/REPO` refreshes just one repository. A full organization sync
catalogs every repository, including repositories without alerts, and refreshes
all open code-scanning alerts. Alerts absent from a successful refresh are kept
in SQLite as non-open history and omitted from the dashboard.

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

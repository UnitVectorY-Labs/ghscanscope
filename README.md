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

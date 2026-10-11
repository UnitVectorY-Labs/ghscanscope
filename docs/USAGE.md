---
layout: default
title: Usage
nav_order: 3
permalink: /usage
---

# Usage
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## Commands

```text
ghscanscope sync [--org ORG] [--repo OWNER/REPO] [--db PATH]
ghscanscope findings [--org ORG] [--repo OWNER/REPO] [--db PATH] [--level LEVEL] [--scanner NAME] [--details] [--json]
ghscanscope web [--db PATH] [--addr ADDRESS]
ghscanscope version
```

Use `ghscanscope sync --help`, `ghscanscope findings --help`, or `ghscanscope web --help` for flag descriptions. Place flags after the command.

### sync

Download repository metadata and open GitHub code-scanning alerts into SQLite. The command prints the number of repositories and open alerts synced, the organization, and elapsed time.

| Flag | Environment variable | Default |
| --- | --- | --- |
| `--org` | `GHSCAN_SCOPE_ORG` | Determined from the selected repository or `.ghscanscope` |
| `--repo` | `GHSCAN_SCOPE_REPO` | Detected checkout, otherwise organization-wide sync |
| `--db` | `GHSCAN_SCOPE_DB` | `~/.local/share/ghscanscope/ghscanscope.db` |

A full organization sync catalogs every repository, including repositories without alerts, and refreshes open code-scanning alerts for the organization. A single repository sync refreshes only that repository and leaves other repositories' alerts unchanged.

After a successful alert refresh, previously open alerts absent from the response remain in SQLite as non-open history and disappear from the dashboard. A failed alert fetch does not retire previously stored alerts. Sync status and errors are recorded locally.

### findings

Read one repository's cached open code-scanning findings without contacting GitHub or requiring authentication. From a GitHub checkout (including subdirectories and worktrees), run `ghscanscope findings`. From any other directory, select a repository with `--repo OWNER/REPO` or `--org OWNER --repo REPO`.

| Flag | Environment variable | Default |
| --- | --- | --- |
| `--org` | `GHSCAN_SCOPE_ORG` | Selected repository's owner |
| `--repo` | `GHSCAN_SCOPE_REPO` | GitHub `origin` checkout detection |
| `--db` | `GHSCAN_SCOPE_DB` | `~/.local/share/ghscanscope/ghscanscope.db` |
| `--level` | — | All priorities, including unknown |
| `--scanner` | — | All scanners |
| `--details` | — | Off; filters enable detailed text automatically |
| `--json` | — | Off; when enabled, always includes full finding records |

With no filters, text output summarizes the repository, metadata cache timestamp, matching and total open counts, priority counts, and scanner counts. Use `--details` to print all finding details. Supplying either filter also prints detailed findings. JSON always contains full records, even without filters; `--details` does not change JSON output.

`--level` accepts `critical`, `high`, `medium`, `low`, and `unknown`. These are the same canonical priorities used by the web view: scanner `error` maps to high, `warning` to medium, and `note`/`recommendation` to low. The original reported severity and its source are retained in each finding. `--scanner` matches the complete cached tool name (for example, `CodeQL` or `Semgrep OSS`). Both filters ignore case, accept repeated flags or comma-separated values, and select any listed value within a filter. Level and scanner filters are combined with AND.

```sh
ghscanscope findings
ghscanscope findings --level critical,high
ghscanscope findings --repo UnitVectorY-Labs/ghscanscope --scanner CodeQL --details
ghscanscope findings --org UnitVectorY-Labs --repo ghscanscope --level high --json > findings.json
```

Detailed text and JSON include the alert number, priority, reported severity and source, scanner name/version/GUID, rule ID/name/description/tags, finding message, file path, start/end lines and columns, ref, commit SHA, analysis key, environment, category, alert URL, and created/updated timestamps. Findings are ordered by priority (critical first, unknown last), then alert number. Source code and additional remediation text are not stored in the cache and are not fetched by this command. Missing locations remain empty paths and zero line/column values in JSON; text explicitly marks unreported locations.

The JSON object has `repository`, `repository_url`, `repository_cached_at`, `total_open`, `matching`, `by_level`, `by_scanner`, and `findings`. Counts by level/scanner describe the matching findings; `total_open` is the repository's count before filters. `findings` is always an array, including `[]` for no matches. Individual records use snake_case keys, such as `rule_description`, `reported_severity`, `path`, `start_line`, and `commit_sha`. JSON contains no text preamble; errors go to standard error.

This exposes the current cached open alerts and each alert's most recent stored instance, just as the web view does. SQLite does not retain separate scan snapshots. `repository_cached_at` is the repository metadata refresh time, not a scanner completion time or a guarantee that the last alert refresh succeeded. A failed alert fetch leaves previously cached findings available. Run `sync` to refresh the cache before exporting when freshness matters. Retired alerts are excluded. Zero matching findings does not establish scan coverage.

A missing database or uncached repository produces an error instructing you to sync first. A cached repository with no open findings, or a filter with no matches, succeeds with zero counts. Exit status does not depend on finding severity.

Flags override environment variables. An explicit repository overrides checkout detection, and an explicit owner must match the repository owner, ignoring case. A short repository name requires an explicit owner. As with `sync`, an explicit owner alone suppresses checkout detection; `findings` then fails because it requires one repository. Organization directory property files do not select a repository for this command.

### web

Serve the local dashboard. This command uses the same database default as `sync`; organization and repository configuration do not apply. Stop the server with Ctrl+C.

| Flag | Environment variable | Default |
| --- | --- | --- |
| `--db` | `GHSCAN_SCOPE_DB` | `~/.local/share/ghscanscope/ghscanscope.db` |
| `--addr` | `GHSCAN_SCOPE_ADDR` | `127.0.0.1:8080` |

The dashboard includes actions to refresh stored organizations or repositories using the current GitHub credentials. Viewing stored findings does not require fetching them again.

### version

Print the application version, Go version, operating system, and architecture.

## Configuration Precedence

Command-line flags override their corresponding environment variables. Unset or empty environment variables fall back to directory defaults. For `sync`, organization and repository selection then follow these rules:

1. An explicit `--repo` or `GHSCAN_SCOPE_REPO` selects that `OWNER/REPO`. Its owner supplies the organization unless an organization is also explicitly set. An explicit organization must match the repository owner, ignoring case.
2. An explicit `--org` or `GHSCAN_SCOPE_ORG`, without an explicit repository, selects the entire organization even inside a checkout.
3. With neither explicitly set, a GitHub checkout selects its repository and owner from the `origin` remote.
4. Otherwise, `org` in the current directory's `.ghscanscope` file selects an organization-wide sync.
5. If no organization can be resolved, the command exits with an error explaining how to supply one.

An explicit empty `--repo=` disables checkout detection, allowing the directory's organization configuration to apply. An explicit empty `--org=` produces a missing-organization error.

### Directory Property File

Create `.ghscanscope` in the directory where you run `sync`:

```properties
# Organization to sync from this folder
org=UnitVectorY-Labs
```

The supported property is `org`. Lines use `key=value`; surrounding whitespace is ignored. Blank lines and whole-line comments starting with `#` or `!` are allowed. Values are unquoted literal text. If `org` appears more than once, the last value wins. Unsupported properties, missing `=`, and empty values produce an error when the file is needed.

Only the current directory's file is read; parent directories are not searched. A detected GitHub checkout takes priority over this file. Explicit organization/repository selection bypasses the file.

### Checkout Detection

Git must be available on `PATH`. ghscanscope reads `git remote get-url origin` from the current working directory, so detection also works in repository subdirectories and Git worktrees.

Supported GitHub.com remote forms include:

```text
https://github.com/OWNER/REPO.git
git@github.com:OWNER/REPO.git
ssh://git@github.com/OWNER/REPO.git
```

The `.git` suffix is optional. Other remote names and hosts are not used for detection. If Git is unavailable, no `origin` exists, or `origin` is not a GitHub.com repository, selection falls back to the directory property file. Detection reads Git metadata; syncing downloads alerts from GitHub rather than scanning files on disk.

## Database Location

The default is `~/.local/share/ghscanscope/ghscanscope.db` on all operating systems, including macOS and Windows. `~` refers to the current user's home directory. The application uses this convention regardless of `XDG_DATA_HOME` or platform-specific application data folders.

Missing parent directories are created automatically. All working directories share the default database, which can hold multiple organizations. Override it with `--db` or `GHSCAN_SCOPE_DB`; relative paths are resolved from the current working directory. Quote or escape custom paths as required by your shell.

Earlier versions defaulted to `.ghscanscope.db` in the working directory. Existing databases are not moved automatically. Continue using one with `--db .ghscanscope.db` on `sync`, `findings`, and `web`, or move it to the new default location while the application is stopped. SQLite may create adjacent `-wal` and `-shm` files while running.

## Exploring Findings

The alerts page starts with **Critical & high** findings grouped by scanner and rule, ordered by priority and then the number of affected repositories. Broaden the priority filter to see other findings. Combine scanner, repository, priority, and other filters; active filters are visible and individually removable. Search matches scanner text, finding messages, rules, tags, and paths. Filter and sort URLs can be bookmarked.

Open an issue group to compare affected repositories and inspect individual occurrences. Finding pages show scanner messages, locations, reported revisions, and scan metadata, with source and alert links to GitHub. Back links preserve the filtered view.

Aggregate counts show **matching / total before filters**, including priority breakdowns. Listing totals cover all locally stored open findings; detail-page totals cover the selected repository or scanner rule. Repository search matches names and descriptions. Zero matching alerts does not establish scan coverage.

The header theme toggle switches between light and dark modes and remembers your preference in the browser. Scanner badges identify the tool that reported each finding.

## Authentication

`GITHUB_TOKEN` takes priority over `GH_TOKEN`, followed by the active GitHub CLI credential for `github.com`. See [Installation](INSTALL.md#authentication).

## Exit Status

- `0`: command completed successfully
- `1`: a configuration, flag parsing, database, server, or GitHub API error occurred (also returned for command help)
- `2`: no command was supplied or the command is unknown

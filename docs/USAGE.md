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
ghscanscope web [--db PATH] [--addr ADDRESS]
ghscanscope version
```

Use `ghscanscope sync --help` or `ghscanscope web --help` for flag descriptions. Place flags after the command.

### sync

Download repository metadata and open GitHub code-scanning alerts into SQLite. The command prints the number of repositories and open alerts synced, the organization, and elapsed time.

| Flag | Environment variable | Default |
| --- | --- | --- |
| `--org` | `GHSCAN_SCOPE_ORG` | Determined from the selected repository or `.ghscanscope` |
| `--repo` | `GHSCAN_SCOPE_REPO` | Detected checkout, otherwise organization-wide sync |
| `--db` | `GHSCAN_SCOPE_DB` | `~/.local/share/ghscanscope/ghscanscope.db` |

A full organization sync catalogs every repository, including repositories without alerts, and refreshes open code-scanning alerts for the organization. A single repository sync refreshes only that repository and leaves other repositories' alerts unchanged.

After a successful alert refresh, previously open alerts absent from the response remain in SQLite as non-open history and disappear from the dashboard. A failed alert fetch does not retire previously stored alerts. Sync status and errors are recorded locally.

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

Command-line flags override their corresponding environment variables. Unset or empty environment variables fall back to directory defaults. Organization and repository selection then follow these rules:

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

Earlier versions defaulted to `.ghscanscope.db` in the working directory. Existing databases are not moved automatically. Continue using one with `--db .ghscanscope.db` on both `sync` and `web`, or move it to the new default location while the application is stopped. SQLite may create adjacent `-wal` and `-shm` files while running.

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

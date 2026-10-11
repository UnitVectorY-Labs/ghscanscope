---
layout: default
title: Examples
nav_order: 4
permalink: /examples
---

# Examples
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## Configure an Organization Folder

Create `~/github/.ghscanscope` as a property file:

```properties
org=UnitVectorY-Labs
```

Run sync from that folder:

```bash
cd ~/github
ghscanscope sync
ghscanscope web
```

The sync catalogs the organization and refreshes its open code-scanning alerts. Open `http://127.0.0.1:8080` to browse them.

## Sync a Checked-Out Repository

```bash
cd ~/github/ghscanscope
ghscanscope sync
```

With an `origin` pointing to `https://github.com/UnitVectorY-Labs/ghscanscope.git` (or its SSH equivalent), this refreshes only `UnitVectorY-Labs/ghscanscope`. No property file is required. Running from a subdirectory or Git worktree selects the same repository.

The organization configuration in `~/github/.ghscanscope` is not read from this checkout. The checkout's remote supplies its owner and repository, and the shared database accumulates findings across runs.

## Override the Local Defaults

Select a whole organization even when running inside a checkout:

```bash
ghscanscope sync --org UnitVectorY-Labs
```

Select a repository from any folder; its owner supplies the organization:

```bash
ghscanscope sync --repo UnitVectorY-Labs/ghorgsync
```

Supply defaults through environment variables and override them with flags:

```bash
export GHSCAN_SCOPE_ORG=UnitVectorY-Labs
ghscanscope sync
ghscanscope sync --org AnotherOrganization
unset GHSCAN_SCOPE_ORG
```

If both organization and repository are explicitly set, their owners must agree. For example:

```bash
ghscanscope sync --org UnitVectorY-Labs --repo UnitVectorY-Labs/ghorgsync
```

## Use a Separate Database

Pass the same database to both commands:

```bash
ghscanscope sync --org UnitVectorY-Labs --db ./data/alerts.db
ghscanscope web --db ./data/alerts.db --addr 127.0.0.1:9090
```

Or use an environment variable shared across folders:

```bash
export GHSCAN_SCOPE_DB="$HOME/scan-data/alerts.db"
ghscanscope sync --org UnitVectorY-Labs
ghscanscope web
```

The parent directory is created automatically. To keep using the previous working-directory database, pass `--db .ghscanscope.db` to both commands.

## Compare Findings Across Repositories

1. Sync an organization and start the web UI.
2. Open the alerts page and inspect a scanner/rule group.
3. Broaden the default **Critical & high** priority filter if needed.
4. Add repository or scanner filters, or search for a rule, message, tag, or path.
5. Open an occurrence to inspect its location and GitHub links.
6. Bookmark the filtered URL to return to that view.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| Organization is required | Set `--org`, `GHSCAN_SCOPE_ORG`, or `org` in the current directory's `.ghscanscope`; for checkout detection, check `git remote get-url origin` |
| Wrong sync scope | Check environment variables; an explicit organization selects a full organization sync, while an explicit repository selects one repository |
| Property file error | Use unquoted `org=NAME`; only `org`, blank lines, and whole-line comments are supported |
| Repository owner does not match organization | Remove the organization override or set it to the selected repository's owner |
| GitHub API permission error | Check the token's repository and code-scanning access; `GITHUB_TOKEN` overrides `GH_TOKEN` and GitHub CLI authentication |
| Database directory cannot be created | Check permissions or select a writable path with `--db` |
| Previously synced data is missing | Use the same `--db` path for sync and web; older working-directory databases are not automatically moved |
| No findings are visible | Broaden the priority and other filters, check sync status, and confirm that GitHub has open code-scanning alerts |
| Listen address is already in use | Choose another address, such as `--addr 127.0.0.1:9090` |

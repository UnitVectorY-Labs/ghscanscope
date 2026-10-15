---
layout: default
title: ghscanscope
nav_order: 1
permalink: /
---

# ghscanscope

A local-first workbench for syncing, triaging, and finding patterns in GitHub code-scanning alerts across repositories.

**ghscanscope** downloads repository metadata and open code-scanning alerts into SQLite (the default) or PostgreSQL, then serves a local web UI for exploring those findings. Sync an entire organization from a configured folder or just one repository from its checkout.

## Key Features

- **Organization and repository sync** — catalog an organization or refresh a single repository
- **Checkout detection** — use the current repository's GitHub `origin` remote without configuration
- **Directory configuration** — set an organization's default in a `.ghscanscope` property file
- **Database choice** — keep SQLite without configuration, or select PostgreSQL with `--db` / `GHSCAN_SCOPE_DB`
- **Versioned migrations** — apply schema upgrades and downgrades with `migrate`
- **Shared local database** — store findings in `~/.local/share/ghscanscope/ghscanscope.db` across working directories
- **Cross-repository triage** — group findings by scanner and rule, then compare affected repositories
- **Search and filters** — combine repository, scanner, priority, and other filters with bookmarkable URLs
- **Readable CLI output** — color-coded output with `NO_COLOR` support and live sync progress
- **CLI exports** — inspect one repository’s cached findings as text or JSON with priority, scanner, and Git ref filters
- **Finding details** — inspect messages, locations, scan metadata, and links to GitHub
- **Local history** — retain alerts that disappear from a successful refresh as non-open history
- **Light and dark themes** — remember your theme preference in the browser

Start with [Installation](INSTALL.md), read the command and configuration reference in [Usage](USAGE.md), or follow the workflows in [Examples](EXAMPLES.md).

See [Databases](DATABASE.md) for backend selection, operational tradeoffs, migrations, and PostgreSQL integration testing.

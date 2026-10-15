---
layout: default
title: Databases
nav_order: 5
permalink: /database
---

# Databases
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## Choose where the cache lives

SQLite and PostgreSQL are fully supported storage backends for the same workbench. Both store repository metadata, cached alerts, and sync status. Crawling, finding priorities, CLI exports, and dashboard behavior are shared. Changing databases changes where the cache lives; it does not change which GitHub repositories are selected or the permissions needed to fetch their alerts.

SQLite remains the default and a first-class option. Use it for a personal workbench, a single host, or a deployment where a portable file and minimal administration are useful. There is no database service to install, and existing SQLite configurations continue to work.

Choose PostgreSQL when several hosts need to reach the same cache, or when you already operate PostgreSQL and want centralized access control, backups, and monitoring. PostgreSQL adds a network dependency and database administration. It does not turn ghscanscope into an authenticated multi-user web service or coordinate overlapping crawls.

| Consideration | SQLite | PostgreSQL |
| --- | --- | --- |
| Setup | Included in the binary; no server | Provision a database and credentials |
| Configuration | A filename, or no configuration | A `postgres://` or `postgresql://` URL |
| Location | Local disk on the application host | A reachable database server |
| Concurrency | One application connection per process; writes serialize | A bounded connection pool per process; server handles connections |
| Operations | Protect and back up the database file | Manage availability, TLS, roles, capacity, and backups |
| Typical fit | Local triage and single-host services | Shared caches and managed infrastructure |

## Configuration

Every database command accepts the same `--db` flag and `GHSCAN_SCOPE_DB` environment variable. A PostgreSQL URL selects PostgreSQL; other values retain SQLite path semantics. `--db` overrides the environment variable. An empty `--db=` selects the default SQLite file even if the environment selects PostgreSQL.

```sh
# No new configuration needed: SQLite remains the default.
ghscanscope sync --org UnitVectorY-Labs
ghscanscope web

# Select another SQLite file.
ghscanscope sync --org UnitVectorY-Labs --db ./data/alerts.db

# Select PostgreSQL for this command.
ghscanscope web --db 'postgresql://scanner@db.example.com/ghscanscope?sslmode=verify-full'

# Select the same database for every command in this shell.
export GHSCAN_SCOPE_DB='postgresql://scanner@db.example.com/ghscanscope?sslmode=verify-full'
ghscanscope migrate up
ghscanscope sync --org UnitVectorY-Labs
ghscanscope findings --repo UnitVectorY-Labs/ghscanscope --json
ghscanscope web
```

Use the same target for `sync`, `findings`, `web`, and `migrate`. The `.ghscanscope` directory property file configures the organization only; it does not select a database. GitHub credentials and database credentials serve separate purposes. Reading cached findings needs database access but does not require GitHub authentication.

### SQLite

The default file is `~/.local/share/ghscanscope/ghscanscope.db` on every operating system. Missing parent directories are created automatically. Relative paths are relative to the current working directory; a custom `~` must be expanded by your shell. Older working-directory caches remain usable with `--db .ghscanscope.db`; they are not moved automatically.

SQLite uses write-ahead logging (WAL) and enforces foreign keys. Keep the file on a local filesystem suitable for SQLite locking. Adjacent `-wal` and `-shm` files are normal while the application is running. Use SQLite's backup facilities for an online backup, or stop all processes using the database before copying it and account for any outstanding WAL. Copying only the main file during active writes can omit committed data. Protect the directory and backups: cached findings can contain private repository names, paths, messages, and other sensitive metadata.

SQLite writes serialize. Schedule one crawl at a time per cache and avoid running schema migrations while other processes use it. If a write reports that the database is locked, check for another writer before retrying. A network database is generally a better fit than sharing a SQLite file across hosts.

### PostgreSQL

PostgreSQL 18 is exercised by the integration suite. Provision the database before running ghscanscope; the application creates its own tables, not the database or login role. Use a dedicated database, or a dedicated schema selected through the URL's `search_path` parameter. No PostgreSQL extensions or superuser privileges are required. The role needs access to the selected schema, permission to create and alter the application objects and migration history, and permission to read and write application tables and generated ID sequences.

Both `postgres://` and `postgresql://` URLs are supported through the [pgx driver](https://github.com/jackc/pgx). Keyword/value DSNs are not database selectors. URL-encode reserved characters in credentials, and quote URLs in shell commands. Prefer an environment variable supplied by your deployment's secret manager over a password in a command line or shell history. Keep connection URLs out of logs and source control.

Set TLS options explicitly for your environment. For a remote server, `sslmode=verify-full` verifies both the certificate chain and server hostname; configure trusted certificates as needed. `sslmode=disable` is used only in the isolated local examples below. See the [PostgreSQL TLS documentation](https://www.postgresql.org/docs/current/libpq-ssl.html) for the verification modes.

Each process uses a pool of up to ten database connections. Include all `web`, `sync`, and `findings` processes when budgeting server connections. Ensure the server remains available to the web process; PostgreSQL is the active cache, not a background replica of a SQLite file. Use direct server connections or session pooling for migration commands, which use a session advisory lock.

Back up PostgreSQL using your normal database backup and restore procedures, such as `pg_dump` and `pg_restore`, and test restores. Retain migration history with application data. Monitor disk space, connection usage, and failed syncs. Database authentication does not protect the dashboard: its default listener remains `127.0.0.1:8080`, and exposing it requires appropriate access controls outside ghscanscope.

## Schema upgrades and downgrades

Schema changes use the maintained [Goose migration library](https://github.com/pressly/goose). Migrations ship inside the binary, so users do not need a separate migration tool or migration files. Both backends use the same numbered migration history and command semantics, with database-specific schema definitions where required.

Normal `sync`, `findings`, and `web` startup automatically applies pending upgrades. This preserves existing SQLite startup behavior, including adopting caches created before migration version tracking. Existing alerts and scanner-reported severity are retained during adoption. Upgrades are recorded once rather than rerunning data backfills on each startup. For a planned rollout, stop other users of the cache, back it up, and apply migrations explicitly before starting the new binary.

```sh
# Uses GHSCAN_SCOPE_DB, or the default SQLite file.
ghscanscope migrate status
ghscanscope migrate up

# Flags go before the migration action.
ghscanscope migrate --db ./data/alerts.db up
ghscanscope migrate --db ./data/alerts.db down
ghscanscope migrate --db ./data/alerts.db up 2
ghscanscope migrate --db ./data/alerts.db down 1
```

| Action | Effect |
| --- | --- |
| `status` | Print the current version; does not apply application migrations |
| `up` | Apply all pending upgrades |
| `up VERSION` | Apply upgrades through the target version |
| `down` | Revert the most recent migration |
| `down VERSION` | Revert migrations until the target version is reached |

Commands print `Database version: N` on success. Version `0` means no application schema is applied. `status` can initialize the database file and migration bookkeeping on a new target, but it does not create application tables. A missing PostgreSQL database or unreachable server is an error. Only versions included in the binary are valid targets.

The current migrations are:

| Version | Upgrade | Downgrade |
| --- | --- | --- |
| 1 | Create the cache tables, or adopt and complete a legacy SQLite schema | Drop all application tables and their cached data |
| 2 | Store canonical priority alongside reported severity and its source; backfill existing alerts | Restore the reported value to the legacy severity field and remove priority/provenance columns |

Downgrades are deliberate administrative operations and can lose data. In particular, **`down 0` deletes all cached repositories, alerts, and sync history**. Downgrading version 2 loses explicit severity provenance and priority overrides; reapplying it derives priorities from the reported values. Back up before downgrading. Stop application processes first, and keep them stopped until the schema and binary versions match your intended deployment. Starting this binary again automatically reapplies pending upgrades.

Each migration runs in a transaction, including its version record. A failed migration rolls back that migration; earlier successfully applied migrations remain applied. Correct the reported cause and retry. Do not manually edit Goose's version table to bypass a failure. PostgreSQL migrations use an advisory lock to serialize migration runners. This protects schema changes, not application crawls; avoid migrations alongside active application traffic on either backend.

## Cache behavior and concurrent use

A successful organization refresh retires previously open alerts absent from the response. A targeted refresh affects only the selected repository. Retired alerts remain as non-open history, and the CLI and dashboard show the current open cache. A failed alert fetch leaves previously stored alerts available. Alert replacement is transactional: a failed write does not leave the previous set partially retired.

Neither backend retains separate scan snapshots or fetches source code during a findings export. Cache timestamps and sync status describe fetching GitHub data, not scanner coverage. Zero findings does not establish that a repository was scanned.

Schedule crawls so their scopes do not overlap. PostgreSQL permits concurrent clients, but an organization-wide crawl and a repository crawl can still overwrite each other's view of current alerts. A shared database does not provide a job scheduler or distributed crawl lock.

## Moving between backends

The migration command changes the schema within one database; it does not copy data between SQLite and PostgreSQL. Selecting a new target starts a separate cache. The simplest transition is to keep the old database as a backup, select the new target, and run `sync` for the desired organizations. This rebuilds current repository metadata and open findings from GitHub; it does not transfer retired alerts or historical sync records. Keep using SQLite if retaining that history in place is the priority, or plan a separate data transfer if it must move.

## Reproduce the PostgreSQL integration suite

Docker and Go are needed for this workflow. These credentials are for a disposable local server bound to loopback:

```sh
docker run --name ghscanscope-postgres18 \
  -e POSTGRES_USER=ghscanscope -e POSTGRES_PASSWORD=ghscanscope \
  -e POSTGRES_DB=ghscanscope -p 127.0.0.1:55432:5432 -d postgres:18

# Wait for readiness; retry this command until it succeeds.
docker exec ghscanscope-postgres18 pg_isready -U ghscanscope

# Existing SQLite tests and legacy-cache compatibility tests.
go test -race -count=1 ./...

# Real PostgreSQL storage, migrations, crawling, findings, and HTML tests.
GHSCAN_SCOPE_TEST_POSTGRES='postgres://ghscanscope:ghscanscope@127.0.0.1:55432/ghscanscope?sslmode=disable' \
  go test -race -count=1 ./...

# Remove the disposable server and its anonymous data volume when finished.
docker rm -fv ghscanscope-postgres18
```

PostgreSQL tests create and remove isolated schemas in the supplied test database; use a dedicated test database and a role allowed to create schemas. With no test URL, the shared backend tests use temporary SQLite files. Legacy SQLite fixtures always exercise SQLite. The GitHub side uses deterministic HTTP fixtures; PostgreSQL storage uses the actual server. CI runs both SQLite tests and a PostgreSQL 18 service job.

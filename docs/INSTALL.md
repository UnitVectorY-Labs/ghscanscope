---
layout: default
title: Installation
nav_order: 2
permalink: /install
---

# Installation
{: .no_toc }

## Table of Contents
{: .no_toc .text-delta }

- TOC
{:toc}

## Prerequisites

- **Go 1.27 or newer:** required for `go install` or building from source
- **Git:** required for automatic detection of a checked-out repository; explicit organization/repository selection and directory configuration work without it
- **GitHub authentication:** credentials with access to the repositories and their code-scanning alerts

SQLite is included in the application; a separate database server is not required.

## Installation Methods

### Download Binary

Download a pre-built binary for your operating system and architecture from [GitHub Releases](https://github.com/UnitVectorY-Labs/ghscanscope/releases). Extract it and add it to your `PATH`.

### Install Using Go

```bash
go install github.com/UnitVectorY-Labs/ghscanscope@latest
```

Ensure the Go binary directory, typically `$(go env GOPATH)/bin`, is on your `PATH`.

### Build from Source

```bash
git clone https://github.com/UnitVectorY-Labs/ghscanscope.git
cd ghscanscope
go build -o ghscanscope .
./ghscanscope version
```

## Authentication

Credentials are selected in this order:

1. `GITHUB_TOKEN`
2. `GH_TOKEN`
3. The active GitHub CLI credential from `gh auth token --hostname github.com`

To use GitHub CLI authentication:

```bash
gh auth login --hostname github.com
```

Alternatively, set `GITHUB_TOKEN` or `GH_TOKEN` in your shell. The credential must have permission to read the selected repositories and code-scanning alerts. API permission errors are reported by the sync command.

## First Run

```bash
ghscanscope sync --org UnitVectorY-Labs
ghscanscope web
```

Open `http://127.0.0.1:8080` to explore the synced findings. The database directory is created automatically. See [Examples](EXAMPLES.md) for configuring a folder and syncing from a checkout.

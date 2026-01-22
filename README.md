# squlito

Terminal SQLite browser built with Go + gocui.

## Install

### macOS/Linux (script)

```bash
curl -fsSL https://raw.githubusercontent.com/guillemus/squlito/main/install.sh | sh
```

Note: Windows install script not supported yet.

Optional:

```bash
VERSION=v0.1.0 PREFIX=$HOME/.local curl -fsSL https://raw.githubusercontent.com/guillemus/squlito/main/install.sh | sh
```

### Manual

Download the archive from the GitHub Releases page and place `squlito` in your PATH.

## Seed data

```bash
go run ./cmd/seed
```

## Run

```bash
go run ./cmd/squlito data/seed.db
```

## Build

```bash
go build ./...
```

## Test

```bash
go test ./...
```

Notes:
- Query results cap at 10k rows and report truncation.
- Cell display truncates to 50 chars.

# squlito

Terminal SQLite browser built with Go + gocui.

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

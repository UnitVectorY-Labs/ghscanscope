# Repository conventions

- Go CLI entry points live at the root; application workflows live in `internal/app`.
- `internal/github` fetches alerts; `internal/store` owns SQLite/PostgreSQL storage and migrations; `internal/web` owns the dashboard and templates.
- Update the relevant `docs/` pages whenever application behavior, commands, configuration, output, or schema changes; keep README examples consistent.
- Use `encoding/json/v2` (and `encoding/json/jsontext` when needed), with explicit JSON field tags.
- Keep storage changes compatible with both backends and add versioned migrations for schema changes.
- Format Go changes with `gofmt`; run `go build ./...`, `go test ./...`, and `go vet ./...` before submitting.

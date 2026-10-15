package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// LatestVersion is the highest schema version shipped in this binary.
const LatestVersion int64 = 2

//go:embed schema_sqlite.sql
var sqliteSchema string

//go:embed schema_postgres.sql
var postgresSchema string

func (s *Store) migrationProvider() (*goose.Provider, error) {
	dialect := goose.DialectSQLite3
	opts := []goose.ProviderOption{goose.WithDisableGlobalRegistry(true), goose.WithGoMigrations(
		goose.NewGoMigration(1, &goose.GoFunc{RunTx: s.createSchema}, &goose.GoFunc{RunTx: dropSchema}),
		goose.NewGoMigration(2, &goose.GoFunc{RunTx: s.upPriority}, &goose.GoFunc{RunTx: downPriority}),
	)}
	if s.postgres {
		dialect = goose.DialectPostgres
		locker, err := lock.NewPostgresSessionLocker()
		if err != nil {
			return nil, err
		}
		opts = append(opts, goose.WithSessionLocker(locker))
	}
	return goose.NewProvider(dialect, s.DB, nil, opts...)
}

// Migrate uses the same version history for automatic upgrades and explicit
// operations. A nil target means all pending upgrades or one downgrade.
func (s *Store) Migrate(ctx context.Context, direction string, target *int64) (int64, error) {
	if direction != "up" && direction != "down" && direction != "status" {
		return 0, fmt.Errorf("migration action must be up, down, or status")
	}
	if target != nil && (*target < 0 || *target > LatestVersion) {
		return 0, fmt.Errorf("migration version must be between 0 and %d", LatestVersion)
	}
	if direction == "status" && target != nil {
		return 0, fmt.Errorf("status does not accept a target version")
	}
	p, err := s.migrationProvider()
	if err != nil {
		return 0, err
	}
	switch direction {
	case "up":
		if target == nil {
			_, err = p.Up(ctx)
		} else {
			_, err = p.UpTo(ctx, *target)
		}
	case "down":
		if target == nil {
			_, err = p.Down(ctx)
		} else {
			_, err = p.DownTo(ctx, *target)
		}
	}
	if err != nil {
		return 0, err
	}
	return p.GetDBVersion(ctx)
}

func (s *Store) createSchema(ctx context.Context, tx *sql.Tx) error {
	schema := sqliteSchema
	if s.postgres {
		schema = postgresSchema
	}
	if err := execStatements(ctx, tx, schema); err != nil {
		return err
	}
	if s.postgres {
		return nil
	}
	// Adopt databases from before version tracking without discarding cached data.
	for _, column := range legacyColumns {
		if err := ensureAlertColumn(ctx, tx, column.name, "ALTER TABLE alerts ADD COLUMN "+column.name+" "+column.definition); err != nil {
			return err
		}
	}
	return nil
}

type alertColumn struct{ name, definition string }

var legacyColumns = []alertColumn{
	{"tool_version", "TEXT NOT NULL DEFAULT ''"},
	{"tool_guid", "TEXT NOT NULL DEFAULT ''"},
	{"rule_description", "TEXT NOT NULL DEFAULT ''"},
	{"rule_tags", "TEXT NOT NULL DEFAULT '[]'"},
	{"message", "TEXT NOT NULL DEFAULT ''"},
	{"ref", "TEXT NOT NULL DEFAULT ''"},
	{"commit_sha", "TEXT NOT NULL DEFAULT ''"},
	{"analysis_key", "TEXT NOT NULL DEFAULT ''"},
	{"environment", "TEXT NOT NULL DEFAULT ''"},
	{"category", "TEXT NOT NULL DEFAULT ''"},
}

func (s *Store) upPriority(ctx context.Context, tx *sql.Tx) error {
	for _, column := range []alertColumn{
		{"priority", "TEXT NOT NULL DEFAULT 'Unknown'"},
		{"reported_severity", "TEXT NOT NULL DEFAULT ''"},
		{"severity_source", "TEXT NOT NULL DEFAULT ''"},
	} {
		statement := "ALTER TABLE alerts ADD COLUMN " + column.name + " " + column.definition
		if s.postgres {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return err
			}
		} else if err := ensureAlertColumn(ctx, tx, column.name, statement); err != nil {
			return err
		}
	}
	return execStatements(ctx, tx, `
 UPDATE alerts SET reported_severity=severity WHERE reported_severity='';
 UPDATE alerts SET severity_source='legacy alerts.severity' WHERE severity_source='';
 UPDATE alerts SET priority=CASE lower(trim(reported_severity)) WHEN 'critical' THEN 'Critical' WHEN 'high' THEN 'High' WHEN 'error' THEN 'High' WHEN 'medium' THEN 'Medium' WHEN 'warning' THEN 'Medium' WHEN 'low' THEN 'Low' WHEN 'note' THEN 'Low' WHEN 'recommendation' THEN 'Low' ELSE 'Unknown' END;
 UPDATE alerts SET severity=priority;
 `)
}

func downPriority(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx, `
 UPDATE alerts SET severity=reported_severity;
 ALTER TABLE alerts DROP COLUMN priority;
 ALTER TABLE alerts DROP COLUMN reported_severity;
 ALTER TABLE alerts DROP COLUMN severity_source;
 `)
}
func dropSchema(ctx context.Context, tx *sql.Tx) error {
	return execStatements(ctx, tx, `DROP TABLE sync_runs; DROP TABLE alerts; DROP TABLE repositories; DROP TABLE organizations;`)
}

// Each schema statement is simple SQL without procedural blocks or semicolons
// in literals. Executing separately also supports pgx's extended query mode.
func execStatements(ctx context.Context, tx *sql.Tx, statements string) error {
	for _, statement := range strings.Split(statements, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
func ensureAlertColumn(ctx context.Context, tx *sql.Tx, column, alterStatement string) error {
	rows, err := tx.QueryContext(ctx, "PRAGMA table_info(alerts)")
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return err
		}
		if name == column {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if found {
		return nil
	}
	_, err = tx.ExecContext(ctx, alterStatement)
	return err
}

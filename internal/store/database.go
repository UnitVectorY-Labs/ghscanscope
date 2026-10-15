package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// IsPostgres recognizes both PostgreSQL URL schemes. All other targets retain
// their existing SQLite filename/URI semantics.
func IsPostgres(target string) bool {
	return strings.HasPrefix(target, "postgres://") || strings.HasPrefix(target, "postgresql://")
}

// Open preserves automatic schema upgrades for existing callers.
func Open(target string) (*Store, error) { return OpenContext(context.Background(), target) }

func OpenContext(ctx context.Context, target string) (*Store, error) {
	s, err := connect(ctx, target)
	if err != nil {
		return nil, err
	}
	if _, err = s.Migrate(ctx, "up", nil); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

// Connect opens a database without applying migrations, for migration commands.
func Connect(ctx context.Context, target string) (*Store, error) { return connect(ctx, target) }

func connect(ctx context.Context, target string) (*Store, error) {
	postgres := IsPostgres(target)
	driver := "sqlite"
	if postgres {
		driver = "pgx"
	}
	db, err := sql.Open(driver, target)
	if err != nil {
		return nil, err
	}
	if postgres {
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(2)
		db.SetConnMaxLifetime(30 * time.Minute)
	} else {
		db.SetMaxOpenConns(1)
		db.SetConnMaxLifetime(0)
	}
	s := &Store{DB: db, postgres: postgres}
	if err = db.PingContext(ctx); err == nil && !postgres {
		_, err = db.ExecContext(ctx, "PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;")
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

// sql binds placeholders in static SQL; statements use no question marks in
// literals. User values are passed separately through database/sql.
func (s *Store) sql(query string) string {
	if !s.postgres {
		return query
	}
	var out strings.Builder
	n := 0
	for _, r := range query {
		if r == '?' {
			n++
			fmt.Fprintf(&out, "$%d", n)
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// These helpers accept only column names supplied by store code. SQLite keeps
// its NOCASE indexes; PostgreSQL uses matching expression indexes, with no
// extension or elevated database privileges required.
func (s *Store) conflictKey(column string) string {
	if s.postgres {
		return "lower(" + column + ")"
	}
	return column
}
func (s *Store) nameOrder(column string) string {
	if s.postgres {
		return "lower(" + column + ")"
	}
	return column + " COLLATE NOCASE"
}
func (s *Store) nameEqual(column string) string {
	if s.postgres {
		return "lower(" + column + ")=lower(?)"
	}
	return column + "=? COLLATE NOCASE"
}
func (s *Store) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.DB.ExecContext(ctx, s.sql(query), args...)
}
func (s *Store) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.DB.QueryRowContext(ctx, s.sql(query), args...)
}
func (s *Store) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.DB.QueryContext(ctx, s.sql(query), args...)
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

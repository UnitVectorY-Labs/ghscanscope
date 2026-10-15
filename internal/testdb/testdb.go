// Package testdb gives backend contract tests isolated databases. PostgreSQL
// tests run against a real server when GHSCAN_SCOPE_TEST_POSTGRES is supplied.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func Target(t *testing.T) string {
	t.Helper()
	target := os.Getenv("GHSCAN_SCOPE_TEST_POSTGRES")
	if target == "" {
		return t.TempDir() + "/test.db"
	}
	u, err := url.Parse(target)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := sql.Open("pgx", target)
	if err != nil {
		t.Fatal(err)
	}
	schema := "test_" + fmt.Sprintf("%x", uuid.New())
	if _, err := admin.ExecContext(context.Background(), "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		defer admin.Close()
		if _, err := admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}

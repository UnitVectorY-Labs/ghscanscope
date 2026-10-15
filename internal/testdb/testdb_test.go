package testdb

import (
	"database/sql"
	"net/url"
	"os"
	"testing"
)

func TestPostgresSchemaNameIsNotTruncated(t *testing.T) {
	if os.Getenv("GHSCAN_SCOPE_TEST_POSTGRES") == "" {
		t.Skip("set GHSCAN_SCOPE_TEST_POSTGRES to test against PostgreSQL")
	}
	target := Target(t)
	u, err := url.Parse(target)
	if err != nil {
		t.Fatal(err)
	}
	want := u.Query().Get("search_path")
	if len(want) > 63 {
		t.Fatalf("schema name exceeds PostgreSQL's 63-byte identifier limit: %d bytes", len(want))
	}
	db, err := sql.Open("pgx", target)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var got string
	if err := db.QueryRow("SELECT current_schema()").Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("PostgreSQL selected schema %q, want exact name %q", got, want)
	}
}

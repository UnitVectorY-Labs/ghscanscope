package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/testdb"
)

func TestMigrationCommand(t *testing.T) {
	target := testdb.Target(t)
	t.Setenv("GHSCAN_SCOPE_DB", target)
	for _, step := range []struct {
		args    []string
		version string
	}{
		{[]string{"status"}, "0"},
		{[]string{"up", "1"}, "1"},
		{[]string{"up"}, "2"},
		{[]string{"down"}, "1"},
		{[]string{"down", "0"}, "0"},
		{[]string{"--db", target, "up"}, "2"},
	} {
		var out bytes.Buffer
		if err := runMigrate(context.Background(), step.args, &out); err != nil || !strings.Contains(out.String(), "Database version: "+step.version) {
			t.Fatalf("%v: out=%q err=%v", step.args, out.String(), err)
		}
	}
	for _, args := range [][]string{nil, {"invalid"}, {"status", "1"}, {"up", "-1"}, {"up", "3"}, {"up", "no"}, {"up", "1", "extra"}} {
		if err := runMigrate(context.Background(), args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestPostgresSelection(t *testing.T) {
	target := "postgresql://example.invalid/cache?sslmode=verify-full"
	t.Setenv("GHSCAN_SCOPE_DB", target)
	_, db, err := syncFlags(nil)
	if err != nil || db != target {
		t.Fatalf("environment: %q %v", db, err)
	}
	_, db, err = syncFlags([]string{"--db", "sqlite.db"})
	if err != nil || db != "sqlite.db" {
		t.Fatalf("flag override: %q %v", db, err)
	}
}

func TestPostgresFlagSelection(t *testing.T) {
	t.Setenv("GHSCAN_SCOPE_DB", "sqlite.db")
	target := "postgres://localhost/cache?sslmode=disable"
	_, db, err := syncFlags([]string{"--db", target})
	if err != nil || db != target {
		t.Fatalf("PostgreSQL flag: %q %v", db, err)
	}
}

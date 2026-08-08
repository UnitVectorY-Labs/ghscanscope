package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestOpenBackfillsLegacySeverityWithoutResync(t *testing.T) {
	path := t.TempDir() + "/legacy.db"
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE organizations (id INTEGER PRIMARY KEY, github_id INTEGER, login TEXT NOT NULL UNIQUE, url TEXT NOT NULL DEFAULT '', last_sync_at TEXT, last_sync_status TEXT NOT NULL DEFAULT '', last_sync_error TEXT NOT NULL DEFAULT '');
CREATE TABLE repositories (id INTEGER PRIMARY KEY, org_id INTEGER NOT NULL, github_id INTEGER NOT NULL, owner TEXT NOT NULL, name TEXT NOT NULL, full_name TEXT NOT NULL UNIQUE, url TEXT NOT NULL DEFAULT '', description TEXT NOT NULL DEFAULT '', visibility TEXT NOT NULL DEFAULT '', archived INTEGER NOT NULL DEFAULT 0, language TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL, UNIQUE(org_id, github_id));
CREATE TABLE alerts (id INTEGER PRIMARY KEY, org_id INTEGER NOT NULL, repository_id INTEGER NOT NULL, github_id INTEGER NOT NULL, tool TEXT NOT NULL, rule_id TEXT NOT NULL, rule_name TEXT NOT NULL DEFAULT '', rule_tags TEXT NOT NULL DEFAULT '[]', severity TEXT NOT NULL DEFAULT '', path TEXT NOT NULL DEFAULT '', start_line INTEGER NOT NULL DEFAULT 0, end_line INTEGER NOT NULL DEFAULT 0, start_column INTEGER NOT NULL DEFAULT 0, end_column INTEGER NOT NULL DEFAULT 0, url TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL, is_open INTEGER NOT NULL DEFAULT 1, UNIQUE(repository_id, github_id));
CREATE TABLE sync_runs (id INTEGER PRIMARY KEY, org_id INTEGER NOT NULL, repository_id INTEGER, started_at TEXT NOT NULL, finished_at TEXT, status TEXT NOT NULL, error TEXT NOT NULL DEFAULT '', repository_count INTEGER NOT NULL DEFAULT 0, alert_count INTEGER NOT NULL DEFAULT 0);`)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = db.Exec(`INSERT INTO organizations VALUES(1,1,'acme','',NULL,'',''); INSERT INTO repositories VALUES(1,1,1,'acme','app','acme/app','','','public',0,'',?); INSERT INTO alerts(id,org_id,repository_id,github_id,tool,rule_id,severity,created_at,updated_at) VALUES(1,1,1,1,'scanner','rule','Warning',?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	alerts, err := s.OpenAlerts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Priority != "Medium" || alerts[0].Severity != "Medium" || alerts[0].ReportedSeverity != "Warning" || alerts[0].SeveritySource != "legacy alerts.severity" {
		t.Fatalf("legacy alert was not faithfully backfilled: %+v", alerts)
	}
}

func TestCanonicalPriority(t *testing.T) {
	for raw, want := range map[string]string{"critical": "Critical", "error": "High", "warning": "Medium", "note": "Low", "informational": "Unknown", "": "Unknown"} {
		if got := CanonicalPriority(raw); got != want {
			t.Errorf("CanonicalPriority(%q) = %q, want %q", raw, got, want)
		}
	}
}

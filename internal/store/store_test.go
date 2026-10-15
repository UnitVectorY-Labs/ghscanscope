package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/testdb"
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

func TestAlertProgressDoesNotChangeTransactionalReplacement(t *testing.T) {
	s, err := Open(testdb.Target(t))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	org, err := s.EnsureOrg(ctx, "acme", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	repo, err := s.UpsertRepository(ctx, Repository{OrgID: org, GitHubID: 1, Owner: "acme", Name: "one", FullName: "acme/one"})
	if err != nil {
		t.Fatal(err)
	}
	original := Alert{OrgID: org, RepositoryID: repo, GitHubID: 1, Severity: "high", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	var events [][2]int
	progress := func(done, total int) { events = append(events, [2]int{done, total}) }
	if err := s.ReplaceOpenAlertsWithProgress(ctx, org, &repo, []Alert{original}, progress); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0] != [2]int{0, 1} || events[1] != [2]int{1, 1} {
		t.Fatalf("progress: %v", events)
	}
	events = nil
	invalid := original
	invalid.GitHubID = 2
	invalid.RepositoryID = 999
	if err := s.ReplaceOpenAlertsWithProgress(ctx, org, &repo, []Alert{invalid}, progress); err == nil {
		t.Fatal("expected foreign key failure")
	}
	if len(events) != 1 || events[0] != [2]int{0, 1} {
		t.Fatalf("reported uncompleted rows: %v", events)
	}
	alerts, err := s.OpenAlerts(ctx)
	if err != nil || len(alerts) != 1 || alerts[0].GitHubID != 1 {
		t.Fatalf("failed replacement changed cache: %v %v", alerts, err)
	}
}

func TestMigrationsAndBackendContract(t *testing.T) {
	ctx := context.Background()
	target := testdb.Target(t)
	s, err := Connect(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	check := func(action string, target *int64, want int64) {
		t.Helper()
		version, err := s.Migrate(ctx, action, target)
		if err != nil || version != want {
			t.Fatalf("%s: version=%d err=%v, want %d", action, version, err, want)
		}
	}
	check("status", nil, 0)
	check("up", nil, 2)
	check("up", nil, 2)
	org, err := s.EnsureOrg(ctx, "Acme", "https://github.com/Acme", 42)
	if err != nil {
		t.Fatal(err)
	}
	sameOrg, err := s.EnsureOrg(ctx, "acme", "https://github.com/acme", 0)
	if err != nil || sameOrg != org {
		t.Fatalf("case insensitive org: %d %v", sameOrg, err)
	}
	repo, err := s.UpsertRepository(ctx, Repository{OrgID: org, GitHubID: 10, Owner: "Acme", Name: "App", FullName: "Acme/App", Archived: true})
	if err != nil {
		t.Fatal(err)
	}
	sameRepo, err := s.UpsertRepository(ctx, Repository{OrgID: org, GitHubID: 10, Owner: "acme", Name: "app", FullName: "acme/app", Archived: true})
	if err != nil || repo != sameRepo {
		t.Fatalf("case insensitive repo: %d %v", sameRepo, err)
	}
	run, err := s.BeginRun(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FinishRun(ctx, run, org, "success", "", 1, 1); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	a := Alert{OrgID: org, RepositoryID: repo, GitHubID: 1, Tool: "scanner", RuleID: "rule", ReportedSeverity: "Warning", SeveritySource: "scanner", CreatedAt: now, UpdatedAt: now, Tags: []string{"security", "quote' and ?"}, Message: "quote' and ?"}
	if err := s.ReplaceOpenAlerts(ctx, org, &repo, []Alert{a}); err != nil {
		t.Fatal(err)
	}
	alerts, err := s.RepositoryAlerts(ctx, "ACME/APP")
	if err != nil || len(alerts) != 1 || alerts[0].Priority != "Medium" || alerts[0].Message != a.Message || !alerts[0].CreatedAt.Equal(now) {
		t.Fatalf("alerts=%+v err=%v", alerts, err)
	}
	repos, err := s.Repositories(ctx)
	if err != nil || len(repos) != 1 || !repos[0].Archived {
		t.Fatalf("repos=%+v err=%v", repos, err)
	}
	orgs, err := s.Organizations(ctx)
	if err != nil || len(orgs) != 1 || orgs[0].LastSync == nil || orgs[0].LastStatus != "success" {
		t.Fatalf("orgs=%+v err=%v", orgs, err)
	}
	// A dependent view makes the downgrade fail after its data update. Both
	// backends must roll back that update and retain migration version 2.
	if _, err := s.DB.ExecContext(ctx, "CREATE VIEW priority_dependency AS SELECT priority FROM alerts"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Migrate(ctx, "down", nil); err == nil {
		t.Fatal("expected dependent view to prevent downgrade")
	}
	check("status", nil, 2)
	var unchanged string
	if err := s.DB.QueryRowContext(ctx, "SELECT severity FROM alerts").Scan(&unchanged); err != nil || unchanged != "Medium" {
		t.Fatalf("failed migration changed severity=%q err=%v", unchanged, err)
	}
	if _, err := s.DB.ExecContext(ctx, "DROP VIEW priority_dependency"); err != nil {
		t.Fatal(err)
	}
	// Downgrade with live data, verify the old representation, then upgrade again.
	check("down", nil, 1)
	var severity string
	if err := s.DB.QueryRowContext(ctx, "SELECT severity FROM alerts").Scan(&severity); err != nil || severity != "Warning" {
		t.Fatalf("downgrade severity=%q err=%v", severity, err)
	}
	check("up", nil, 2)
	alerts, err = s.OpenAlerts(ctx)
	if err != nil || len(alerts) != 1 || alerts[0].ReportedSeverity != "Warning" || alerts[0].Priority != "Medium" {
		t.Fatalf("upgrade alerts=%+v err=%v", alerts, err)
	}
	// A normal reopen must not run backfills over application data again.
	if _, err := s.DB.ExecContext(ctx, "UPDATE alerts SET priority='Critical', severity='Critical'"); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	alerts, err = s.OpenAlerts(ctx)
	if err != nil || len(alerts) != 1 || alerts[0].Priority != "Critical" {
		t.Fatalf("reopen alerts=%+v err=%v", alerts, err)
	}
	zero := int64(0)
	check("down", &zero, 0)
	if _, err := s.DB.QueryContext(ctx, "SELECT * FROM alerts"); err == nil {
		t.Fatal("down to zero retained alerts table")
	}
	check("up", nil, 2)
	alerts, err = s.OpenAlerts(ctx)
	if err != nil || len(alerts) != 0 {
		t.Fatalf("fresh upgrade alerts=%+v err=%v", alerts, err)
	}
}

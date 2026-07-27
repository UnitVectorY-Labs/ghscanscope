package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct{ DB *sql.DB }

type Organization struct {
	ID         int64
	Login      string
	URL        string
	LastSync   *time.Time
	LastStatus string
}

type Repository struct {
	ID                                                            int64
	OrgID                                                         int64
	GitHubID                                                      int64
	Owner, Name, FullName, URL, Description, Visibility, Language string
	Archived                                                      bool
	UpdatedAt                                                     time.Time
}

type Alert struct {
	ID, OrgID, RepositoryID, GitHubID                 int64
	Org, Repository, Tool, RuleID, RuleName, Severity string
	Path                                              string
	StartLine, EndLine, StartColumn, EndColumn        int
	URL                                               string
	CreatedAt, UpdatedAt                              time.Time
	Open                                              bool
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	s := &Store{DB: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) migrate(ctx context.Context) error {
	const schema = `
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
CREATE TABLE IF NOT EXISTS organizations (
 id INTEGER PRIMARY KEY, github_id INTEGER, login TEXT NOT NULL UNIQUE COLLATE NOCASE, url TEXT NOT NULL DEFAULT '',
 last_sync_at TEXT, last_sync_status TEXT NOT NULL DEFAULT '', last_sync_error TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS repositories (
 id INTEGER PRIMARY KEY, org_id INTEGER NOT NULL REFERENCES organizations(id), github_id INTEGER NOT NULL,
 owner TEXT NOT NULL, name TEXT NOT NULL, full_name TEXT NOT NULL UNIQUE COLLATE NOCASE, url TEXT NOT NULL DEFAULT '',
 description TEXT NOT NULL DEFAULT '', visibility TEXT NOT NULL DEFAULT '', archived INTEGER NOT NULL DEFAULT 0,
 language TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL, UNIQUE(org_id, github_id)
);
CREATE TABLE IF NOT EXISTS alerts (
 id INTEGER PRIMARY KEY, org_id INTEGER NOT NULL REFERENCES organizations(id), repository_id INTEGER NOT NULL REFERENCES repositories(id),
 github_id INTEGER NOT NULL, tool TEXT NOT NULL, rule_id TEXT NOT NULL, rule_name TEXT NOT NULL DEFAULT '', severity TEXT NOT NULL DEFAULT '',
 path TEXT NOT NULL DEFAULT '', start_line INTEGER NOT NULL DEFAULT 0, end_line INTEGER NOT NULL DEFAULT 0,
 start_column INTEGER NOT NULL DEFAULT 0, end_column INTEGER NOT NULL DEFAULT 0, url TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL, is_open INTEGER NOT NULL DEFAULT 1,
 UNIQUE(repository_id, github_id)
);
CREATE TABLE IF NOT EXISTS sync_runs (
 id INTEGER PRIMARY KEY, org_id INTEGER NOT NULL REFERENCES organizations(id), repository_id INTEGER REFERENCES repositories(id),
 started_at TEXT NOT NULL, finished_at TEXT, status TEXT NOT NULL, error TEXT NOT NULL DEFAULT '',
 repository_count INTEGER NOT NULL DEFAULT 0, alert_count INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_alerts_open ON alerts(is_open);
CREATE INDEX IF NOT EXISTS idx_alerts_group ON alerts(tool, rule_id);
CREATE INDEX IF NOT EXISTS idx_repositories_org ON repositories(org_id);
`
	_, err := s.DB.ExecContext(ctx, schema)
	return err
}

func (s *Store) EnsureOrg(ctx context.Context, login, url string, githubID int64) (int64, error) {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO organizations(login,url,github_id) VALUES(?,?,?)
 ON CONFLICT(login) DO UPDATE SET url=excluded.url, github_id=CASE WHEN excluded.github_id=0 THEN organizations.github_id ELSE excluded.github_id END`, login, url, githubID)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.DB.QueryRowContext(ctx, `SELECT id FROM organizations WHERE login=?`, login).Scan(&id)
	return id, err
}

func (s *Store) BeginRun(ctx context.Context, orgID int64) (int64, error) {
	res, err := s.DB.ExecContext(ctx, `INSERT INTO sync_runs(org_id,started_at,status) VALUES(?,?,?)`, orgID, time.Now().UTC().Format(time.RFC3339Nano), "running")
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) FinishRun(ctx context.Context, runID, orgID int64, status, message string, repoCount, alertCount int) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.DB.ExecContext(ctx, `UPDATE sync_runs SET finished_at=?,status=?,error=?,repository_count=?,alert_count=? WHERE id=?`, now, status, message, repoCount, alertCount, runID)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE organizations SET last_sync_at=?,last_sync_status=?,last_sync_error=? WHERE id=?`, now, status, message, orgID)
	return err
}

func (s *Store) UpsertRepository(ctx context.Context, r Repository) (int64, error) {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO repositories(org_id,github_id,owner,name,full_name,url,description,visibility,archived,language,updated_at)
 VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(full_name) DO UPDATE SET org_id=excluded.org_id,github_id=excluded.github_id,owner=excluded.owner,name=excluded.name,url=excluded.url,description=excluded.description,visibility=excluded.visibility,archived=excluded.archived,language=excluded.language,updated_at=excluded.updated_at`,
		r.OrgID, r.GitHubID, r.Owner, r.Name, r.FullName, r.URL, r.Description, r.Visibility, r.Archived, r.Language, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.DB.QueryRowContext(ctx, `SELECT id FROM repositories WHERE full_name=?`, r.FullName).Scan(&id)
	return id, err
}

func (s *Store) ReplaceOpenAlerts(ctx context.Context, orgID int64, repoID *int64, alerts []Alert) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if repoID == nil {
		_, err = tx.ExecContext(ctx, `UPDATE alerts SET is_open=0 WHERE org_id=?`, orgID)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE alerts SET is_open=0 WHERE repository_id=?`, *repoID)
	}
	if err != nil {
		return err
	}
	for _, a := range alerts {
		_, err = tx.ExecContext(ctx, `INSERT INTO alerts(org_id,repository_id,github_id,tool,rule_id,rule_name,severity,path,start_line,end_line,start_column,end_column,url,created_at,updated_at,is_open)
 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1) ON CONFLICT(repository_id,github_id) DO UPDATE SET tool=excluded.tool,rule_id=excluded.rule_id,rule_name=excluded.rule_name,severity=excluded.severity,path=excluded.path,start_line=excluded.start_line,end_line=excluded.end_line,start_column=excluded.start_column,end_column=excluded.end_column,url=excluded.url,created_at=excluded.created_at,updated_at=excluded.updated_at,is_open=1`,
			a.OrgID, a.RepositoryID, a.GitHubID, a.Tool, a.RuleID, a.RuleName, a.Severity, a.Path, a.StartLine, a.EndLine, a.StartColumn, a.EndColumn, a.URL, a.CreatedAt.UTC().Format(time.RFC3339Nano), a.UpdatedAt.UTC().Format(time.RFC3339Nano))
		if err != nil {
			return fmt.Errorf("store alert %d: %w", a.GitHubID, err)
		}
	}
	return tx.Commit()
}

func scanTime(value sql.NullString) *time.Time {
	if !value.Valid {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil
	}
	return &t
}

func (s *Store) Organizations(ctx context.Context) ([]Organization, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,login,url,last_sync_at,last_sync_status FROM organizations ORDER BY login COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Organization
	for rows.Next() {
		var o Organization
		var last sql.NullString
		if err := rows.Scan(&o.ID, &o.Login, &o.URL, &last, &o.LastStatus); err != nil {
			return nil, err
		}
		o.LastSync = scanTime(last)
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) Repositories(ctx context.Context) ([]Repository, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,org_id,github_id,owner,name,full_name,url,description,visibility,archived,language,updated_at FROM repositories ORDER BY full_name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Repository
	for rows.Next() {
		var r Repository
		var updated string
		if err := rows.Scan(&r.ID, &r.OrgID, &r.GitHubID, &r.Owner, &r.Name, &r.FullName, &r.URL, &r.Description, &r.Visibility, &r.Archived, &r.Language, &updated); err != nil {
			return nil, err
		}
		r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) OpenAlerts(ctx context.Context) ([]Alert, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT a.id,a.org_id,a.repository_id,a.github_id,o.login,r.full_name,a.tool,a.rule_id,a.rule_name,a.severity,a.path,a.start_line,a.end_line,a.start_column,a.end_column,a.url,a.created_at,a.updated_at,a.is_open FROM alerts a JOIN organizations o ON o.id=a.org_id JOIN repositories r ON r.id=a.repository_id WHERE a.is_open=1 ORDER BY a.tool,a.rule_id,r.full_name,a.github_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Alert
	for rows.Next() {
		var a Alert
		var created, updated string
		if err := rows.Scan(&a.ID, &a.OrgID, &a.RepositoryID, &a.GitHubID, &a.Org, &a.Repository, &a.Tool, &a.RuleID, &a.RuleName, &a.Severity, &a.Path, &a.StartLine, &a.EndLine, &a.StartColumn, &a.EndColumn, &a.URL, &created, &updated, &a.Open); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		a.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, a)
	}
	return out, rows.Err()
}

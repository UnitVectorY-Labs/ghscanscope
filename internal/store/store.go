package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Store struct {
	DB       *sql.DB
	postgres bool
}

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
	// Priority is the app's canonical priority. Severity is retained for
	// backwards-compatible callers and contains the scanner-reported value.
	Priority, ReportedSeverity, SeveritySource string
	RepositoryURL                              string
	RuleDescription, ToolVersion, ToolGUID     string
	Message, Ref, CommitSHA, AnalysisKey       string
	Environment, Category                      string
	Tags                                       []string
	Path                                       string
	StartLine, EndLine, StartColumn, EndColumn int
	URL                                        string
	CreatedAt, UpdatedAt                       time.Time
	Open                                       bool
}

// CanonicalPriority maps scanner-specific labels to the five labels shown by
// the application. Values that are not explicitly understood remain Unknown.
func CanonicalPriority(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return "Critical"
	case "high", "error":
		return "High"
	case "medium", "warning":
		return "Medium"
	case "low", "note", "recommendation":
		return "Low"
	default:
		return "Unknown"
	}
}

func (s *Store) EnsureOrg(ctx context.Context, login, url string, githubID int64) (int64, error) {
	var id int64
	err := s.queryRow(ctx, `INSERT INTO organizations(login,url,github_id) VALUES(?,?,?)
 ON CONFLICT(`+s.conflictKey("login")+`) DO UPDATE SET url=excluded.url, github_id=CASE WHEN excluded.github_id=0 THEN organizations.github_id ELSE excluded.github_id END RETURNING id`, login, url, githubID).Scan(&id)
	return id, err
}

func (s *Store) BeginRun(ctx context.Context, orgID int64) (int64, error) {
	var id int64
	err := s.queryRow(ctx, `INSERT INTO sync_runs(org_id,started_at,status) VALUES(?,?,?) RETURNING id`, orgID, time.Now().UTC().Format(time.RFC3339Nano), "running").Scan(&id)
	return id, err
}

func (s *Store) FinishRun(ctx context.Context, runID, orgID int64, status, message string, repoCount, alertCount int) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.exec(ctx, `UPDATE sync_runs SET finished_at=?,status=?,error=?,repository_count=?,alert_count=? WHERE id=?`, now, status, message, repoCount, alertCount, runID)
	if err != nil {
		return err
	}
	_, err = s.exec(ctx, `UPDATE organizations SET last_sync_at=?,last_sync_status=?,last_sync_error=? WHERE id=?`, now, status, message, orgID)
	return err
}

func (s *Store) UpsertRepository(ctx context.Context, r Repository) (int64, error) {
	var id int64
	err := s.queryRow(ctx, `INSERT INTO repositories(org_id,github_id,owner,name,full_name,url,description,visibility,archived,language,updated_at)
 VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(`+s.conflictKey("full_name")+`) DO UPDATE SET org_id=excluded.org_id,github_id=excluded.github_id,owner=excluded.owner,name=excluded.name,url=excluded.url,description=excluded.description,visibility=excluded.visibility,archived=excluded.archived,language=excluded.language,updated_at=excluded.updated_at RETURNING id`,
		r.OrgID, r.GitHubID, r.Owner, r.Name, r.FullName, r.URL, r.Description, r.Visibility, boolInt(r.Archived), r.Language, time.Now().UTC().Format(time.RFC3339Nano)).Scan(&id)
	return id, err
}

func (s *Store) ReplaceOpenAlerts(ctx context.Context, orgID int64, repoID *int64, alerts []Alert) error {
	return s.ReplaceOpenAlertsWithProgress(ctx, orgID, repoID, alerts, nil)
}

// ReplaceOpenAlertsWithProgress reports rows written inside the transaction.
// Completion of the callback does not imply the transaction has committed.
func (s *Store) ReplaceOpenAlertsWithProgress(ctx context.Context, orgID int64, repoID *int64, alerts []Alert, progress func(done, total int)) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if repoID == nil {
		_, err = tx.ExecContext(ctx, s.sql(`UPDATE alerts SET is_open=0 WHERE org_id=?`), orgID)
	} else {
		_, err = tx.ExecContext(ctx, s.sql(`UPDATE alerts SET is_open=0 WHERE repository_id=?`), *repoID)
	}
	if err != nil {
		return err
	}
	if progress != nil {
		progress(0, len(alerts))
	}
	for i, a := range alerts {
		reported := a.ReportedSeverity
		if reported == "" {
			reported = a.Severity
		}
		priority := CanonicalPriority(a.Priority)
		if a.Priority == "" {
			priority = CanonicalPriority(reported)
		}
		source := a.SeveritySource
		if source == "" {
			source = "legacy alerts.severity"
		}
		tags, marshalErr := json.Marshal(a.Tags)
		if marshalErr != nil {
			return fmt.Errorf("store alert %d tags: %w", a.GitHubID, marshalErr)
		}
		_, err = tx.ExecContext(ctx, s.sql(`INSERT INTO alerts(org_id,repository_id,github_id,tool,tool_version,tool_guid,rule_id,rule_name,rule_description,rule_tags,severity,priority,reported_severity,severity_source,message,ref,commit_sha,analysis_key,environment,category,path,start_line,end_line,start_column,end_column,url,created_at,updated_at,is_open)
	VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1) ON CONFLICT(repository_id,github_id) DO UPDATE SET tool=excluded.tool,tool_version=excluded.tool_version,tool_guid=excluded.tool_guid,rule_id=excluded.rule_id,rule_name=excluded.rule_name,rule_description=excluded.rule_description,rule_tags=excluded.rule_tags,severity=excluded.severity,priority=excluded.priority,reported_severity=excluded.reported_severity,severity_source=excluded.severity_source,message=excluded.message,ref=excluded.ref,commit_sha=excluded.commit_sha,analysis_key=excluded.analysis_key,environment=excluded.environment,category=excluded.category,path=excluded.path,start_line=excluded.start_line,end_line=excluded.end_line,start_column=excluded.start_column,end_column=excluded.end_column,url=excluded.url,created_at=excluded.created_at,updated_at=excluded.updated_at,is_open=1`),
			a.OrgID, a.RepositoryID, a.GitHubID, a.Tool, a.ToolVersion, a.ToolGUID, a.RuleID, a.RuleName, a.RuleDescription, string(tags), priority, priority, reported, source, a.Message, a.Ref, a.CommitSHA, a.AnalysisKey, a.Environment, a.Category, a.Path, a.StartLine, a.EndLine, a.StartColumn, a.EndColumn, a.URL, a.CreatedAt.UTC().Format(time.RFC3339Nano), a.UpdatedAt.UTC().Format(time.RFC3339Nano))
		if err != nil {
			return fmt.Errorf("store alert %d: %w", a.GitHubID, err)
		}
		if progress != nil {
			progress(i+1, len(alerts))
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
	rows, err := s.query(ctx, `SELECT id,login,url,last_sync_at,last_sync_status FROM organizations ORDER BY `+s.nameOrder("login"))
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
	rows, err := s.query(ctx, `SELECT id,org_id,github_id,owner,name,full_name,url,description,visibility,archived,language,updated_at FROM repositories ORDER BY `+s.nameOrder("full_name"))
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
	return s.openAlerts(ctx, "", nil)
}

// RepositoryAlerts returns only the current cached open alerts for one repository.
func (s *Store) RepositoryAlerts(ctx context.Context, fullName string) ([]Alert, error) {
	return s.openAlerts(ctx, " AND "+s.nameEqual("r.full_name"), []any{fullName})
}

func (s *Store) openAlerts(ctx context.Context, where string, args []any) ([]Alert, error) {
	rows, err := s.query(ctx, `SELECT a.id,a.org_id,a.repository_id,a.github_id,o.login,r.full_name,r.url,a.tool,a.tool_version,a.tool_guid,a.rule_id,a.rule_name,a.rule_description,a.rule_tags,a.severity,a.priority,a.reported_severity,a.severity_source,a.message,a.ref,a.commit_sha,a.analysis_key,a.environment,a.category,a.path,a.start_line,a.end_line,a.start_column,a.end_column,a.url,a.created_at,a.updated_at,a.is_open FROM alerts a JOIN organizations o ON o.id=a.org_id JOIN repositories r ON r.id=a.repository_id WHERE a.is_open=1`+where+` ORDER BY a.updated_at DESC,a.github_id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Alert
	for rows.Next() {
		var a Alert
		var created, updated, tags string
		if err := rows.Scan(&a.ID, &a.OrgID, &a.RepositoryID, &a.GitHubID, &a.Org, &a.Repository, &a.RepositoryURL, &a.Tool, &a.ToolVersion, &a.ToolGUID, &a.RuleID, &a.RuleName, &a.RuleDescription, &tags, &a.Severity, &a.Priority, &a.ReportedSeverity, &a.SeveritySource, &a.Message, &a.Ref, &a.CommitSHA, &a.AnalysisKey, &a.Environment, &a.Category, &a.Path, &a.StartLine, &a.EndLine, &a.StartColumn, &a.EndColumn, &a.URL, &created, &updated, &a.Open); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tags), &a.Tags)
		a.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		a.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, a)
	}
	return out, rows.Err()
}

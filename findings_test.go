package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/app"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
)

func findingsCache(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "cache.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	org, err := s.EnsureOrg(ctx, "Owner", "https://github.com/Owner", 1)
	if err != nil {
		t.Fatal(err)
	}
	repo, err := s.UpsertRepository(ctx, store.Repository{OrgID: org, GitHubID: 1, Owner: "Owner", Name: "repo", FullName: "Owner/repo", URL: "https://github.com/Owner/repo"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.UpsertRepository(ctx, store.Repository{OrgID: org, GitHubID: 2, Owner: "Owner", Name: "other", FullName: "Owner/other"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpsertRepository(ctx, store.Repository{OrgID: org, GitHubID: 3, Owner: "Owner", Name: "empty", FullName: "Owner/empty"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	makeAlert := func(id, repoID int64, severity, tool string) store.Alert {
		return store.Alert{OrgID: org, RepositoryID: repoID, GitHubID: id, Severity: severity, Tool: tool, RuleID: "rule/test", RuleName: "Unsafe input", RuleDescription: "Validate input\nbefore use", Message: "Untrusted value", Tags: []string{"security", "cwe-20"}, Path: "src/file.go", StartLine: 12, EndLine: 14, StartColumn: 2, EndColumn: 8, Ref: "refs/heads/main", CommitSHA: "abc123", AnalysisKey: "analysis", Environment: "env", Category: "category", ToolVersion: "1.2", ToolGUID: "guid", SeveritySource: "rule.security_severity_level", URL: "https://github.com/Owner/repo/security/code-scanning/1", CreatedAt: now, UpdatedAt: now}
	}
	stale := makeAlert(99, repo, "critical", "CodeQL")
	if err := s.ReplaceOpenAlerts(ctx, org, &repo, []store.Alert{stale}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceOpenAlerts(ctx, org, &repo, []store.Alert{makeAlert(1, repo, "critical", "CodeQL"), makeAlert(2, repo, "error", "Semgrep OSS"), makeAlert(3, repo, "warning", "CodeQL"), makeAlert(4, repo, "note", "CodeQL"), makeAlert(5, repo, "none", "Other")}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceOpenAlerts(ctx, org, &other, []store.Alert{makeAlert(6, other, "critical", "CodeQL")}); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFindingsCommand(t *testing.T) {
	db := findingsCache(t)
	for _, key := range []string{"GHSCAN_SCOPE_ORG", "GHSCAN_SCOPE_REPO", "GHSCAN_SCOPE_DB", "GITHUB_TOKEN", "GH_TOKEN"} {
		t.Setenv(key, "")
	}
	cases := []struct {
		name    string
		args    []string
		count   int
		details bool
		json    bool
	}{
		{"summary", nil, 5, false, false},
		{"details", []string{"--details"}, 5, true, false},
		{"levels", []string{"--level", "critical,high"}, 2, true, false},
		{"repeated levels", []string{"--level", "critical", "--level", "HIGH"}, 2, true, false},
		{"scanner", []string{"--scanner", "semgrep oss"}, 1, true, false},
		{"intersection", []string{"--level", "critical,high", "--scanner", "codeql"}, 1, true, false},
		{"multiple scanners", []string{"--scanner", "CodeQL", "--scanner", "Semgrep OSS"}, 4, true, false},
		{"no matches", []string{"--scanner", "missing"}, 0, true, false},
		{"json", []string{"--json"}, 5, false, true},
		{"filtered json", []string{"--json", "--level", "HIGH"}, 1, false, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			args := append([]string{"--db", db, "--repo", "owner/REPO"}, tt.args...)
			if err := runFindings(context.Background(), args, &b); err != nil {
				t.Fatal(err)
			}
			if tt.json {
				var r app.FindingsReport
				if err := json.Unmarshal(b.Bytes(), &r); err != nil {
					t.Fatal(err)
				}
				if r.Repository != "Owner/repo" || r.TotalOpen != 5 || r.Matching != tt.count || len(r.Findings) != tt.count {
					t.Fatalf("report: %+v", r)
				}
				if tt.count == 5 {
					for i, f := range r.Findings {
						if f.Number != int64(i+1) {
							t.Fatalf("unexpected order: %+v", r.Findings)
						}
					}
				}
				for _, f := range r.Findings {
					if f.Path != "src/file.go" || f.StartLine != 12 || f.EndLine != 14 || f.StartColumn != 2 || f.EndColumn != 8 || f.Message != "Untrusted value" || f.RuleDescription != "Validate input\nbefore use" || f.CommitSHA != "abc123" || len(f.Tags) != 2 || f.ScannerVersion != "1.2" || f.AnalysisKey != "analysis" || f.SeveritySource != "rule.security_severity_level" {
						t.Fatalf("lost detail: %+v", f)
					}
				}
				if tt.count == 1 && (r.Findings[0].Number != 2 || r.Findings[0].Priority != "High" || r.Findings[0].ReportedSeverity != "error") {
					t.Fatalf("priority: %+v", r.Findings)
				}
			} else {
				if !strings.Contains(b.String(), fmt.Sprintf("Open findings: %d matching / 5 total", tt.count)) || strings.Contains(b.String(), "#99") || strings.Contains(b.String(), "#6 ") {
					t.Fatalf("wrong scope: %s", b.String())
				}
				if got := strings.Count(b.String(), "  Message:"); got != 0 && !tt.details || tt.details && got != tt.count {
					t.Fatalf("detail count %d: %s", got, b.String())
				}
				if tt.details && tt.count > 0 && !strings.Contains(b.String(), "Lines: 12-14; columns: 2-8") {
					t.Fatal(b.String())
				}
			}
		})
	}
	t.Setenv("GHSCAN_SCOPE_ORG", "Owner")
	t.Setenv("GHSCAN_SCOPE_REPO", "repo")
	t.Setenv("GHSCAN_SCOPE_DB", db)
	var b bytes.Buffer
	if err := runFindings(context.Background(), nil, &b); err != nil {
		t.Fatal(err)
	}
	var empty app.FindingsReport
	b.Reset()
	if err := runFindings(context.Background(), []string{"--repo", "empty", "--json"}, &b); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b.Bytes(), &empty); err != nil || empty.Findings == nil || empty.TotalOpen != 0 {
		t.Fatalf("empty report: %+v %v", empty, err)
	}
	for _, args := range [][]string{{"--level", "urgent"}, {"--scanner", ","}, {"--repo", "Owner/missing"}, {"--repo", "Other/repo"}, {"extra"}} {
		if err := runFindings(context.Background(), args, &b); err == nil {
			t.Fatalf("expected failure: %v", args)
		}
	}
	missing := filepath.Join(t.TempDir(), "missing.db")
	if err := runFindings(context.Background(), []string{"--db", missing}, &b); err == nil {
		t.Fatal("expected missing database error")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("created missing database: %v", err)
	}
}

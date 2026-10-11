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
		ref := "refs/heads/main"
		switch id {
		case 3:
			ref = "refs/heads/feature"
		case 4:
			ref = "refs/heads/Main"
		case 5:
			ref = ""
		}
		return store.Alert{OrgID: org, RepositoryID: repoID, GitHubID: id, Severity: severity, Tool: tool, RuleID: "rule/test", RuleName: "Unsafe input", RuleDescription: "Validate input\nbefore use", Message: "Untrusted value", Tags: []string{"security", "cwe-20"}, Path: "src/file.go", StartLine: 12, EndLine: 14, StartColumn: 2, EndColumn: 8, Ref: ref, CommitSHA: "abc123", AnalysisKey: "analysis", Environment: "env", Category: "category", ToolVersion: "1.2", ToolGUID: "guid", SeveritySource: "rule.security_severity_level", URL: "https://github.com/Owner/repo/security/code-scanning/1", CreatedAt: now, UpdatedAt: now}
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
	t.Setenv("NO_COLOR", "")
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
		{"verbose", []string{"--verbose"}, 5, true, false},
		{"verbose with levels", []string{"--verbose", "--level", "critical,high"}, 2, true, false},
		{"verbose with scanner", []string{"--verbose", "--scanner", "codeql"}, 3, true, false},
		{"ref", []string{"--ref", "refs/heads/main"}, 2, true, false},
		{"case-sensitive ref", []string{"--ref", "refs/heads/Main"}, 1, true, false},
		{"multiple refs", []string{"--ref", "refs/heads/main,refs/heads/feature"}, 3, true, false},
		{"repeated refs", []string{"--ref", "refs/heads/main", "--ref", "refs/heads/feature"}, 3, true, false},
		{"ref intersection", []string{"--ref", "refs/heads/main", "--level", "critical,high", "--scanner", "codeql", "--verbose"}, 1, true, false},
		{"no ref matches", []string{"--ref", "refs/heads/missing"}, 0, true, false},
		{"ref requires full value", []string{"--ref", "main"}, 0, true, false},
		{"ref json", []string{"--ref", "refs/heads/main", "--json"}, 2, false, true},
		{"verbose json", []string{"--verbose", "--json"}, 5, false, true},
		{"no ref matches json", []string{"--ref", "refs/heads/missing", "--json"}, 0, false, true},
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
				if r.Findings == nil {
					t.Fatal("findings must be an array even without matches")
				}
				if tt.name == "ref json" {
					for _, f := range r.Findings {
						if f.Ref != "refs/heads/main" {
							t.Fatalf("wrong ref: %+v", f)
						}
					}
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
				if tt.name == "filtered json" && (r.Findings[0].Number != 2 || r.Findings[0].Priority != "High" || r.Findings[0].ReportedSeverity != "error") {
					t.Fatalf("priority: %+v", r.Findings)
				}
			} else {
				if !strings.Contains(b.String(), fmt.Sprintf("Open findings: %d matching / 5 total", tt.count)) || strings.Contains(b.String(), "#99") || strings.Contains(b.String(), "#6 ") {
					t.Fatalf("wrong scope: %s", b.String())
				}
				if got := strings.Count(b.String(), "  Message:"); got != 0 && !tt.details || tt.details && got != tt.count {
					t.Fatalf("detail count %d: %s", got, b.String())
				}
				if tt.name == "verbose" || tt.name == "verbose with levels" || tt.name == "verbose with scanner" {
					for _, field := range []string{"Description: Validate input\n    before use", "Message: Untrusted value", "Ref: refs/heads/main", "Commit: abc123", "Analysis key: analysis", "Environment: env", "Scanner version: 1.2", "Scanner GUID: guid", "Reported severity:", "Severity source:"} {
						if !strings.Contains(b.String(), field) {
							t.Fatalf("verbose output missing %q: %s", field, b.String())
						}
					}
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
	for _, args := range [][]string{{"--level", "urgent"}, {"--scanner", ","}, {"--ref", ""}, {"--ref", ","}, {"--repo", "Owner/missing"}, {"--repo", "Other/repo"}, {"extra"}} {
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

func TestFindingsColorAndJSON(t *testing.T) {
	db := findingsCache(t)
	for _, key := range []string{"GHSCAN_SCOPE_ORG", "GHSCAN_SCOPE_REPO", "GHSCAN_SCOPE_DB"} {
		t.Setenv(key, "")
	}
	t.Setenv("NO_COLOR", "")
	if err := os.Unsetenv("NO_COLOR"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"--verbose"}, {"--level", "critical,high"}} {
		var b bytes.Buffer
		if err := runFindings(context.Background(), append([]string{"--db", db, "--repo", "Owner/repo"}, args...), &b); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(b.String(), "\x1b[") {
			t.Fatalf("missing color: %q", b.String())
		}
	}
	var b bytes.Buffer
	if err := runFindings(context.Background(), []string{"--db", db, "--repo", "Owner/repo", "--json"}, &b); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b.Bytes()) || strings.Contains(b.String(), "\x1b") || strings.Contains(b.String(), `\u001b`) {
		t.Fatalf("JSON contains color: %s", b.String())
	}
}

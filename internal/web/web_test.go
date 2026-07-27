package web

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gh "github.com/UnitVectorY-Labs/ghscanscope/internal/github"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
)

type unusedGitHub struct{}

func (unusedGitHub) OrganizationRepositories(context.Context, string) ([]gh.Repository, error) {
	return nil, nil
}
func (unusedGitHub) Repository(context.Context, string, string) (gh.Repository, error) {
	return gh.Repository{}, nil
}
func (unusedGitHub) OrganizationAlerts(context.Context, string) ([]gh.Alert, error) { return nil, nil }
func (unusedGitHub) RepositoryAlerts(context.Context, string, string) ([]gh.Alert, error) {
	return nil, nil
}

func TestDashboardGroupsAlertsAndShowsZeroAlertRepository(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/web.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	orgID, _ := s.EnsureOrg(ctx, "acme", "https://github.com/acme", 1)
	one, _ := s.UpsertRepository(ctx, store.Repository{OrgID: orgID, GitHubID: 1, Owner: "acme", Name: "one", FullName: "acme/one", URL: "https://github.com/acme/one", Visibility: "public"})
	_, _ = s.UpsertRepository(ctx, store.Repository{OrgID: orgID, GitHubID: 2, Owner: "acme", Name: "zero", FullName: "acme/zero", URL: "https://github.com/acme/zero", Visibility: "private"})
	now := time.Now()
	err = s.ReplaceOpenAlerts(ctx, orgID, nil, []store.Alert{{OrgID: orgID, RepositoryID: one, GitHubID: 7, Tool: "CodeQL", RuleID: "go/xss", RuleName: "XSS", Severity: "high", URL: "https://example.test/7", CreatedAt: now, UpdatedAt: now}, {OrgID: orgID, RepositoryID: one, GitHubID: 8, Tool: "CodeQL", RuleID: "go/xss", RuleName: "XSS", Severity: "high", URL: "https://example.test/8", CreatedAt: now, UpdatedAt: now}})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	New(s, unusedGitHub{}).ServeHTTP(recorder, httptest.NewRequest("GET", "/?group_tool=CodeQL&group_rule=go%2Fxss", nil))
	response := recorder.Result()
	body, _ := io.ReadAll(response.Body)
	text := string(body)
	if response.StatusCode != 200 {
		t.Fatalf("status %d: %s", response.StatusCode, text)
	}
	for _, want := range []string{"acme/zero", "go/xss", "<td>2</td>", "https://example.test/7", `hx-post="/sync"`} {
		if !strings.Contains(text, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
}

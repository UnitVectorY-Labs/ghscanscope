package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

type fixture struct {
	store            *store.Store
	handler          http.Handler
	repositoryID     int64
	zeroRepositoryID int64
	alertID          int64
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	s, err := store.Open(t.TempDir() + "/web.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx := context.Background()
	orgID, _ := s.EnsureOrg(ctx, "acme", "https://github.com/acme", 1)
	one, _ := s.UpsertRepository(ctx, store.Repository{OrgID: orgID, GitHubID: 1, Owner: "acme", Name: "one", FullName: "acme/one", URL: "https://github.com/acme/one", Description: "Primary service", Visibility: "public", Language: "Go"})
	zero, _ := s.UpsertRepository(ctx, store.Repository{OrgID: orgID, GitHubID: 2, Owner: "acme", Name: "zero", FullName: "acme/zero", URL: "https://github.com/acme/zero", Visibility: "private"})
	two, _ := s.UpsertRepository(ctx, store.Repository{OrgID: orgID, GitHubID: 3, Owner: "acme", Name: "two", FullName: "acme/two", URL: "https://github.com/acme/two", Visibility: "public", Language: "Go"})
	archived, _ := s.UpsertRepository(ctx, store.Repository{OrgID: orgID, GitHubID: 4, Owner: "acme", Name: "old", FullName: "acme/old", URL: "https://github.com/acme/old", Visibility: "public", Archived: true})
	now := time.Now().UTC()
	alerts := []store.Alert{
		{OrgID: orgID, RepositoryID: one, GitHubID: 7, Tool: "CodeQL", ToolVersion: "2.20", RuleID: "go/xss", RuleName: "Reflected XSS", RuleDescription: "Unsanitized input reaches an HTML response.", Tags: []string{"security", "external/cwe/cwe-079"}, Severity: "high", Message: "This response includes untrusted input.", Ref: "refs/heads/main", CommitSHA: "0123456789abcdef", AnalysisKey: ".github/workflows/codeql.yml:analyze", Environment: "{}", Category: "/language:go", Path: "internal/server.go", StartLine: 42, URL: "https://example.test/7", CreatedAt: now, UpdatedAt: now},
		{OrgID: orgID, RepositoryID: one, GitHubID: 8, Tool: "CodeQL", RuleID: "go/xss", RuleName: "Reflected XSS", Severity: "high", URL: "https://example.test/8", CreatedAt: now, UpdatedAt: now},
		{OrgID: orgID, RepositoryID: two, GitHubID: 9, Tool: "CodeQL", RuleID: "go/xss", RuleName: "Reflected XSS", Severity: "high", URL: "https://example.test/9", CreatedAt: now, UpdatedAt: now},
		{OrgID: orgID, RepositoryID: two, GitHubID: 10, Tool: "CodeQL", RuleID: "go/style", RuleName: "Style issue", Severity: "low", URL: "https://example.test/10", CreatedAt: now, UpdatedAt: now},
		{OrgID: orgID, RepositoryID: archived, GitHubID: 11, Tool: "CodeQL", RuleID: "go/critical", RuleName: "Archived critical", Severity: "critical", URL: "https://example.test/11", CreatedAt: now, UpdatedAt: now},
	}
	if err := s.ReplaceOpenAlerts(ctx, orgID, nil, alerts); err != nil {
		t.Fatal(err)
	}
	stored, _ := s.OpenAlerts(ctx)
	var detailAlertID int64
	for _, alert := range stored {
		if alert.GitHubID == 7 {
			detailAlertID = alert.ID
		}
	}
	return fixture{store: s, handler: New(s, unusedGitHub{}), repositoryID: one, zeroRepositoryID: zero, alertID: detailAlertID}
}

func get(t *testing.T, handler http.Handler, target string) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	response := recorder.Result()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET %s returned %d: %s", target, response.StatusCode, body)
	}
	return string(body)
}

func TestAlertsPageGroupsEquivalentFindingsAndDefaultsToSeverityDescending(t *testing.T) {
	f := newFixture(t)
	body := get(t, f.handler, "/alerts")
	for _, want := range []string{"Alert groups", "Repositories", "Reflected XSS", "2 groups · 4 occurrences", "Repositories", "Occurrences", `aria-label="Filter priorities"`, ">High<", "Tabler Icons</a>"} {
		if !strings.Contains(body, want) {
			t.Errorf("alerts page missing %q", want)
		}
	}
	if strings.Contains(body, "Archived critical") {
		t.Fatal("archived repository alert appeared in the web interface")
	}
	if strings.Index(body, "Reflected XSS") > strings.Index(body, "Style issue") {
		t.Fatal("alerts are not sorted by descending severity by default")
	}
	if !strings.Contains(body, `<td class="alert-count">2</td><td class="alert-count has-alerts mobile-hide">3</td>`) {
		t.Fatal("equivalent finding counts are incorrect")
	}
}

func TestPageLoadsPinnedHTMX4FromUnpkgWithIntegrity(t *testing.T) {
	f := newFixture(t)
	body := get(t, f.handler, "/alerts")
	want := `<script src="https://unpkg.com/htmx.org@4.0.0/dist/htmx.min.js" integrity="sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc" crossorigin="anonymous"></script>`
	if !strings.Contains(body, want) {
		t.Fatalf("page does not load the pinned HTMX 4 unpkg asset with SRI: %s", body)
	}
}

func TestSeverityFilterUsesCanonicalPrioritiesAndDetailsShowProvenance(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	// Add an unsupported scanner value to the existing organization without
	// changing the rest of the fixture's alert set.
	var orgID int64
	if err := f.store.DB.QueryRow(`SELECT id FROM organizations WHERE login='acme'`).Scan(&orgID); err != nil {
		t.Fatal(err)
	}
	if err := f.store.ReplaceOpenAlerts(ctx, orgID, &f.repositoryID, []store.Alert{{OrgID: orgID, RepositoryID: f.repositoryID, GitHubID: 99, Tool: "Other", RuleID: "unknown", RuleName: "Unknown scanner value", Severity: "informational", ReportedSeverity: "informational", SeveritySource: "rule.severity", CreatedAt: now, UpdatedAt: now}}); err != nil {
		t.Fatal(err)
	}
	body := get(t, f.handler, "/alerts?priority=Unknown")
	if !strings.Contains(body, "Unknown scanner value") || !strings.Contains(body, ">Unknown<") || strings.Contains(body, "Reflected XSS") {
		t.Fatalf("canonical Unknown filter failed: %s", body)
	}
	alertID := int64(0)
	if err := f.store.DB.QueryRow(`SELECT id FROM alerts WHERE github_id=99`).Scan(&alertID); err != nil {
		t.Fatal(err)
	}
	detail := get(t, f.handler, "/alerts/"+strconv.FormatInt(alertID, 10))
	if !strings.Contains(detail, "Reported severity: informational (from rule.severity)") {
		t.Fatalf("reported severity provenance is missing: %s", detail)
	}
}

func TestAlertFiltersAreMultiSelectAndSortingIsShareable(t *testing.T) {
	f := newFixture(t)
	body := get(t, f.handler, "/alerts?priority=High&priority=Low&sort=rule&dir=asc")
	for _, want := range []string{`value="High" checked`, `value="Low" checked`, `sort=severity`, "Reflected XSS", "Style issue"} {
		if !strings.Contains(body, want) {
			t.Errorf("multi-filtered page missing %q", want)
		}
	}
}

func TestRepositoriesPageIncludesZeroAlertRepository(t *testing.T) {
	f := newFixture(t)
	body := get(t, f.handler, "/repositories")
	for _, want := range []string{"acme/zero", "Primary service", ">Total ", "High 2", "Low 1", `aria-label="Filter repository priorities"`, `aria-label="View repository alerts"`} {
		if !strings.Contains(body, want) {
			t.Errorf("repositories page missing %q", want)
		}
	}
}

func TestRepositoriesFilterByPriority(t *testing.T) {
	f := newFixture(t)
	body := get(t, f.handler, "/repositories?priority=High&sort=alerts&dir=desc")
	if !strings.Contains(body, `value="High" checked`) || !strings.Contains(body, "acme/one") || !strings.Contains(body, "acme/two") || strings.Contains(body, `href="/repositories/`+strconv.FormatInt(f.zeroRepositoryID, 10)+`">acme/zero`) || strings.Index(body, "acme/one") > strings.Index(body, "acme/two") {
		t.Fatalf("priority repository filter did not keep only matching repositories: %s", body)
	}
}

func TestWebSyncRejectsUnknownOrganization(t *testing.T) {
	f := newFixture(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader("org=new-org"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	f.handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "choose a stored organization") {
		t.Fatalf("unknown organization was not rejected: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestRepositoryRuleAndAlertDrillDowns(t *testing.T) {
	f := newFixture(t)
	repository := get(t, f.handler, "/repositories/"+strconv.FormatInt(f.repositoryID, 10))
	if !strings.Contains(repository, "Refresh repository") || !strings.Contains(repository, "Reflected XSS") || !strings.Contains(repository, `aria-label="Filter tools"`) || !strings.Contains(repository, `class="breadcrumbs"`) {
		t.Fatalf("repository drill-down is incomplete")
	}
	rule := get(t, f.handler, "/rules?tool=CodeQL&rule=go%2Fxss")
	if strings.Contains(rule, "Affected repositories") || !strings.Contains(rule, "<strong>3</strong> occurrences across <strong>2</strong> repositories") || strings.Count(rule, "<table") != 1 || !strings.Contains(rule, "acme/one") || !strings.Contains(rule, `aria-label="View alert details"`) {
		t.Fatalf("rule drill-down is incomplete")
	}
	alert := get(t, f.handler, "/alerts/"+strconv.FormatInt(f.alertID, 10))
	for _, want := range []string{"Finding message", "This response includes untrusted input.", "internal/server.go", "refs/heads/main", "0123456789", "Open on GitHub", "external/cwe/cwe-079", "https://github.com/acme/one/blob/0123456789abcdef/internal/server.go#L42"} {
		if !strings.Contains(alert, want) {
			t.Errorf("alert detail missing %q", want)
		}
	}
}

func TestFiltersSubmitImmediatelyThroughHTMX(t *testing.T) {
	f := newFixture(t)
	body := get(t, f.handler, "/alerts?priority=High&filter_open=priority")
	for _, want := range []string{`hx-get="/alerts"`, `hx-target="#main-content"`, `onchange="this.form.elements.filter_open.value='priority';this.form.requestSubmit()"`, `name="filter_open" value="priority"`} {
		if !strings.Contains(body, want) {
			t.Errorf("immediate filter markup missing %q", want)
		}
	}
	if strings.Contains(body, ">Apply</button>") || !strings.Contains(body, `class="column-filter active" open`) {
		t.Fatal("filters still require Apply or do not remain open after an HTMX swap")
	}
}

func TestSingleOrganizationIsPreselectedForSync(t *testing.T) {
	f := newFixture(t)
	body := get(t, f.handler, "/alerts")
	if !strings.Contains(body, `<option value="acme" selected>acme</option>`) || strings.Contains(body, "Sync organization…") {
		t.Fatal("the only stored organization was not preselected")
	}
}

func TestErrorResponsesRenderHTMXSwappableHTMLPages(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		target  string
		status  int
		message string
	}{
		{"/repositories/999999", http.StatusNotFound, "The requested repository does not exist."},
		{"/alerts/999999", http.StatusNotFound, "The requested alert does not exist."},
		{"/rules", http.StatusBadRequest, "tool and rule are required"},
	}
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		f.handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.target, nil))
		response := recorder.Result()
		body, _ := io.ReadAll(response.Body)
		if response.StatusCode != tc.status {
			t.Fatalf("GET %s returned %d, want %d: %s", tc.target, response.StatusCode, tc.status, body)
		}
		if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
			t.Fatalf("GET %s content type = %q, want text/html so htmx 4 can swap the error", tc.target, contentType)
		}
		for _, want := range []string{`<main id="main-content"`, tc.message, `https://unpkg.com/htmx.org@4.0.0/dist/htmx.min.js`} {
			if !strings.Contains(string(body), want) {
				t.Errorf("GET %s error page missing %q", tc.target, want)
			}
		}
	}
}

func TestRuleFilterCanReturnNoOccurrences(t *testing.T) {
	f := newFixture(t)
	body := get(t, f.handler, "/rules?tool=CodeQL&rule=go%2Fxss&repo=acme%2Fzero&filter_open=repo")
	if !strings.Contains(body, "No occurrences match these filters.") || !strings.Contains(body, "Individual occurrences") {
		t.Fatal("empty rule filtering did not retain the occurrence table")
	}
}

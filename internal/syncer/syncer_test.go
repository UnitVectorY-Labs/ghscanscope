package syncer

import (
	"context"
	"testing"
	"time"

	gh "github.com/UnitVectorY-Labs/ghscanscope/internal/github"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
)

type fakeGitHub struct {
	repos                 []gh.Repository
	repo                  gh.Repository
	orgAlerts, repoAlerts []gh.Alert
}

func (f *fakeGitHub) OrganizationRepositories(context.Context, string) ([]gh.Repository, error) {
	return f.repos, nil
}
func (f *fakeGitHub) Repository(context.Context, string, string) (gh.Repository, error) {
	return f.repo, nil
}
func (f *fakeGitHub) OrganizationAlerts(context.Context, string) ([]gh.Alert, error) {
	return f.orgAlerts, nil
}
func (f *fakeGitHub) RepositoryAlerts(context.Context, string, string) ([]gh.Alert, error) {
	return f.repoAlerts, nil
}

func repository(id int64, name string) gh.Repository {
	var r gh.Repository
	r.ID = id
	r.Name = name
	r.FullName = "acme/" + name
	r.HTMLURL = "https://github.com/" + r.FullName
	r.Owner.Login = "acme"
	r.Visibility = "public"
	return r
}
func alert(number int64, repo gh.Repository, rule string) gh.Alert {
	var a gh.Alert
	a.Number = number
	a.Repository = repo
	a.Tool.Name = "CodeQL"
	a.Rule.ID = rule
	a.Rule.Name = "Test rule"
	a.Rule.Description = "A detailed rule description"
	a.Rule.Tags = []string{"security", "test"}
	a.Rule.SecuritySeverity = "high"
	a.Tool.Version = "2.20.0"
	a.HTMLURL = "https://github.com/" + repo.FullName + "/security/code-scanning/"
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	a.MostRecentInstance.Location.Path = "main.go"
	a.MostRecentInstance.Location.StartLine = 4
	a.MostRecentInstance.Message.Text = "Untrusted data reaches this location"
	a.MostRecentInstance.Ref = "refs/heads/main"
	a.MostRecentInstance.CommitSHA = "0123456789abcdef"
	a.MostRecentInstance.AnalysisKey = ".github/workflows/codeql.yml:analyze"
	return a
}

func TestFullSyncCatalogsZeroAlertRepositoriesAndClosesMissing(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	one, two := repository(1, "one"), repository(2, "two")
	fake := &fakeGitHub{repos: []gh.Repository{one, two}, orgAlerts: []gh.Alert{alert(10, one, "go/sql-injection")}}
	sy := &Syncer{Store: s, GitHub: fake}
	result, err := sy.Sync(ctx, "acme", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Repositories != 2 || result.Alerts != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	repos, _ := s.Repositories(ctx)
	if len(repos) != 2 {
		t.Fatalf("wanted two repositories, got %d", len(repos))
	}
	alerts, _ := s.OpenAlerts(ctx)
	if len(alerts) != 1 {
		t.Fatalf("wanted one alert, got %d", len(alerts))
	}
	if alerts[0].RuleDescription == "" || alerts[0].Message == "" || alerts[0].CommitSHA == "" || len(alerts[0].Tags) != 2 {
		t.Fatalf("alert detail fields were not persisted: %+v", alerts[0])
	}
	fake.orgAlerts = nil
	if _, err = sy.Sync(ctx, "acme", ""); err != nil {
		t.Fatal(err)
	}
	alerts, _ = s.OpenAlerts(ctx)
	if len(alerts) != 0 {
		t.Fatalf("missing alert remained open")
	}
	var retained, open int
	if err = s.DB.QueryRow(`SELECT count(*),sum(is_open) FROM alerts`).Scan(&retained, &open); err != nil {
		t.Fatal(err)
	}
	if retained != 1 || open != 0 {
		t.Fatalf("wanted retained closed alert, got count=%d open=%d", retained, open)
	}
}

func TestTargetedSyncDoesNotCloseOtherRepositoryAlerts(t *testing.T) {
	s, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	one, two := repository(1, "one"), repository(2, "two")
	fake := &fakeGitHub{repos: []gh.Repository{one, two}, orgAlerts: []gh.Alert{alert(10, one, "r1"), alert(20, two, "r2")}}
	sy := &Syncer{Store: s, GitHub: fake}
	if _, err = sy.Sync(ctx, "acme", ""); err != nil {
		t.Fatal(err)
	}
	fake.repo = one
	// The repository alert endpoint can omit the embedded repository object.
	targetAlert := alert(11, gh.Repository{}, "r3")
	fake.repoAlerts = []gh.Alert{targetAlert}
	if _, err = sy.Sync(ctx, "acme", "acme/one"); err != nil {
		t.Fatal(err)
	}
	alerts, _ := s.OpenAlerts(ctx)
	if len(alerts) != 2 || alerts[0].Repository == "" || alerts[1].Repository == "" {
		t.Fatalf("target sync changed other repo alerts: %+v", alerts)
	}
	var emptyRepositories int
	if err = s.DB.QueryRow(`SELECT count(*) FROM repositories WHERE full_name=''`).Scan(&emptyRepositories); err != nil {
		t.Fatal(err)
	}
	if emptyRepositories != 0 {
		t.Fatal("target sync created an empty repository")
	}
}

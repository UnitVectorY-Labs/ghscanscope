package syncer

import (
	"context"
	"fmt"
	"strings"
	"time"

	gh "github.com/UnitVectorY-Labs/ghscanscope/internal/github"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
)

type GitHub interface {
	OrganizationRepositories(context.Context, string) ([]gh.Repository, error)
	Repository(context.Context, string, string) (gh.Repository, error)
	OrganizationAlerts(context.Context, string) ([]gh.Alert, error)
	RepositoryAlerts(context.Context, string, string) ([]gh.Alert, error)
}
type Result struct {
	Repositories, Alerts int
	Duration             time.Duration
}
type Syncer struct {
	Store  *store.Store
	GitHub GitHub
}

func (s *Syncer) Sync(ctx context.Context, org, repo string) (result Result, err error) {
	started := time.Now()
	org = strings.TrimSpace(org)
	if org == "" {
		return result, fmt.Errorf("organization is required")
	}
	orgID, err := s.Store.EnsureOrg(ctx, org, "https://github.com/"+org, 0)
	if err != nil {
		return result, err
	}
	runID, err := s.Store.BeginRun(ctx, orgID)
	if err != nil {
		return result, err
	}
	defer func() {
		result.Duration = time.Since(started)
		status, message := "success", ""
		if err != nil {
			status = "error"
			message = err.Error()
		}
		finishErr := s.Store.FinishRun(context.WithoutCancel(ctx), runID, orgID, status, message, result.Repositories, result.Alerts)
		if err == nil && finishErr != nil {
			err = finishErr
		}
	}()

	repoIDs := map[string]int64{}
	var targetRepoID *int64
	if repo == "" {
		repos, e := s.GitHub.OrganizationRepositories(ctx, org)
		if e != nil {
			return result, e
		}
		for _, r := range repos {
			id, e := s.upsertRepo(ctx, orgID, r)
			if e != nil {
				return result, e
			}
			repoIDs[strings.ToLower(r.FullName)] = id
			result.Repositories++
		}
	} else {
		parts := strings.Split(repo, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return result, fmt.Errorf("repository must be OWNER/REPO")
		}
		if !strings.EqualFold(parts[0], org) {
			return result, fmt.Errorf("repository owner %q does not match organization %q", parts[0], org)
		}
		r, e := s.GitHub.Repository(ctx, parts[0], parts[1])
		if e != nil {
			return result, e
		}
		id, e := s.upsertRepo(ctx, orgID, r)
		if e != nil {
			return result, e
		}
		repoIDs[strings.ToLower(r.FullName)] = id
		targetRepoID = &id
		result.Repositories = 1
	}
	var alerts []gh.Alert
	if repo == "" {
		alerts, err = s.GitHub.OrganizationAlerts(ctx, org)
	} else {
		parts := strings.Split(repo, "/")
		alerts, err = s.GitHub.RepositoryAlerts(ctx, parts[0], parts[1])
	}
	if err != nil {
		return result, err
	}
	stored := make([]store.Alert, 0, len(alerts))
	for _, a := range alerts {
		var repoID int64
		if targetRepoID != nil {
			// Repository-scoped code-scanning responses do not consistently include
			// the embedded repository object, so the requested repository is the
			// authoritative association for every returned alert.
			repoID = *targetRepoID
		} else {
			var ok bool
			repoID, ok = repoIDs[strings.ToLower(a.Repository.FullName)]
			if !ok {
				if strings.TrimSpace(a.Repository.FullName) == "" {
					return result, fmt.Errorf("organization alert %d has no repository", a.Number)
				}
				repoID, err = s.upsertRepo(ctx, orgID, a.Repository)
				if err != nil {
					return result, err
				}
				repoIDs[strings.ToLower(a.Repository.FullName)] = repoID
			}
		}
		reportedSeverity, severitySource := scannerSeverity(a)
		loc := a.MostRecentInstance.Location
		stored = append(stored, store.Alert{
			OrgID: orgID, RepositoryID: repoID, GitHubID: a.Number,
			Tool: a.Tool.Name, ToolVersion: a.Tool.Version, ToolGUID: a.Tool.GUID,
			RuleID: a.Rule.ID, RuleName: a.Rule.Name, RuleDescription: a.Rule.Description,
			Tags: a.Rule.Tags, Severity: reportedSeverity, ReportedSeverity: reportedSeverity,
			Priority: store.CanonicalPriority(reportedSeverity), SeveritySource: severitySource,
			Message: a.MostRecentInstance.Message.Text, Ref: a.MostRecentInstance.Ref,
			CommitSHA: a.MostRecentInstance.CommitSHA, AnalysisKey: a.MostRecentInstance.AnalysisKey,
			Environment: a.MostRecentInstance.Environment, Category: a.MostRecentInstance.Category,
			Path: loc.Path, StartLine: loc.StartLine, EndLine: loc.EndLine,
			StartColumn: loc.StartColumn, EndColumn: loc.EndColumn,
			URL: a.HTMLURL, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt, Open: true,
		})
	}
	if err = s.Store.ReplaceOpenAlerts(ctx, orgID, targetRepoID, stored); err != nil {
		return result, err
	}
	result.Alerts = len(stored)
	return result, nil
}

// scannerSeverity deliberately retains both the exact source value and the
// field GitHub supplied it from. The canonical app priority is derived later.
func scannerSeverity(a gh.Alert) (value, source string) {
	if strings.TrimSpace(a.Rule.SecuritySeverity) != "" && !strings.EqualFold(strings.TrimSpace(a.Rule.SecuritySeverity), "none") {
		return a.Rule.SecuritySeverity, "rule.security_severity_level"
	}
	if strings.TrimSpace(a.Rule.Severity) != "" {
		return a.Rule.Severity, "rule.severity"
	}
	return "", "no severity field"
}

func (s *Syncer) upsertRepo(ctx context.Context, orgID int64, r gh.Repository) (int64, error) {
	visibility := r.Visibility
	if visibility == "" {
		if r.Private {
			visibility = "private"
		} else {
			visibility = "public"
		}
	}
	description, language := "", ""
	if r.Description != nil {
		description = *r.Description
	}
	if r.Language != nil {
		language = *r.Language
	}
	return s.Store.UpsertRepository(ctx, store.Repository{OrgID: orgID, GitHubID: r.ID, Owner: r.Owner.Login, Name: r.Name, FullName: r.FullName, URL: r.HTMLURL, Description: description, Visibility: visibility, Archived: r.Archived, Language: language})
}

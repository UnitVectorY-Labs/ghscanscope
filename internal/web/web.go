package web

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/syncer"
)

type Server struct {
	store    *store.Store
	github   syncer.GitHub
	template *template.Template
	version  string
	mu       sync.Mutex
}

type Stats struct {
	Totals                                   *Stats
	Open, Critical, High, Medium, Low, Other int
	AffectedRepositories, TotalRepositories  int
	LastSync                                 *time.Time
}

type Group struct {
	Totals                                     Stats
	Tool, RuleID, RuleName, Priority, Severity string
	RepositoryCount, AlertCount                int
	Description, URL, Name                     string
	Critical, High, Medium, Low, Other         int
	UpdatedAt                                  time.Time
}

type RepoView struct {
	Totals Stats
	store.Repository
	Org                                string
	AlertCount                         int
	Critical, High, Medium, Low, Other int
}

type FilterChip struct{ Label, URL string }

type countComparison struct{ Shown, Total int }

type severityCounts struct{ Critical, High, Medium, Low, Other int }

type Page struct {
	TotalGroups                                                         int
	View, Title, Eyebrow, Notice, Version, ErrorMessage                 string
	Action, ResetURL, CurrentURL, BackURL, BackLabel, Query, RefreshURL string
	Chips                                                               []FilterChip
	RuleRepositories                                                    []RepoView
	RuleTags                                                            []string
	Organizations                                                       []store.Organization
	Repositories                                                        []RepoView
	AllRepositories                                                     []RepoView
	Groups                                                              []Group
	Alerts                                                              []store.Alert
	Repository                                                          *RepoView
	Alert                                                               *store.Alert
	Stats                                                               Stats
	Filters                                                             map[string][]string
	Tools, Rules                                                        []string
	Severities, Visibilities                                            []string
	Languages                                                           []string
	RuleTool, RuleID, RuleName                                          string
	RulePriority, RuleSeverity, RuleDescription, RuleMessage            string
	RuleRepositoryCount                                                 int
	FilterOpen                                                          string
	SortLinks                                                           map[string]string
	RepoSortLinks                                                       map[string]string
	AlertSortLinks                                                      map[string]string
	SortColumn, SortDirection                                           string
}

type dataSet struct {
	Organizations []store.Organization
	Repositories  []store.Repository
	Alerts        []store.Alert
	OrgNames      map[int64]string
	RepoByID      map[int64]store.Repository
}

func New(s *store.Store, g syncer.GitHub, version string) http.Handler {
	srv := &Server{store: s, github: g, version: version, template: template.Must(template.New("page").Funcs(template.FuncMap{
		"countOf": func(shown, total int) countComparison {
			return countComparison{Shown: shown, Total: total}
		},
		"shortsha": func(value string) string {
			if len(value) > 10 {
				return value[:10]
			}
			return value
		},
		"sortGlyph": func(current, column, direction string) string {
			if current != column {
				return "↕"
			}
			if direction == "desc" {
				return "↓"
			}
			return "↑"
		},
		"sortAria": func(current, column, direction string) string {
			if current != column {
				return "none"
			}
			if direction == "desc" {
				return "descending"
			}
			return "ascending"
		},
		"severityClass": severityClass,
		"scannerClass":  scannerClass,
		"selected":      selected,
		"codeURL":       codeURL,
		"findingName":   findingName,
		"repoURL": func(page Page, id int64) string {
			return pageLink(page, "/repositories/"+strconv.FormatInt(id, 10), nil)
		},
		"alertURL": func(page Page, id int64) string {
			return "/alerts/" + strconv.FormatInt(id, 10) + "?from=" + url.QueryEscape(page.CurrentURL)
		},
		"ruleURL": func(page Page, tool, rule string) string {
			return pageLink(page, "/rules", url.Values{"tool": {tool}, "rule": {rule}})
		},
		"lineRange": lineRange,
	}).Parse(pageTemplate))}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", srv.home)
	mux.HandleFunc("GET /alerts", srv.alerts)
	mux.HandleFunc("GET /alerts/{id}", srv.alertDetail)
	mux.HandleFunc("GET /repositories", srv.repositories)
	mux.HandleFunc("GET /repositories/{id}", srv.repositoryDetail)
	mux.HandleFunc("GET /rules", srv.ruleDetail)
	mux.HandleFunc("POST /sync", srv.syncOrganization)
	mux.HandleFunc("POST /repositories/{id}/sync", srv.syncRepository)
	return mux
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/alerts", http.StatusTemporaryRedirect)
}

func (s *Server) load(r *http.Request) (dataSet, error) {
	ctx := r.Context()
	organizations, err := s.store.Organizations(ctx)
	if err != nil {
		return dataSet{}, err
	}
	allRepositories, err := s.store.Repositories(ctx)
	if err != nil {
		return dataSet{}, err
	}
	allAlerts, err := s.store.OpenAlerts(ctx)
	if err != nil {
		return dataSet{}, err
	}
	d := dataSet{Organizations: organizations, OrgNames: map[int64]string{}, RepoByID: map[int64]store.Repository{}}
	for _, organization := range organizations {
		d.OrgNames[organization.ID] = organization.Login
	}
	active := map[int64]bool{}
	for _, repository := range allRepositories {
		if repository.Archived {
			continue
		}
		d.Repositories = append(d.Repositories, repository)
		d.RepoByID[repository.ID] = repository
		active[repository.ID] = true
	}
	for _, alert := range allAlerts {
		if active[alert.RepositoryID] {
			d.Alerts = append(d.Alerts, alert)
		}
	}
	return d, nil
}

func (s *Server) alerts(w http.ResponseWriter, r *http.Request) {
	d, err := s.load(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	filters := queryFilters(r)
	if len(filters["priority"]) == 0 && r.URL.Query().Get("scope") != "all" {
		filters["priority"] = []string{"Critical", "High"}
	}
	filtered := filterAlerts(d, filters)
	groups := groupAlerts(filtered)
	totals := map[string]Stats{}
	for _, group := range groupAlerts(d.Alerts) {
		totals[group.Tool+"\x00"+group.RuleID] = groupStats(group)
	}
	for i := range groups {
		groups[i].Totals = totals[groups[i].Tool+"\x00"+groups[i].RuleID]
	}
	sortColumn, sortDirection := sortChoice(r, "sort", "dir", "severity", "desc")
	sortGroups(groups, sortColumn, sortDirection)
	page := s.basePage(d, filters, filtered)
	page.View, page.Title, page.Eyebrow = "alerts", "Issues across repositories", "Cross-repository priorities"
	page.Groups = groups
	page.TotalGroups = len(totals)
	for i := range page.Groups {
		page.Groups[i].URL = drillURL(r, filters, "/rules", url.Values{"tool": {groups[i].Tool}, "rule": {groups[i].RuleID}})
	}
	s.preparePage(&page, r, d)
	page.FilterOpen = r.URL.Query().Get("filter_open")
	page.SortColumn, page.SortDirection = sortColumn, sortDirection
	page.SortLinks = sortLinks(r, "sort", "dir", []string{"rule", "severity", "tool", "repositories", "alerts", "updated"})
	if r.URL.Query().Get("sync") == "success" {
		page.Notice = "Organization data refreshed successfully."
	}
	s.render(w, page)
}

func (s *Server) repositories(w http.ResponseWriter, r *http.Request) {
	d, err := s.load(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	filters := queryFilters(r)
	alertFilters := map[string][]string{}
	for key, values := range filters {
		if key != "q" {
			alertFilters[key] = values
		}
	}
	matchingAlerts := filterAlerts(d, alertFilters)
	counts := repositoryCounts(matchingAlerts)
	severityCounts := repositorySeverityCounts(matchingAlerts)
	totals := repositoryAlertStats(d.Alerts)
	matchingRepositories := map[int64]bool{}
	var repositoryViews []RepoView
	for _, repository := range d.Repositories {
		if !matchesSearch(filters, repository.FullName, repository.Description) || !matchesRepository(filters, repository, d.OrgNames[repository.OrgID]) || (!matchesRepositoryPriority(filters, severityCounts[repository.ID]) || ((len(filters["tool"]) > 0 || len(filters["rule"]) > 0) && counts[repository.ID] == 0)) {
			continue
		}
		matchingRepositories[repository.ID] = true
		view := RepoView{Repository: repository, Org: d.OrgNames[repository.OrgID], AlertCount: counts[repository.ID]}
		applySeverityCounts(&view, severityCounts[repository.ID])
		view.Totals = totals[repository.ID]
		repositoryViews = append(repositoryViews, view)
	}
	statsAlerts := make([]store.Alert, 0)
	for _, alert := range matchingAlerts {
		if matchingRepositories[alert.RepositoryID] && allows(filters["priority"], alert.Priority) {
			statsAlerts = append(statsAlerts, alert)
		}
	}
	page := s.basePage(d, filters, statsAlerts)
	page.View, page.Title, page.Eyebrow = "repositories", "Repositories", "Repository priorities"
	page.Repositories = repositoryViews
	sortColumn, sortDirection := sortChoice(r, "sort", "dir", "priority", "desc")
	sortRepositories(page.Repositories, sortColumn, sortDirection)
	page.SortColumn, page.SortDirection = sortColumn, sortDirection
	page.FilterOpen = r.URL.Query().Get("filter_open")
	page.Stats.TotalRepositories = len(repositoryViews)
	page.SortLinks = sortLinks(r, "sort", "dir", []string{"repository", "organization", "visibility", "language", "alerts", "priority"})
	s.preparePage(&page, r, d)
	s.render(w, page)
}

func (s *Server) repositoryDetail(w http.ResponseWriter, r *http.Request) {
	s.repositoryPage(w, r, "")
}

func (s *Server) repositoryPage(w http.ResponseWriter, r *http.Request, notice string) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.errorPage(w, http.StatusNotFound, "Page not found", "The requested repository does not exist.")
		return
	}
	// Refresh renders this same page. Subsequent GET filters and sort links must
	// use the repository URL rather than the POST-only /sync endpoint.
	if r.Method == http.MethodPost {
		r = r.Clone(r.Context())
		canonical := *r.URL
		canonical.Path = "/repositories/" + strconv.FormatInt(id, 10)
		r.URL = &canonical
	}
	d, err := s.load(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	repository, ok := d.RepoByID[id]
	if !ok {
		s.errorPage(w, http.StatusNotFound, "Page not found", "The requested repository does not exist.")
		return
	}
	filters := queryFilters(r)
	alerts := make([]store.Alert, 0)
	for _, alert := range d.Alerts {
		if alert.RepositoryID == id && matchesAlert(filters, alert, repository) {
			alerts = append(alerts, alert)
		}
	}
	sortColumn, sortDirection := sortChoice(r, "sort", "dir", "severity", "desc")
	sortAlerts(alerts, sortColumn, sortDirection)
	page := s.basePage(d, filters, alerts)
	view := RepoView{Repository: repository, Org: d.OrgNames[repository.OrgID], AlertCount: len(alerts)}
	page.View, page.Title, page.Eyebrow = "repository", repository.FullName, "Repository details"
	page.Repository, page.Alerts, page.Notice = &view, alerts, notice
	page.FilterOpen = r.URL.Query().Get("filter_open")
	page.Stats.TotalRepositories = 1
	total := statsForRepository(d, id, d.Alerts)
	page.Stats.Totals = &total
	page.SortColumn, page.SortDirection = sortColumn, sortDirection
	page.SortLinks = sortLinks(r, "sort", "dir", []string{"rule", "severity", "tool", "location", "updated"})
	s.preparePage(&page, r, d)
	s.render(w, page)
}

func (s *Server) ruleDetail(w http.ResponseWriter, r *http.Request) {
	tool, ruleID := strings.TrimSpace(r.URL.Query().Get("tool")), strings.TrimSpace(r.URL.Query().Get("rule"))
	if tool == "" || ruleID == "" {
		s.errorPage(w, http.StatusBadRequest, "Invalid request", "tool and rule are required")
		return
	}
	d, err := s.load(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	filters := queryFilters(r)
	delete(filters, "tool")
	delete(filters, "rule")
	allOccurrences := make([]store.Alert, 0)
	for _, alert := range d.Alerts {
		if alert.Tool == tool && alert.RuleID == ruleID {
			allOccurrences = append(allOccurrences, alert)
		}
	}
	if len(allOccurrences) == 0 {
		s.errorPage(w, http.StatusNotFound, "Page not found", "No open alerts match this rule.")
		return
	}
	alerts := make([]store.Alert, 0, len(allOccurrences))
	for _, alert := range allOccurrences {
		if matchesAlert(filters, alert, d.RepoByID[alert.RepositoryID]) {
			alerts = append(alerts, alert)
		}
	}
	alertSort, alertDirection := sortChoice(r, "alert_sort", "alert_dir", "repository", "asc")
	sortAlerts(alerts, alertSort, alertDirection)
	page := s.basePage(d, filters, alerts)
	representative := allOccurrences[0]
	page.View, page.Title, page.Eyebrow = "rule", findingName(representative), "Issue group"
	page.RuleTool, page.RuleID, page.RuleName = tool, ruleID, representative.RuleName
	page.RulePriority, page.RuleSeverity, page.RuleDescription = representative.Priority, representative.Priority, representative.RuleDescription
	page.Alerts = alerts
	page.RuleRepositoryCount = len(repositoryCounts(alerts))
	total := calculateStats(d, nil, allOccurrences)
	page.Stats.Totals = &total
	page.RuleDescription = explanatoryDescription(page.RuleDescription, ruleID)
	page.RuleTags = representative.Tags
	for _, occurrence := range alerts {
		if occurrence.Message != "" {
			page.RuleMessage = occurrence.Message
			break
		}
	}
	for _, occurrence := range allOccurrences {
		if page.RuleDescription == "" {
			page.RuleDescription = explanatoryDescription(occurrence.RuleDescription, ruleID)
		}
		if len(page.RuleTags) == 0 {
			page.RuleTags = occurrence.Tags
		}
	}
	if groups := groupAlerts(alerts); len(groups) > 0 {
		page.RulePriority = groups[0].Priority
		page.RuleSeverity = groups[0].Priority
	}
	counts := repositoryCounts(alerts)
	severities := repositorySeverityCounts(alerts)
	totals := repositoryAlertStats(allOccurrences)
	for _, repo := range d.Repositories {
		if counts[repo.ID] > 0 {
			view := RepoView{Repository: repo, Org: d.OrgNames[repo.OrgID], AlertCount: counts[repo.ID]}
			applySeverityCounts(&view, severities[repo.ID])
			view.Totals = totals[repo.ID]
			page.RuleRepositories = append(page.RuleRepositories, view)
		}
	}
	sortRepositories(page.RuleRepositories, "alerts", "desc")
	page.FilterOpen = r.URL.Query().Get("filter_open")
	page.SortColumn, page.SortDirection = alertSort, alertDirection
	page.AlertSortLinks = sortLinks(r, "alert_sort", "alert_dir", []string{"repository", "location", "message", "updated"})
	s.preparePage(&page, r, d)
	s.render(w, page)
}

func (s *Server) alertDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.errorPage(w, http.StatusNotFound, "Page not found", "The requested alert does not exist.")
		return
	}
	d, err := s.load(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	var selectedAlert *store.Alert
	for i := range d.Alerts {
		if d.Alerts[i].ID == id {
			selectedAlert = &d.Alerts[i]
			break
		}
	}
	if selectedAlert == nil {
		s.errorPage(w, http.StatusNotFound, "Page not found", "The requested alert does not exist.")
		return
	}
	repository := d.RepoByID[selectedAlert.RepositoryID]
	repoView := RepoView{Repository: repository, Org: d.OrgNames[repository.OrgID], AlertCount: repositoryCounts(d.Alerts)[repository.ID]}
	page := s.basePage(d, map[string][]string{}, []store.Alert{*selectedAlert})
	page.View, page.Title, page.Eyebrow = "alert", findingName(*selectedAlert), "Individual occurrence"
	page.Alert, page.Repository = selectedAlert, &repoView
	page.Stats.TotalRepositories = 1
	s.preparePage(&page, r, d)
	s.render(w, page)
}

func (s *Server) syncOrganization(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.errorPage(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}
	org := strings.TrimSpace(r.FormValue("org"))
	organizations, err := s.store.Organizations(r.Context())
	if err != nil {
		s.serverError(w, err)
		return
	}
	known := false
	for _, organization := range organizations {
		if strings.EqualFold(organization.Login, org) {
			org = organization.Login
			known = true
			break
		}
	}
	if !known {
		s.errorPage(w, http.StatusBadRequest, "Invalid request", "choose a stored organization")
		return
	}
	s.mu.Lock()
	_, err = (&syncer.Syncer{Store: s.store, GitHub: s.github}).Sync(r.Context(), org, "")
	s.mu.Unlock()
	if err != nil {
		s.errorPage(w, http.StatusBadGateway, "Sync failed", err.Error())
		return
	}
	http.Redirect(w, r, "/alerts?sync=success", http.StatusSeeOther)
}

func (s *Server) syncRepository(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.errorPage(w, http.StatusNotFound, "Page not found", "The requested repository does not exist.")
		return
	}
	d, err := s.load(r)
	if err != nil {
		s.serverError(w, err)
		return
	}
	repository, ok := d.RepoByID[id]
	if !ok {
		s.errorPage(w, http.StatusNotFound, "Page not found", "The requested repository does not exist.")
		return
	}
	s.mu.Lock()
	_, err = (&syncer.Syncer{Store: s.store, GitHub: s.github}).Sync(r.Context(), d.OrgNames[repository.OrgID], repository.FullName)
	s.mu.Unlock()
	if err != nil {
		s.errorPage(w, http.StatusBadGateway, "Sync failed", err.Error())
		return
	}
	s.repositoryPage(w, r, "Repository data refreshed successfully.")
}

func (s *Server) basePage(d dataSet, filters map[string][]string, scopedAlerts []store.Alert) Page {
	tools, rules, visibilities, languages := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, alert := range d.Alerts {
		tools[alert.Tool], rules[alert.RuleID] = true, true
	}
	counts := repositoryCounts(d.Alerts)
	total := calculateStats(d, nil, d.Alerts)
	page := Page{Organizations: d.Organizations, Stats: calculateStats(d, filters, scopedAlerts), Filters: filters, Version: s.version}
	page.Stats.Totals = &total
	for _, repository := range d.Repositories {
		visibilities[repository.Visibility], languages[repository.Language] = true, true
		page.AllRepositories = append(page.AllRepositories, RepoView{Repository: repository, Org: d.OrgNames[repository.OrgID], AlertCount: counts[repository.ID]})
	}
	page.Tools, page.Rules, page.Severities = keys(tools), keys(rules), []string{"Critical", "High", "Medium", "Low", "Unknown"}
	page.Visibilities, page.Languages = keys(visibilities), keys(languages)
	return page
}

func groupStats(group Group) Stats {
	return Stats{Open: group.AlertCount, AffectedRepositories: group.RepositoryCount, Critical: group.Critical, High: group.High, Medium: group.Medium, Low: group.Low, Other: group.Other}
}

func statsForRepository(d dataSet, id int64, alerts []store.Alert) Stats {
	var scoped []store.Alert
	for _, alert := range alerts {
		if alert.RepositoryID == id {
			scoped = append(scoped, alert)
		}
	}
	return calculateStats(d, nil, scoped)
}

func groupAlerts(alerts []store.Alert) []Group {
	groups := map[string]*Group{}
	repositories := map[string]map[int64]bool{}
	for _, alert := range alerts {
		key := alert.Tool + "\x00" + alert.RuleID
		group := groups[key]
		if group == nil {
			group = &Group{Tool: alert.Tool, RuleID: alert.RuleID, RuleName: alert.RuleName, Priority: alert.Priority, Severity: alert.Priority, UpdatedAt: alert.UpdatedAt}
			group.Name, group.Description = findingName(alert), explanatoryDescription(alert.RuleDescription, alert.RuleID)
			groups[key], repositories[key] = group, map[int64]bool{}
		}
		if group.Description == "" {
			group.Description = explanatoryDescription(alert.RuleDescription, alert.RuleID)
		}
		switch severityClass(alert.Priority) {
		case "critical":
			group.Critical++
		case "high":
			group.High++
		case "medium":
			group.Medium++
		case "low":
			group.Low++
		default:
			group.Other++
		}
		group.AlertCount++
		repositories[key][alert.RepositoryID] = true
		group.RepositoryCount = len(repositories[key])
		if severityRank(alert.Priority) > severityRank(group.Priority) {
			group.Priority = alert.Priority
			group.Severity = alert.Priority
		}
		if alert.UpdatedAt.After(group.UpdatedAt) {
			group.UpdatedAt = alert.UpdatedAt
		}
	}
	out := make([]Group, 0, len(groups))
	for _, group := range groups {
		out = append(out, *group)
	}
	return out
}

func queryFilters(r *http.Request) map[string][]string {
	filters := map[string][]string{}
	for _, key := range []string{"org", "repo", "priority", "tool", "rule", "visibility", "language", "q"} {
		for _, value := range r.URL.Query()[key] {
			if value = strings.TrimSpace(value); value != "" {
				filters[key] = append(filters[key], value)
			}
		}
	}
	return filters
}

func filterAlerts(d dataSet, filters map[string][]string) []store.Alert {
	out := make([]store.Alert, 0, len(d.Alerts))
	for _, alert := range d.Alerts {
		if matchesAlert(filters, alert, d.RepoByID[alert.RepositoryID]) {
			out = append(out, alert)
		}
	}
	return out
}

func matchesAlert(filters map[string][]string, alert store.Alert, repository store.Repository) bool {
	return allows(filters["org"], alert.Org) && allows(filters["repo"], alert.Repository) &&
		allows(filters["priority"], alert.Priority) && allows(filters["tool"], alert.Tool) &&
		allows(filters["rule"], alert.RuleID) && allows(filters["visibility"], repository.Visibility) &&
		allows(filters["language"], repository.Language) && matchesSearch(filters, alert.RuleName, alert.RuleID, alert.RuleDescription, alert.Message, alert.Path, alert.Repository, alert.Tool, strings.Join(alert.Tags, " "))
}

func matchesRepository(filters map[string][]string, repository store.Repository, org string) bool {
	return allows(filters["org"], org) && allows(filters["repo"], repository.FullName) &&
		allows(filters["visibility"], repository.Visibility) && allows(filters["language"], repository.Language)
}

func matchesRepositoryPriority(filters map[string][]string, counts severityCounts) bool {
	priorities := filters["priority"]
	if len(priorities) == 0 {
		return true
	}
	for _, priority := range priorities {
		switch strings.ToLower(priority) {
		case "critical":
			if counts.Critical > 0 {
				return true
			}
		case "high":
			if counts.High > 0 {
				return true
			}
		case "medium":
			if counts.Medium > 0 {
				return true
			}
		case "low":
			if counts.Low > 0 {
				return true
			}
		case "unknown":
			if counts.Other > 0 {
				return true
			}
		}
	}
	return false
}

func allows(values []string, candidate string) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if strings.EqualFold(value, candidate) {
			return true
		}
	}
	return false
}

func selected(values []string, candidate string) bool { return allowsSelected(values, candidate) }

func allowsSelected(values []string, candidate string) bool {
	for _, value := range values {
		if strings.EqualFold(value, candidate) {
			return true
		}
	}
	return false
}

func calculateStats(d dataSet, filters map[string][]string, alerts []store.Alert) Stats {
	stats := Stats{Open: len(alerts)}
	repositories := map[int64]bool{}
	for _, alert := range alerts {
		repositories[alert.RepositoryID] = true
		switch severityClass(alert.Priority) {
		case "critical":
			stats.Critical++
		case "high":
			stats.High++
		case "medium":
			stats.Medium++
		case "low":
			stats.Low++
		default:
			stats.Other++
		}
	}
	stats.AffectedRepositories = len(repositories)
	for _, repository := range d.Repositories {
		if matchesRepository(filters, repository, d.OrgNames[repository.OrgID]) {
			stats.TotalRepositories++
		}
	}
	for _, organization := range d.Organizations {
		if !allows(filters["org"], organization.Login) {
			continue
		}
		if organization.LastSync != nil && (stats.LastSync == nil || organization.LastSync.After(*stats.LastSync)) {
			value := *organization.LastSync
			stats.LastSync = &value
		}
	}
	return stats
}

func repositoryCounts(alerts []store.Alert) map[int64]int {
	counts := map[int64]int{}
	for _, alert := range alerts {
		counts[alert.RepositoryID]++
	}
	return counts
}

func repositorySeverityCounts(alerts []store.Alert) map[int64]severityCounts {
	counts := map[int64]severityCounts{}
	for _, alert := range alerts {
		value := counts[alert.RepositoryID]
		switch severityClass(alert.Priority) {
		case "critical":
			value.Critical++
		case "high":
			value.High++
		case "medium":
			value.Medium++
		case "low":
			value.Low++
		default:
			value.Other++
		}
		counts[alert.RepositoryID] = value
	}
	return counts
}

// repositoryAlertStats aggregates only the counts used by repository rows.
func repositoryAlertStats(alerts []store.Alert) map[int64]Stats {
	stats := map[int64]Stats{}
	for id, counts := range repositorySeverityCounts(alerts) {
		stats[id] = Stats{
			Open:     counts.Critical + counts.High + counts.Medium + counts.Low + counts.Other,
			Critical: counts.Critical, High: counts.High, Medium: counts.Medium, Low: counts.Low, Other: counts.Other,
		}
	}
	return stats
}

func applySeverityCounts(view *RepoView, counts severityCounts) {
	view.Critical, view.High, view.Medium, view.Low, view.Other = counts.Critical, counts.High, counts.Medium, counts.Low, counts.Other
}

func sortChoice(r *http.Request, sortKey, directionKey, defaultSort, defaultDirection string) (string, string) {
	column, direction := r.URL.Query().Get(sortKey), r.URL.Query().Get(directionKey)
	if column == "" {
		column, direction = defaultSort, defaultDirection
	}
	if direction != "asc" && direction != "desc" {
		direction = defaultDirection
	}
	return column, direction
}

func sortLinks(r *http.Request, sortKey, directionKey string, columns []string) map[string]string {
	currentColumn, currentDirection := r.URL.Query().Get(sortKey), r.URL.Query().Get(directionKey)
	links := map[string]string{}
	for _, column := range columns {
		values := cloneValues(r.URL.Query())
		direction := "asc"
		if currentColumn == column && currentDirection != "desc" {
			direction = "desc"
		}
		values.Set(sortKey, column)
		values.Set(directionKey, direction)
		links[column] = r.URL.Path + "?" + values.Encode()
	}
	return links
}

func cloneValues(source url.Values) url.Values {
	clone := url.Values{}
	for key, values := range source {
		clone[key] = append([]string(nil), values...)
	}
	return clone
}

func sortGroups(groups []Group, column, direction string) {
	sort.SliceStable(groups, func(i, j int) bool {
		left, right := groups[i], groups[j]
		comparison := 0
		switch column {
		case "rule":
			comparison = strings.Compare(strings.ToLower(left.Name+left.RuleID), strings.ToLower(right.Name+right.RuleID))
		case "tool":
			comparison = strings.Compare(strings.ToLower(left.Tool), strings.ToLower(right.Tool))
		case "repositories":
			comparison = left.RepositoryCount - right.RepositoryCount
		case "alerts":
			comparison = left.AlertCount - right.AlertCount
		case "updated":
			comparison = left.UpdatedAt.Compare(right.UpdatedAt)
		default:
			comparison = severityRank(left.Priority) - severityRank(right.Priority)
			if comparison == 0 {
				comparison = left.RepositoryCount - right.RepositoryCount
				if comparison == 0 {
					comparison = left.AlertCount - right.AlertCount
				}
			}
		}
		if comparison == 0 {
			return left.Tool+left.RuleID < right.Tool+right.RuleID
		}
		return ordered(comparison, direction)
	})
}

func sortRepositories(repositories []RepoView, column, direction string) {
	sort.SliceStable(repositories, func(i, j int) bool {
		left, right := repositories[i], repositories[j]
		comparison := 0
		switch column {
		case "organization":
			comparison = strings.Compare(strings.ToLower(left.Org), strings.ToLower(right.Org))
		case "visibility":
			comparison = strings.Compare(left.Visibility, right.Visibility)
		case "language":
			comparison = strings.Compare(strings.ToLower(left.Language), strings.ToLower(right.Language))
		case "alerts":
			comparison = left.AlertCount - right.AlertCount
		case "priority":
			comparison = left.Critical - right.Critical
			if comparison == 0 {
				comparison = left.High - right.High
			}
			if comparison == 0 {
				comparison = left.AlertCount - right.AlertCount
			}
		case "critical":
			comparison = left.Critical - right.Critical
		case "high":
			comparison = left.High - right.High
		case "medium":
			comparison = left.Medium - right.Medium
		case "low":
			comparison = left.Low - right.Low
		default:
			comparison = strings.Compare(strings.ToLower(left.FullName), strings.ToLower(right.FullName))
		}
		return ordered(comparison, direction)
	})
}

func sortAlerts(alerts []store.Alert, column, direction string) {
	sort.SliceStable(alerts, func(i, j int) bool {
		left, right := alerts[i], alerts[j]
		comparison := 0
		switch column {
		case "rule":
			comparison = strings.Compare(strings.ToLower(left.RuleName+left.RuleID), strings.ToLower(right.RuleName+right.RuleID))
		case "tool":
			comparison = strings.Compare(strings.ToLower(left.Tool), strings.ToLower(right.Tool))
		case "repository":
			comparison = strings.Compare(strings.ToLower(left.Repository), strings.ToLower(right.Repository))
		case "location":
			comparison = strings.Compare(strings.ToLower(left.Path), strings.ToLower(right.Path))
		case "message":
			comparison = strings.Compare(strings.ToLower(left.Message), strings.ToLower(right.Message))
		case "updated":
			comparison = left.UpdatedAt.Compare(right.UpdatedAt)
		default:
			comparison = severityRank(left.Priority) - severityRank(right.Priority)
		}
		return ordered(comparison, direction)
	})
}

func ordered(comparison int, direction string) bool {
	if direction == "desc" {
		return comparison > 0
	}
	return comparison < 0
}

func severityRank(value string) int {
	switch strings.ToLower(value) {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	default:
		return 1
	}
}

func severityClass(value string) string {
	switch severityRank(value) {
	case 5:
		return "critical"
	case 4:
		return "high"
	case 3:
		return "medium"
	case 2:
		return "low"
	default:
		return "unknown"
	}
}

func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		if key != "" {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func codeURL(alert store.Alert) string {
	if alert.RepositoryURL == "" || alert.Path == "" {
		return alert.URL
	}
	revision := alert.CommitSHA
	if revision == "" {
		revision = strings.TrimPrefix(alert.Ref, "refs/heads/")
	}
	if revision == "" {
		revision = "HEAD"
	}
	parts := strings.Split(alert.Path, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	result := strings.TrimRight(alert.RepositoryURL, "/") + "/blob/" + url.PathEscape(revision) + "/" + strings.Join(parts, "/")
	if alert.StartLine > 0 {
		result += "#L" + strconv.Itoa(alert.StartLine)
		if alert.EndLine > alert.StartLine {
			result += "-L" + strconv.Itoa(alert.EndLine)
		}
	}
	return result
}

func (s *Server) render(w http.ResponseWriter, page Page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.template.Execute(w, page); err != nil {
		s.serverError(w, err)
	}
}

func (s *Server) serverError(w http.ResponseWriter, err error) {
	s.errorPage(w, http.StatusInternalServerError, "Something went wrong", fmt.Sprintf("ghscanscope: %v", err))
}

// errorPage renders a full HTML document so htmx 4, which swaps 4xx and 5xx
// responses by default, can select #main-content or body and swap in a
// styled error instead of wiping the target with a text/plain body.
func (s *Server) errorPage(w http.ResponseWriter, status int, title, message string) {
	page := Page{View: "error", Title: title, Eyebrow: "Error", ErrorMessage: message, Version: s.version}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = s.template.Execute(w, page)
}

// Names remain grounded in scanner text rather than inventing remediation advice.
func findingName(a store.Alert) string {
	name := a.RuleName
	if name != "" && name != a.RuleID && (strings.ContainsAny(name, " ") || !strings.ContainsAny(name, "-_.")) {
		return name
	}
	if description := explanatoryDescription(a.RuleDescription, a.RuleID); description != "" {
		return description
	}
	if name == "" || name == a.RuleID {
		name = a.RuleID
		if i := strings.LastIndexAny(name, "/."); i >= 0 {
			name = name[i+1:]
		}
	}
	// Readable formatting of the scanner's identifier; the exact ID is also shown.
	if strings.ContainsAny(name, "-_") && !strings.HasPrefix(name, "GO-") {
		name = strings.NewReplacer("-", " ", "_", " ").Replace(name)
		if len(name) > 0 {
			name = strings.ToUpper(name[:1]) + name[1:]
		}
	}
	return name
}
func lineRange(a store.Alert) string {
	if a.Path == "" {
		return "Location not reported"
	}
	result := a.Path
	if a.StartLine > 0 {
		result += ":" + strconv.Itoa(a.StartLine)
		if a.StartColumn > 0 {
			result += ":" + strconv.Itoa(a.StartColumn)
		}
		if a.EndLine > a.StartLine || (a.EndLine == a.StartLine && a.EndColumn > a.StartColumn) {
			result += "–" + strconv.Itoa(a.EndLine)
			if a.EndColumn > 0 {
				result += ":" + strconv.Itoa(a.EndColumn)
			}
		}
	}
	return result
}
func matchesSearch(filters map[string][]string, fields ...string) bool {
	haystack := strings.ToLower(strings.Join(fields, " "))
	for _, query := range filters["q"] {
		for _, term := range strings.Fields(strings.ToLower(query)) {
			if !strings.Contains(haystack, term) {
				return false
			}
		}
	}
	return true
}
func safeBack(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return ""
	}
	if parsed.Path == "/alerts" || parsed.Path == "/repositories" || parsed.Path == "/rules" {
		return parsed.RequestURI()
	}
	for _, prefix := range []string{"/repositories/", "/alerts/"} {
		if strings.HasPrefix(parsed.Path, prefix) {
			if id, err := strconv.ParseInt(strings.TrimPrefix(parsed.Path, prefix), 10, 64); err == nil && id > 0 {
				return parsed.RequestURI()
			}
		}
	}
	return ""
}
func drillURL(r *http.Request, filters map[string][]string, path string, extra url.Values) string {
	values := url.Values{"scope": {"all"}}
	for key, list := range filters {
		values[key] = append([]string(nil), list...)
	}
	for key, list := range extra {
		values[key] = list
	}
	values.Set("from", r.URL.Path+"?"+effectiveQuery(r, filters).Encode())
	return path + "?" + values.Encode()
}
func effectiveQuery(r *http.Request, filters map[string][]string) url.Values {
	values := cloneValues(r.URL.Query())
	for key, list := range filters {
		values[key] = append([]string(nil), list...)
	}
	values.Set("scope", "all")
	return values
}
func (s *Server) preparePage(page *Page, r *http.Request, d dataSet) {
	page.Action = r.URL.Path
	refresh := *r.URL
	refresh.Path += "/sync"
	page.RefreshURL = refresh.RequestURI()
	values := effectiveQuery(r, page.Filters)
	page.CurrentURL = r.URL.Path + "?" + values.Encode()
	page.BackURL = safeBack(r.URL.Query().Get("from"))
	if page.BackURL == "" {
		page.BackURL = "/alerts"
		if page.View == "repository" {
			page.BackURL = "/repositories"
		}
	}
	page.BackLabel = "previous view"
	if parent, err := url.Parse(page.BackURL); err == nil {
		switch {
		case parent.Path == "/alerts":
			page.BackLabel = "issues"
		case parent.Path == "/repositories":
			page.BackLabel = "repositories"
		case parent.Path == "/rules":
			page.BackLabel = "issue group"
		case strings.HasPrefix(parent.Path, "/alerts/"):
			page.BackLabel = "finding"
		case strings.HasPrefix(parent.Path, "/repositories/"):
			page.BackLabel = "repository findings"
		}
	}
	page.Query = strings.Join(page.Filters["q"], " ")
	reset := url.Values{"scope": {"all"}}
	if page.View == "rule" {
		reset.Set("tool", page.RuleTool)
		reset.Set("rule", page.RuleID)
	}
	if from := safeBack(r.URL.Query().Get("from")); from != "" {
		reset.Set("from", from)
	}
	page.ResetURL = r.URL.Path + "?" + reset.Encode()
	for _, key := range []string{"priority", "tool", "org", "repo", "rule", "visibility", "language", "q"} {
		for _, value := range page.Filters[key] {
			remove := cloneValues(values)
			remove.Del(key)
			for _, other := range page.Filters[key] {
				if other != value {
					remove.Add(key, other)
				}
			}
			page.Chips = append(page.Chips, FilterChip{Label: key + ": " + value, URL: r.URL.Path + "?" + remove.Encode()})
		}
	}
	// Keep filter options local to the repository or rule being inspected.
	if page.View == "repository" || page.View == "rule" {
		tools, rules := map[string]bool{}, map[string]bool{}
		for _, alert := range d.Alerts {
			if page.View == "repository" && alert.RepositoryID != page.Repository.ID {
				continue
			}
			if page.View == "rule" && (alert.Tool != page.RuleTool || alert.RuleID != page.RuleID) {
				continue
			}
			tools[alert.Tool], rules[alert.RuleID] = true, true
		}
		page.Tools, page.Rules = keys(tools), keys(rules)
	}
}

func pageLink(page Page, path string, extra url.Values) string {
	values := url.Values{"scope": {"all"}, "from": {page.CurrentURL}}
	for key, list := range page.Filters {
		if key != "q" || page.View != "repositories" {
			values[key] = append([]string(nil), list...)
		}
	}
	if page.View == "rule" {
		values.Set("tool", page.RuleTool)
		values.Set("rule", page.RuleID)
	}
	for key, list := range extra {
		values[key] = list
	}
	return path + "?" + values.Encode()
}

func explanatoryDescription(description, ruleID string) string {
	text := strings.TrimSpace(description)
	if text == ruleID || text == "Semgrep Finding: "+ruleID {
		return ""
	}
	return description
}

// Scanner colors use a fixed neutral palette; unknown tools receive a generic badge.
func scannerClass(tool string) string {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case "codeql":
		return "codeql"
	case "semgrep", "semgrep oss":
		return "semgrep"
	case "govulncheck":
		return "govulncheck"
	case "zizmor":
		return "zizmor"
	default:
		return "other"
	}
}

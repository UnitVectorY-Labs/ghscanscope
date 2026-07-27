package web

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/syncer"
)

type Server struct {
	store    *store.Store
	github   syncer.GitHub
	template *template.Template
	mu       sync.Mutex
}
type Group struct {
	Tool, RuleID, RuleName, Severity string
	RepositoryCount, AlertCount      int
}
type RepoView struct {
	store.Repository
	Org        string
	AlertCount int
}
type Page struct {
	Organizations                          []store.Organization
	Repositories                           []RepoView
	Groups                                 []Group
	Alerts                                 []store.Alert
	Filters                                map[string]string
	Tools, Rules, Severities, Visibilities []string
	Notice, Error                          string
}

func New(s *store.Store, g syncer.GitHub) http.Handler {
	srv := &Server{store: s, github: g, template: template.Must(template.New("dashboard").Funcs(template.FuncMap{"timefmt": func(v any) string {
		if v == nil {
			return "Never"
		}
		return fmt.Sprint(v)
	}}).Parse(pageTemplate))}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", srv.dashboard)
	mux.HandleFunc("POST /sync", srv.sync)
	return mux
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	page, err := s.buildPage(r)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := s.template.Execute(w, page); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
func (s *Server) sync(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	org, repo := strings.TrimSpace(r.FormValue("org")), strings.TrimSpace(r.FormValue("repo"))
	if org == "" {
		http.Error(w, "organization is required", 400)
		return
	}
	s.mu.Lock()
	_, err := (&syncer.Syncer{Store: s.store, GitHub: s.github}).Sync(r.Context(), org, repo)
	s.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	http.Redirect(w, r, "/?org="+org, http.StatusSeeOther)
}

func (s *Server) buildPage(r *http.Request) (Page, error) {
	ctx := r.Context()
	orgs, err := s.store.Organizations(ctx)
	if err != nil {
		return Page{}, err
	}
	repos, err := s.store.Repositories(ctx)
	if err != nil {
		return Page{}, err
	}
	alerts, err := s.store.OpenAlerts(ctx)
	if err != nil {
		return Page{}, err
	}
	f := map[string]string{}
	for _, key := range []string{"org", "repo", "severity", "tool", "rule", "visibility", "archived", "group_tool", "group_rule"} {
		f[key] = r.URL.Query().Get(key)
	}
	orgNames := map[int64]string{}
	for _, o := range orgs {
		orgNames[o.ID] = o.Login
	}
	repoMap := map[int64]store.Repository{}
	for _, repo := range repos {
		repoMap[repo.ID] = repo
	}
	setTool, setRule, setSeverity, setVisibility := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, a := range alerts {
		setTool[a.Tool] = true
		setRule[a.RuleID] = true
		setSeverity[a.Severity] = true
	}
	for _, repo := range repos {
		setVisibility[repo.Visibility] = true
	}
	filtered := make([]store.Alert, 0)
	for _, a := range alerts {
		repo := repoMap[a.RepositoryID]
		if !match(f, a, repo) {
			continue
		}
		filtered = append(filtered, a)
	}
	groups := map[string]*Group{}
	repoSets := map[string]map[int64]bool{}
	for _, a := range filtered {
		key := a.Tool + "\x00" + a.RuleID
		g := groups[key]
		if g == nil {
			g = &Group{Tool: a.Tool, RuleID: a.RuleID, RuleName: a.RuleName, Severity: a.Severity}
			groups[key] = g
			repoSets[key] = map[int64]bool{}
		}
		g.AlertCount++
		repoSets[key][a.RepositoryID] = true
		g.RepositoryCount = len(repoSets[key])
	}
	groupList := make([]Group, 0, len(groups))
	for _, g := range groups {
		groupList = append(groupList, *g)
	}
	sort.Slice(groupList, func(i, j int) bool {
		if groupList[i].AlertCount != groupList[j].AlertCount {
			return groupList[i].AlertCount > groupList[j].AlertCount
		}
		return groupList[i].Tool+groupList[i].RuleID < groupList[j].Tool+groupList[j].RuleID
	})
	repoCounts := map[int64]int{}
	for _, a := range filtered {
		repoCounts[a.RepositoryID]++
	}
	var repoViews []RepoView
	for _, repo := range repos {
		if f["org"] != "" && !strings.EqualFold(orgNames[repo.OrgID], f["org"]) {
			continue
		}
		if f["repo"] != "" && !strings.EqualFold(repo.FullName, f["repo"]) {
			continue
		}
		if f["visibility"] != "" && repo.Visibility != f["visibility"] {
			continue
		}
		if f["archived"] != "" && strconv.FormatBool(repo.Archived) != f["archived"] {
			continue
		}
		repoViews = append(repoViews, RepoView{Repository: repo, Org: orgNames[repo.OrgID], AlertCount: repoCounts[repo.ID]})
	}
	drill := filtered
	if f["group_tool"] != "" || f["group_rule"] != "" {
		drill = nil
		for _, a := range filtered {
			if a.Tool == f["group_tool"] && a.RuleID == f["group_rule"] {
				drill = append(drill, a)
			}
		}
	} else {
		drill = nil
	}
	return Page{Organizations: orgs, Repositories: repoViews, Groups: groupList, Alerts: drill, Filters: f, Tools: keys(setTool), Rules: keys(setRule), Severities: keys(setSeverity), Visibilities: keys(setVisibility)}, nil
}
func match(f map[string]string, a store.Alert, r store.Repository) bool {
	return (f["org"] == "" || strings.EqualFold(a.Org, f["org"])) && (f["repo"] == "" || strings.EqualFold(a.Repository, f["repo"])) && (f["severity"] == "" || a.Severity == f["severity"]) && (f["tool"] == "" || a.Tool == f["tool"]) && (f["rule"] == "" || a.RuleID == f["rule"]) && (f["visibility"] == "" || r.Visibility == f["visibility"]) && (f["archived"] == "" || strconv.FormatBool(r.Archived) == f["archived"])
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

const pageTemplate = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>ghscanscope</title><script src="https://unpkg.com/htmx.org@2.0.4"></script><style>
:root{font-family:ui-sans-serif,system-ui;color:#172033;background:#f6f8fb}body{margin:0}header{background:#172033;color:white;padding:1rem 2rem;display:flex;align-items:center;justify-content:space-between}main{max-width:1400px;margin:auto;padding:1.5rem}.card{background:white;border:1px solid #dbe1ea;border-radius:9px;padding:1rem;margin-bottom:1rem;box-shadow:0 1px 2px #0001}.filters{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:.75rem}.filters label,.sync label{font-size:.78rem;color:#536174}.filters select,.sync select,.sync input{display:block;width:100%;box-sizing:border-box;padding:.5rem;margin-top:.25rem;border:1px solid #bdc6d3;border-radius:5px;background:white}button,.button{background:#2857c5;color:white;border:0;border-radius:5px;padding:.55rem .9rem;text-decoration:none;cursor:pointer}table{width:100%;border-collapse:collapse;font-size:.9rem}th,td{text-align:left;padding:.65rem;border-bottom:1px solid #e5e9ef}th{color:#536174}a{color:#2857c5}.pill{padding:.15rem .4rem;border-radius:1rem;background:#eef2f8;font-size:.78rem}.muted{color:#68758a}.sync{display:flex;gap:.75rem;align-items:end}.sync label{min-width:200px}h1,h2{margin-top:0}details{margin-bottom:1rem}@media(max-width:700px){header{padding:1rem}main{padding:.75rem}.sync{display:block}.sync>*{margin-bottom:.5rem}.scroll{overflow:auto}}
</style></head><body><header><strong>ghscanscope</strong><span>Local code-scanning explorer</span></header><main>
<section class="card"><h2>Refresh data</h2><form class="sync" method="post" action="/sync" hx-post="/sync" hx-target="body" hx-push-url="true" hx-disabled-elt="button"><label>Organization<select name="org" required><option value="">Choose…</option>{{range .Organizations}}<option value="{{.Login}}">{{.Login}}</option>{{end}}</select></label><label>Repository (optional)<input name="repo" placeholder="OWNER/REPO"></label><button type="submit">Sync from GitHub</button></form><p class="muted">{{range .Organizations}}<strong>{{.Login}}</strong>: {{if .LastSync}}{{.LastSync.Format "2006-01-02 15:04:05 MST"}}{{else}}Never{{end}} ({{.LastStatus}})&nbsp; {{end}}</p></section>
<section class="card"><h2>Filters</h2><form method="get" class="filters"><label>Organization<select name="org"><option value="">All</option>{{range .Organizations}}<option value="{{.Login}}" {{if eq $.Filters.org .Login}}selected{{end}}>{{.Login}}</option>{{end}}</select></label><label>Repository<select name="repo"><option value="">All</option>{{range .Repositories}}<option value="{{.FullName}}" {{if eq $.Filters.repo .FullName}}selected{{end}}>{{.FullName}}</option>{{end}}</select></label><label>Severity<select name="severity"><option value="">All</option>{{range .Severities}}<option {{if eq $.Filters.severity .}}selected{{end}}>{{.}}</option>{{end}}</select></label><label>Tool<select name="tool"><option value="">All</option>{{range .Tools}}<option {{if eq $.Filters.tool .}}selected{{end}}>{{.}}</option>{{end}}</select></label><label>Rule<select name="rule"><option value="">All</option>{{range .Rules}}<option {{if eq $.Filters.rule .}}selected{{end}}>{{.}}</option>{{end}}</select></label><label>Visibility<select name="visibility"><option value="">All</option>{{range .Visibilities}}<option {{if eq $.Filters.visibility .}}selected{{end}}>{{.}}</option>{{end}}</select></label><label>Archived<select name="archived"><option value="">All</option><option value="false" {{if eq .Filters.archived "false"}}selected{{end}}>Active</option><option value="true" {{if eq .Filters.archived "true"}}selected{{end}}>Archived</option></select></label><div><button type="submit">Apply filters</button></div></form></section>
<section class="card"><h2>Alert groups <span class="pill">{{len .Groups}}</span></h2><div class="scroll"><table><thead><tr><th>Tool</th><th>Rule</th><th>Name</th><th>Severity</th><th>Repositories</th><th>Alerts</th><th></th></tr></thead><tbody>{{range .Groups}}<tr><td>{{.Tool}}</td><td><code>{{.RuleID}}</code></td><td>{{.RuleName}}</td><td><span class="pill">{{.Severity}}</span></td><td>{{.RepositoryCount}}</td><td>{{.AlertCount}}</td><td><a href="?org={{urlquery $.Filters.org}}&repo={{urlquery $.Filters.repo}}&severity={{urlquery $.Filters.severity}}&tool={{urlquery $.Filters.tool}}&rule={{urlquery $.Filters.rule}}&visibility={{urlquery $.Filters.visibility}}&archived={{urlquery $.Filters.archived}}&group_tool={{urlquery .Tool}}&group_rule={{urlquery .RuleID}}#alerts">View</a></td></tr>{{else}}<tr><td colspan="7" class="muted">No open alerts match these filters.</td></tr>{{end}}</tbody></table></div></section>
{{if or .Filters.group_tool .Filters.group_rule}}<section class="card" id="alerts"><h2>Alerts: {{.Filters.group_tool}} / {{.Filters.group_rule}}</h2><div class="scroll"><table><thead><tr><th>Repository</th><th>Severity</th><th>Location</th><th>Updated</th><th></th></tr></thead><tbody>{{range .Alerts}}<tr><td>{{.Repository}}</td><td>{{.Severity}}</td><td><code>{{.Path}}{{if .StartLine}}:{{.StartLine}}{{end}}</code></td><td>{{.UpdatedAt.Format "2006-01-02"}}</td><td><a href="{{.URL}}" target="_blank" rel="noopener">GitHub ↗</a></td></tr>{{end}}</tbody></table></div></section>{{end}}
<section class="card"><h2>Repositories <span class="pill">{{len .Repositories}}</span></h2><div class="scroll"><table><thead><tr><th>Repository</th><th>Description</th><th>Visibility</th><th>Language</th><th>Archived</th><th>Open alerts</th></tr></thead><tbody>{{range .Repositories}}<tr><td><a href="{{.URL}}" target="_blank" rel="noopener">{{.FullName}}</a></td><td>{{.Description}}</td><td>{{.Visibility}}</td><td>{{.Language}}</td><td>{{if .Archived}}Yes{{else}}No{{end}}</td><td>{{.AlertCount}}</td></tr>{{else}}<tr><td colspan="6" class="muted">No repositories stored. Run a sync to begin.</td></tr>{{end}}</tbody></table></div></section>
</main></body></html>`

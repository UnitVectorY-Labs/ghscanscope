package web

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
)

func negotiatedGet(handler http.Handler, target, accept string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.Header.Set("Accept", accept)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestRepresentationNegotiation(t *testing.T) {
	f := newFixture(t)
	for _, tc := range []struct {
		name, accept string
		markdown     bool
	}{
		{"no Accept", "", false},
		{"browser", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8", false},
		{"markdown", "text/markdown", true},
		{"markdown preferred", "text/markdown, text/html;q=0.8", true},
		{"html preferred", "text/html, text/markdown;q=0.5", false},
		{"markdown forbidden", "text/markdown;q=0", false},
		{"wildcard", "*/*", false},
		{"text wildcard", "text/*", false},
		{"equal explicit", "text/markdown, text/html", false},
		{"explicit versus wildcard", "text/markdown, */*", true},
		{"specific markdown forbidden", "text/markdown;q=0, */*;q=1", false},
		{"specific html lower", "text/html;q=0.2, text/markdown;q=0.5, text/*;q=1", true},
		{"wildcard preferred", "text/markdown;q=0.5, */*;q=0.8", false},
		{"html excluded", "text/html;q=0, text/markdown;q=0.1", true},
		{"case insensitive", "TEXT/MARKDOWN;Q=0.9, text/html;q=0.1", true},
		{"utf8", "text/markdown;charset=UTF-8", true},
		{"specific charset excluded", "text/markdown;charset=utf-8;q=0, text/markdown;q=1", false},
		{"specific html charset", "text/html;charset=utf-8;q=0.1, text/html;q=1, text/markdown;q=0.8", true},
		{"unsupported variant", "text/markdown;variant=unknown", false},
		{"unsupported charset", "text/markdown;charset=iso-8859-1", false},
		{"invalid q", "text/markdown;q=2", false},
		{"negative q", "text/markdown;q=-1", false},
		{"nan q", "text/markdown;q=NaN", false},
		{"excess precision", "text/markdown;q=0.9999", false},
		{"quoted comma", `application/example;value="a,b", text/markdown`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := negotiatedGet(f.handler, "/alerts", tc.accept)
			wantType := "text/html; charset=utf-8"
			if tc.markdown {
				wantType = "text/markdown; charset=utf-8"
			}
			if w.Code != 200 || w.Header().Get("Content-Type") != wantType {
				t.Fatalf("status %d, type %q", w.Code, w.Header().Get("Content-Type"))
			}
			if w.Header().Get("Vary") != "Accept" {
				t.Fatalf("missing Vary: Accept: %v", w.Header())
			}
			if tc.markdown != strings.Contains(w.Body.String(), "# Issues across repositories") {
				t.Fatal("incorrect representation body")
			}
		})
	}
	// Request body type and client identity must not affect negotiation.
	r := httptest.NewRequest("GET", "/alerts", nil)
	r.Header.Set("Content-Type", "text/markdown")
	r.Header.Set("User-Agent", "ChatGPT agent")
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	if w.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatal("request Content-Type or user-agent changed representation")
	}
	r.Header.Add("Accept", "text/html;q=0.2")
	r.Header.Add("Accept", "text/markdown;q=0.8")
	w = httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	if w.Header().Get("Content-Type") != "text/markdown; charset=utf-8" {
		t.Fatal("multiple Accept field lines not combined")
	}
}

func TestMarkdownContentAndLinks(t *testing.T) {
	f := newFixture(t)
	repo := strconv.FormatInt(f.repositoryID, 10)
	alert := strconv.FormatInt(f.alertID, 10)
	for _, tc := range []struct {
		target string
		wants  []string
	}{
		{"/alerts", []string{"# Issues across repositories", "Reflected XSS", "Open occurrences: 3 / 4", "Issue groups: 1 / 2", "High 3 / 3", "http://example.com/rules?", "rule=go%2Fxss", "priority=High"}},
		{"/repositories", []string{"# Repositories", "acme/one", "Primary service", "Visibility: private", "No open alerts does not establish scanning coverage", "http://example.com/repositories/" + repo + "?"}},
		{"/repositories/" + repo, []string{"# acme/one", "Primary service", "Visibility: public", "Language: Go", "Open occurrences: 2 / 2", "This response includes untrusted input", markdownText("internal/server.go:42"), "http://example.com/alerts/" + alert + "?from=", "https://github.com/acme/one"}},
		{"/rules?tool=CodeQL&rule=go%2Fxss&repo=acme%2Fone", []string{"# Reflected XSS", "Open occurrences: 2 / 3", "Affected repositories: 1 / 2", "## Affected repositories", "Rule: go/xss", "Scanner: CodeQL", "Unsanitized input reaches an HTML response", "external/cwe/cwe", "http://example.com/repositories/" + repo + "?", "repo=acme%2Fone", "http://example.com/alerts/" + alert + "?from="}},
		{"/alerts/" + alert, []string{"# Reflected XSS", "Status: Open", "GitHub alert: 7", "Priority: High", "Scanner version: 2", "Rule: go/xss", "## Finding message", "This response includes untrusted input", "0123456789abcdef", "Reported severity: high", "Severity source: legacy alerts", "Analysis key:", "Environment:", "Category: /language:go", "First reported:", "Alert updated:", "https://github.com/acme/one/blob/0123456789abcdef/internal/server.go#L42", "https://example.test/7", "http://example.com/rules?"}},
	} {
		t.Run(tc.target, func(t *testing.T) {
			md := negotiatedGet(f.handler, tc.target, "text/markdown")
			html := negotiatedGet(f.handler, tc.target, "text/html")
			if md.Code != 200 || html.Code != 200 {
				t.Fatalf("statuses: md %d, html %d", md.Code, html.Code)
			}
			body := md.Body.String()
			for _, want := range tc.wants {
				if !strings.Contains(body, want) {
					t.Errorf("missing %q in:\n%s", want, body)
				}
			}
			for _, forbidden := range []string{"<script", "<style", "<svg", "<form", "<div", "class=", "hx-", "AllRepositories", "ToolGUID", "Archived critical", "/sync", "theme-toggle"} {
				if strings.Contains(body, forbidden) {
					t.Errorf("unexpected UI/internal content %q", forbidden)
				}
			}
			if !strings.Contains(body, "](<http://example.com/repositories>)") {
				t.Error("missing related link")
			}
			if len(body)*2 >= html.Body.Len() {
				t.Errorf("Markdown not substantially smaller: %d vs %d", len(body), html.Body.Len())
			}
		})
	}
}

func TestMarkdownScopeAndVisibilityMatchHTML(t *testing.T) {
	f := newFixture(t)
	for _, accept := range []string{"text/html", "text/markdown"} {
		for _, tc := range []struct{ target, included, excluded string }{
			{"/alerts", "Reflected XSS", "Style issue"},
			{"/alerts?scope=all", "Style issue", "Archived critical"},
			{"/alerts?priority=Low", "Style issue", "Reflected XSS"},
			{"/repositories?q=Primary", "Primary service", "acme/old"},
			{"/repositories/" + strconv.FormatInt(f.zeroRepositoryID, 10), "acme/zero", "Reflected XSS"},
		} {
			w := negotiatedGet(f.handler, tc.target, accept)
			if w.Code != 200 || !strings.Contains(w.Body.String(), tc.included) || strings.Contains(w.Body.String(), tc.excluded) {
				t.Fatalf("scope mismatch %s %s: %s", accept, tc.target, w.Body.String())
			}
		}
	}
	// Closed alerts and archived repositories are excluded by the shared loader.
	if _, err := f.store.DB.Exec("UPDATE alerts SET is_open=0 WHERE id=$1", f.alertID); err != nil {
		t.Fatal(err)
	}
	for _, accept := range []string{"text/html", "text/markdown"} {
		w := negotiatedGet(f.handler, "/alerts/"+strconv.FormatInt(f.alertID, 10), accept)
		if w.Code != 404 {
			t.Fatalf("closed alert visible in %s: %d", accept, w.Code)
		}
	}
}

func TestMarkdownStatusesRedirectsAndHead(t *testing.T) {
	f := newFixture(t)
	var archivedID int64
	if err := f.store.DB.QueryRow("SELECT id FROM repositories WHERE archived=1").Scan(&archivedID); err != nil {
		t.Fatal(err)
	}
	for _, accept := range []string{"text/html", "text/markdown"} {
		for _, tc := range []struct {
			target string
			status int
		}{
			{"/", 307}, {"/unknown", 404}, {"/alerts/not-an-id", 404}, {"/alerts/999999", 404},
			{"/repositories/no", 404}, {"/repositories/999999", 404}, {"/repositories/" + strconv.FormatInt(archivedID, 10), 404},
			{"/rules", 400}, {"/rules?tool=CodeQL&rule=missing", 404},
		} {
			w := negotiatedGet(f.handler, tc.target, accept)
			if w.Code != tc.status {
				t.Errorf("%s %s: %d, expected %d", accept, tc.target, w.Code, tc.status)
			}
			if tc.status >= 400 && accept == "text/markdown" {
				if w.Header().Get("Content-Type") != "text/markdown; charset=utf-8" || strings.Contains(w.Body.String(), "<!doctype") || !strings.Contains(w.Body.String(), "# ") {
					t.Errorf("incorrect Markdown error body: %s", w.Body.String())
				}
			}
			if tc.status == 307 && w.Header().Get("Location") != "/alerts" {
				t.Error("redirect changed")
			}
		}
		r := httptest.NewRequest("HEAD", "/alerts", nil)
		r.Header.Set("Accept", accept)
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.Len() != 0 || !strings.HasPrefix(w.Header().Get("Content-Type"), accept) {
			t.Fatalf("HEAD: %d, %v, body %d", w.Code, w.Header(), w.Body.Len())
		}
	}
	// A database failure must not become a successful Markdown page.
	if err := f.store.Close(); err != nil {
		t.Fatal(err)
	}
	for _, accept := range []string{"text/html", "text/markdown"} {
		if w := negotiatedGet(f.handler, "/alerts", accept); w.Code != 500 {
			t.Fatalf("database error: %s %d", accept, w.Code)
		}
	}
}

func TestNegotiatedResponsesCannotMixInCaches(t *testing.T) {
	f := newFixture(t)
	for _, accept := range []string{"text/html", "text/markdown", "text/html"} {
		r := httptest.NewRequest("GET", "/alerts", nil)
		r.Header.Set("Accept", accept)
		r.Header.Set("If-None-Match", `"some-other-representation"`)
		r.Header.Set("If-Modified-Since", "Wed, 21 Oct 2037 07:28:00 GMT")
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), accept) {
			t.Fatalf("wrong cached representation: %d %v", w.Code, w.Header())
		}
		if w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("Vary") != "Accept" || w.Header().Get("ETag") != "" || w.Header().Get("Last-Modified") != "" {
			t.Fatalf("unsafe cache headers: %v", w.Header())
		}
	}
}

func TestMarkdownUsesExistingAccessBoundary(t *testing.T) {
	f := newFixture(t)
	// The app has no built-in login. Deployments can apply an access gate to the
	// same handler; Accept cannot take a second route around it.
	gated := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Header.Get("Authorization") {
		case "":
			http.Error(w, "Unauthorized", 401)
		case "Bearer forbidden":
			http.Error(w, "Forbidden", 403)
		default:
			f.handler.ServeHTTP(w, r)
		}
	})
	for _, accept := range []string{"text/html", "text/markdown"} {
		for _, tc := range []struct {
			auth   string
			status int
		}{{"", 401}, {"Bearer forbidden", 403}, {"Bearer allowed", 200}} {
			r := httptest.NewRequest("GET", "/alerts", nil)
			r.Header.Set("Accept", accept)
			r.Header.Set("Authorization", tc.auth)
			w := httptest.NewRecorder()
			gated.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("%s auth %q: status %d", accept, tc.auth, w.Code)
			}
			if tc.status != 200 && strings.Contains(w.Body.String(), "Reflected XSS") {
				t.Fatal("data leaked through access gate")
			}
		}
	}
}

func TestMarkdownEscapesScannerTextAndUnsafeLinks(t *testing.T) {
	f := newFixture(t)
	malicious := "<script>alert('x')</script>\n# Injected heading\n[bad](javascript:alert(1))\n![image](https://bad.test)"
	if _, err := f.store.DB.Exec("UPDATE alerts SET message=$1, url=$2 WHERE id=$3", malicious, "javascript:alert(1)", f.alertID); err != nil {
		t.Fatal(err)
	}
	w := negotiatedGet(f.handler, "/alerts/"+strconv.FormatInt(f.alertID, 10), "text/markdown")
	if !strings.Contains(w.Body.String(), markdownText(malicious)) || strings.Contains(w.Body.String(), "<script>") || strings.Contains(w.Body.String(), "](<javascript:") {
		t.Fatalf("unsafe Markdown: %s", w.Body.String())
	}
	r := httptest.NewRequest("GET", "https://app.example/alerts?q=a%26b", nil)
	r.Header.Set("X-Forwarded-Host", "evil.test")
	m := markdownWriter{request: r}
	if got := m.absolute("/repositories?repo=acme%2Fone&scope=all"); got != "https://app.example/repositories?repo=acme%2Fone&scope=all" {
		t.Fatalf("wrong canonical URL: %s", got)
	}
	// Unused store fields must never be serialized automatically.
	body := renderMarkdown(r, Page{View: "alert", Title: "Finding", Alert: &store.Alert{ToolGUID: "hidden-tool-guid"}, Repository: &RepoView{}, AllRepositories: []RepoView{{Repository: store.Repository{FullName: "hidden-repo"}}}})
	if strings.Contains(body, "hidden-tool-guid") || strings.Contains(body, "hidden-repo") {
		t.Fatal("non-display data leaked")
	}
}

func TestBufferedRepresentationsEscapeUntrustedContent(t *testing.T) {
	f := newFixture(t)
	payload := `<img src=x onerror="alert('xss')"><script>alert('xss')</script>`
	if _, err := f.store.DB.Exec("UPDATE alerts SET message=$1, rule_description=$1, url=$2 WHERE id=$3", payload, "javascript:alert(1)", f.alertID); err != nil {
		t.Fatal(err)
	}
	for _, accept := range []string{"text/html", "text/markdown"} {
		t.Run(accept, func(t *testing.T) {
			w := negotiatedGet(f.handler, "/alerts/"+strconv.FormatInt(f.alertID, 10), accept)
			if w.Code != http.StatusOK || w.Header().Get("Content-Type") != accept+"; charset=utf-8" {
				t.Fatalf("unexpected response: %d %v", w.Code, w.Header())
			}
			if w.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("missing MIME sniffing protection")
			}
			body := w.Body.String()
			escaped := template.HTMLEscapeString(payload)
			if accept == "text/markdown" {
				escaped = markdownText(payload)
			}
			if strings.Count(body, escaped) != 2 {
				t.Fatalf("message and description must remain escaped: %s", body)
			}
			for _, unsafe := range []string{"<img src=x", "<script>alert", `href="javascript:`, "](<javascript:"} {
				if strings.Contains(body, unsafe) {
					t.Errorf("unsafe rendered content: %q", unsafe)
				}
			}
		})
	}
}

func TestMarkdownEmptyFilteredView(t *testing.T) {
	f := newFixture(t)
	w := negotiatedGet(f.handler, "/alerts?q=absent", "text/markdown")
	if !strings.Contains(w.Body.String(), "No issue groups match these filters") || !strings.Contains(w.Body.String(), "q=absent") {
		t.Fatal("empty filtered resource lost context")
	}
	// Context cancellation follows the same database/error path for both formats.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, accept := range []string{"text/html", "text/markdown"} {
		r := httptest.NewRequest("GET", "/alerts", nil).WithContext(ctx)
		r.Header.Set("Accept", accept)
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		if w.Code != 500 {
			t.Fatalf("cancelled request changed status: %d", w.Code)
		}
	}
}

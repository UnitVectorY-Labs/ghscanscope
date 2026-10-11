package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOrganizationRepositoriesPaginationAndHeaders(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing authorization")
		}
		if r.URL.Path != "/orgs/acme/repos" || r.URL.Query().Get("type") != "all" || r.URL.Query().Get("per_page") != "100" {
			t.Errorf("unexpected request %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "1" {
			fmt.Fprint(w, "[")
			for i := 0; i < 100; i++ {
				if i > 0 {
					fmt.Fprint(w, ",")
				}
				fmt.Fprintf(w, `{"id":%d,"full_name":"acme/r%d"}`, i+1, i+1)
			}
			fmt.Fprint(w, "]")
			return
		}
		fmt.Fprint(w, `[{"id":101,"full_name":"acme/r101"}]`)
	}))
	defer server.Close()
	c := &Client{HTTP: server.Client(), BaseURL: server.URL, Token: "test-token"}
	var pages, counts []int
	c.OnPage = func(path string, page, items int) {
		if !strings.HasPrefix(path, "/orgs/acme/repos") {
			t.Fatalf("wrong progress path: %s", path)
		}
		pages = append(pages, page)
		counts = append(counts, items)
	}
	repos, err := c.OrganizationRepositories(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 || pages[0] != 1 || pages[1] != 2 || counts[0] != 100 || counts[1] != 101 {
		t.Fatalf("page progress: %v %v", pages, counts)
	}
	if len(repos) != 101 || requests != 2 {
		t.Fatalf("got %d repositories in %d requests", len(repos), requests)
	}
}

func TestRepositoryAlertsRequestsOnlyOpen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/widget/code-scanning/alerts" || r.URL.Query().Get("state") != "open" {
			t.Errorf("unexpected URL: %s", r.URL.String())
		}
		fmt.Fprint(w, `[]`)
	}))
	defer server.Close()
	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.RepositoryAlerts(context.Background(), "acme", "widget")
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPIErrorIncludesGitHubMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		fmt.Fprint(w, `{"message":"Resource not accessible"}`)
	}))
	defer server.Close()
	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.OrganizationAlerts(context.Background(), "acme")
	if err == nil || !strings.Contains(err.Error(), "Resource not accessible") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFailedPageDoesNotAdvanceProgress(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"message":"denied"}`)
			return
		}
		fmt.Fprint(w, "[")
		for i := 0; i < 100; i++ {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprint(w, `{"number":1}`)
		}
		fmt.Fprint(w, "]")
	}))
	defer server.Close()
	c := &Client{HTTP: server.Client(), BaseURL: server.URL, OnPage: func(path string, pages, items int) {
		calls++
		if pages != 1 || items != 100 {
			t.Fatalf("incorrect successful progress: %d %d", pages, items)
		}
	}}
	if _, err := c.OrganizationAlerts(context.Background(), "acme"); err == nil {
		t.Fatal("expected page error")
	}
	if calls != 1 {
		t.Fatalf("failed page emitted progress: %d", calls)
	}
}

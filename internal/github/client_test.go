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
	repos, err := c.OrganizationRepositories(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
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

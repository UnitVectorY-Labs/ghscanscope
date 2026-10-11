package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gh "github.com/UnitVectorY-Labs/ghscanscope/internal/github"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
)

func TestSyncProgressAndSummary(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	for _, target := range []string{"", "acme/one"} {
		for _, fail := range []bool{false, true} {
			t.Run(fmt.Sprintf("repo=%s/fail=%v", target, fail), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "/code-scanning/alerts") {
						if fail {
							w.WriteHeader(http.StatusForbidden)
							fmt.Fprint(w, `{"message":"denied"}`)
							return
						}
						var a gh.Alert
						a.Number = 1
						a.Rule.ID = "test"
						a.Rule.SecuritySeverity = "high"
						a.Tool.Name = "CodeQL"
						a.Repository.ID = 1
						a.Repository.FullName = "acme/one"
						a.Repository.Name = "one"
						a.Repository.Owner.Login = "acme"
						json.NewEncoder(w).Encode([]gh.Alert{a})
					} else {
						var repo gh.Repository
						repo.ID = 1
						repo.FullName = "acme/one"
						repo.Name = "one"
						repo.Owner.Login = "acme"
						if target == "" {
							json.NewEncoder(w).Encode([]gh.Repository{repo})
						} else {
							json.NewEncoder(w).Encode(repo)
						}
					}
				}))
				defer server.Close()
				path := filepath.Join(t.TempDir(), "cache.db")
				var out, progress bytes.Buffer
				err := runSync(context.Background(), path, "acme", target, &out, &progress, &gh.Client{HTTP: server.Client(), BaseURL: server.URL})
				if fail {
					if err == nil || !strings.Contains(err.Error(), "denied") || out.Len() != 0 || !strings.Contains(progress.String(), "Sync failed") || strings.Contains(progress.String(), "Sync complete") {
						t.Fatalf("err=%v out=%q progress=%q", err, out.String(), progress.String())
					}
				} else {
					if err != nil || !strings.Contains(out.String(), "Synced 1 repositories and 1 open alerts for acme") || !strings.Contains(progress.String(), "Caching alerts  100% 1/1") || !strings.Contains(progress.String(), "Sync complete") {
						t.Fatalf("err=%v out=%q progress=%q", err, out.String(), progress.String())
					}
					s, err := store.Open(path)
					if err != nil {
						t.Fatal(err)
					}
					defer s.Close()
					alerts, err := s.OpenAlerts(context.Background())
					if err != nil || len(alerts) != 1 {
						t.Fatalf("alerts=%v err=%v", alerts, err)
					}
				}
				if strings.ContainsAny(out.String()+progress.String(), "\x1b\r") {
					t.Fatal("NO_COLOR logs contain terminal controls")
				}
				if strings.Contains(out.String(), "Caching") {
					t.Fatal("progress contaminated stdout")
				}
			})
		}
	}
}

func TestCancelledSyncFinishesProgress(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	ctx, cancel := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { cancel(); <-r.Context().Done() }))
	defer server.Close()
	var out, progress bytes.Buffer
	err := runSync(ctx, filepath.Join(t.TempDir(), "cache.db"), "acme", "", &out, &progress, &gh.Client{HTTP: server.Client(), BaseURL: server.URL})
	if err == nil || out.Len() != 0 || !strings.Contains(progress.String(), "Sync failed") {
		t.Fatalf("err=%v out=%q progress=%q", err, out.String(), progress.String())
	}
}

func TestSyncColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if err := os.Unsetenv("NO_COLOR"); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `[]`) }))
	defer server.Close()
	var out, progress bytes.Buffer
	if err := runSync(context.Background(), filepath.Join(t.TempDir(), "cache.db"), "acme", "", &out, &progress, &gh.Client{HTTP: server.Client(), BaseURL: server.URL}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\x1b[") || !strings.Contains(progress.String(), "\x1b[") {
		t.Fatalf("missing sync colors: %q %q", out.String(), progress.String())
	}
}

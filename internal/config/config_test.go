package config

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGitHubRepository(t *testing.T) {
	for _, tt := range []struct{ remote, want string }{
		{"https://github.com/UnitVectorY-Labs/ghscanscope.git", "UnitVectorY-Labs/ghscanscope"},
		{"git@github.com:UnitVectorY-Labs/ghscanscope.git", "UnitVectorY-Labs/ghscanscope"},
		{"ssh://git@github.com/UnitVectorY-Labs/ghscanscope.git", "UnitVectorY-Labs/ghscanscope"},
		{"ssh://git@github.com:22/org/repo.git", "org/repo"},
		{"https://github.com/org/repo/", "org/repo"},
		{"https://GITHUB.COM/org/repo", "org/repo"},
		{"https://gitlab.com/org/repo.git", ""},
		{"https://github.com.evil.example/org/repo.git", ""},
		{"/local/org/repo", ""},
		{"https://github.com/org", ""},
		{"https://github.com/org/repo/tree/main", ""},
		{"https://github.com/org/repo?x=1", ""},
		{"https://github.com/org/..", ""},
	} {
		t.Run(tt.remote, func(t *testing.T) {
			if got := GitHubRepository(tt.remote); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	// Keep discovery isolated from the caller's Git environment.
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func TestResolveSync(t *testing.T) {
	for _, tt := range []struct {
		name, remote, property string
		opts                   SyncOptions
		org, repo, errText     string
	}{
		{name: "directory config", property: "# organization\n\norg = UnitVectorY-Labs\n", org: "UnitVectorY-Labs"},
		{name: "checkout", remote: "git@github.com:Checkout/project.git", org: "Checkout", repo: "Checkout/project"},
		{name: "checkout beats config", remote: "https://github.com/Checkout/project.git", property: "org=Parent", org: "Checkout", repo: "Checkout/project"},
		{name: "explicit org overrides checkout", remote: "https://github.com/Checkout/project.git", opts: SyncOptions{Org: "Override", OrgSet: true}, org: "Override"},
		{name: "explicit repo overrides config and checkout", remote: "https://github.com/Checkout/project.git", property: "org=Parent", opts: SyncOptions{Repo: "Other/target", RepoSet: true}, org: "Other", repo: "Other/target"},
		{name: "explicit both", opts: SyncOptions{Org: "other", Repo: "Other/target", OrgSet: true, RepoSet: true}, org: "other", repo: "Other/target"},
		{name: "mismatch", opts: SyncOptions{Org: "One", Repo: "Two/target", OrgSet: true, RepoSet: true}, errText: "does not match"},
		{name: "invalid repo", opts: SyncOptions{Repo: "target", RepoSet: true}, errText: "OWNER/REPO"},
		{name: "no defaults", errText: "organization is required"},
		{name: "non github remote", remote: "https://gitlab.com/Other/project", property: "org=Parent", org: "Parent"},
		{name: "malformed config", property: "org Parent", errText: "expected org="},
		{name: "empty property", property: "org=", errText: "expected org="},
		{name: "override ignores malformed config", property: "bad", opts: SyncOptions{Org: "Explicit", OrgSet: true}, org: "Explicit"},
		{name: "explicit empty org", property: "org=Parent", opts: SyncOptions{OrgSet: true}, errText: "organization is required"},
		{name: "explicit empty repo suppresses checkout", remote: "https://github.com/Checkout/project", property: "org=Parent", opts: SyncOptions{RepoSet: true}, org: "Parent"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.remote != "" {
				git(t, dir, "init")
				git(t, dir, "remote", "add", "origin", tt.remote)
			}
			if tt.property != "" {
				if err := os.WriteFile(filepath.Join(dir, ".ghscanscope"), []byte(tt.property), 0600); err != nil {
					t.Fatal(err)
				}
			}
			org, repo, err := ResolveSync(context.Background(), dir, tt.opts)
			if tt.errText != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errText) {
					t.Fatalf("got error %v, want %q", err, tt.errText)
				}
				return
			}
			if err != nil || org != tt.org || repo != tt.repo {
				t.Fatalf("got (%q, %q, %v), want (%q, %q)", org, repo, err, tt.org, tt.repo)
			}
		})
	}
}

func TestSubdirectoryAndWorktree(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init")
	git(t, dir, "remote", "add", "origin", "https://github.com/Owner/project.git")
	git(t, dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "initial")
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	worktree := filepath.Join(t.TempDir(), "worktree")
	git(t, dir, "worktree", "add", "-b", "test-worktree", worktree)
	for _, cwd := range []string{nested, worktree} {
		org, repo, err := ResolveSync(context.Background(), cwd, SyncOptions{})
		if err != nil || org != "Owner" || repo != "Owner/project" {
			t.Fatalf("%s: %q %q %v", cwd, org, repo, err)
		}
		org, repo, err = ResolveFindings(context.Background(), cwd, SyncOptions{})
		if err != nil || org != "Owner" || repo != "Owner/project" {
			t.Fatalf("findings %s: %q %q %v", cwd, org, repo, err)
		}
	}
}

func TestConfigIsCurrentDirectoryOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".ghscanscope"), []byte("org=Parent"), 0600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ResolveSync(context.Background(), nested, SyncOptions{}); err == nil {
		t.Fatal("unexpected ancestor config")
	}
}

func TestDatabasePath(t *testing.T) {
	home := t.TempDir()
	homeEnv := "HOME"
	if runtime.GOOS == "windows" {
		homeEnv = "USERPROFILE"
	}
	t.Setenv(homeEnv, home)
	path, err := DatabasePath("")
	want := filepath.Join(home, ".local", "share", "ghscanscope", "ghscanscope.db")
	if err != nil || path != want {
		t.Fatalf("got %q %v, want %q", path, err, want)
	}
	if st, err := os.Stat(filepath.Dir(path)); err != nil || !st.IsDir() {
		t.Fatalf("directory missing: %v", err)
	}
	explicit := filepath.Join(t.TempDir(), "nested", "custom.db")
	if got, err := DatabasePath(explicit); err != nil || got != explicit {
		t.Fatalf("override: %q %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(home, "blocked"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := DatabasePath(filepath.Join(home, "blocked", "db")); err == nil {
		t.Fatal("expected mkdir error")
	}
	t.Setenv(homeEnv, "")
	if _, err := DatabasePath(explicit); err != nil {
		t.Fatalf("override should not require home: %v", err)
	}
}

func TestResolveFindings(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init")
	git(t, dir, "remote", "add", "origin", "https://github.com/Owner/project.git")
	for _, tt := range []struct {
		opts SyncOptions
		want string
		fail bool
	}{
		{SyncOptions{}, "Owner/project", false},
		{SyncOptions{Repo: "Other/target", RepoSet: true}, "Other/target", false},
		{SyncOptions{Org: "Other", Repo: "target", OrgSet: true, RepoSet: true}, "Other/target", false},
		{SyncOptions{Org: "Other", OrgSet: true}, "", true},
		{SyncOptions{RepoSet: true}, "", true},
		{SyncOptions{Org: "Other", Repo: "Owner/project", OrgSet: true, RepoSet: true}, "", true},
	} {
		_, repo, err := ResolveFindings(context.Background(), dir, tt.opts)
		if (err != nil) != tt.fail || repo != tt.want {
			t.Fatalf("%+v: %q %v", tt.opts, repo, err)
		}
	}
	if _, _, err := ResolveFindings(context.Background(), t.TempDir(), SyncOptions{}); err == nil {
		t.Fatal("repository required outside a checkout")
	}
}

func TestPostgresDatabaseTarget(t *testing.T) {
	for _, target := range []string{"postgres://user:password@localhost/cache?sslmode=disable", "postgresql://localhost/cache"} {
		got, err := DatabasePath(target)
		if err != nil || got != target {
			t.Fatalf("DatabasePath(%q)=%q, %v", target, got, err)
		}
	}
}

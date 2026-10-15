// Package config resolves directory-based CLI defaults.
package config

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
)

// SyncOptions records whether a value came from a flag or environment variable.
// An explicit organization selects an organization-wide sync unless a repository
// was also explicitly supplied.
type SyncOptions struct {
	Org, Repo       string
	OrgSet, RepoSet bool
}

// ResolveFindings requires a single repository, using the same checkout and
// explicit selection rules as sync. A short name is accepted with an owner.
func ResolveFindings(ctx context.Context, dir string, opts SyncOptions) (string, string, error) {
	if opts.RepoSet && opts.OrgSet && strings.TrimSpace(opts.Repo) != "" && !strings.Contains(opts.Repo, "/") {
		opts.Repo = strings.TrimSpace(opts.Org) + "/" + strings.TrimSpace(opts.Repo)
	}
	org, repo, err := ResolveSync(ctx, dir, opts)
	if err != nil {
		return "", "", fmt.Errorf("resolve findings repository (use --repo OWNER/REPO or GHSCAN_SCOPE_REPO): %w", err)
	}
	if repo == "" {
		return "", "", fmt.Errorf("repository is required: use --repo OWNER/REPO, GHSCAN_SCOPE_REPO, or run inside a GitHub checkout")
	}
	return org, repo, nil
}

func ResolveSync(ctx context.Context, dir string, opts SyncOptions) (string, string, error) {
	org, repo := strings.TrimSpace(opts.Org), strings.TrimSpace(opts.Repo)
	if !opts.OrgSet && !opts.RepoSet {
		output, err := exec.CommandContext(ctx, "git", "-C", dir, "remote", "get-url", "origin").Output()
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		if err == nil {
			repo = GitHubRepository(strings.TrimSpace(string(output)))
		}
	}
	if repo != "" {
		parts := strings.Split(repo, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("repository must be OWNER/REPO")
		}
		if !opts.OrgSet {
			org = parts[0]
		}
		if !strings.EqualFold(org, parts[0]) {
			return "", "", fmt.Errorf("repository owner %q does not match organization %q", parts[0], org)
		}
	}
	if org == "" && !opts.OrgSet {
		var err error
		org, err = directoryOrg(filepath.Join(dir, ".ghscanscope"))
		if err != nil {
			return "", "", err
		}
	}
	if org == "" {
		return "", "", fmt.Errorf("organization is required: use --org, GHSCAN_SCOPE_ORG, a GitHub origin remote, or org in .ghscanscope")
	}
	return org, repo, nil
}

// GitHubRepository accepts GitHub.com HTTPS, SSH URL, and SCP-style remotes.
func GitHubRepository(remote string) string {
	var path string
	if strings.HasPrefix(remote, "git@github.com:") {
		path = strings.TrimPrefix(remote, "git@github.com:")
	} else {
		u, err := url.Parse(remote)
		if err != nil || !strings.EqualFold(u.Hostname(), "github.com") || (u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "ssh" && u.Scheme != "git") || u.RawQuery != "" || u.Fragment != "" {
			return ""
		}
		path = strings.TrimPrefix(u.Path, "/")
	}
	path = strings.TrimSuffix(strings.TrimSuffix(path, "/"), ".git")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.ContainsAny(path, " \t\r\n?#") || parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
		return ""
	}
	return path
}

func directoryOrg(path string) (string, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	defer f.Close()
	var org string
	scanner := bufio.NewScanner(f)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") || strings.HasPrefix(text, "!") {
			continue
		}
		key, value, ok := strings.Cut(text, "=")
		if !ok || strings.TrimSpace(key) != "org" || strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("%s:%d: expected org=GITHUB_ORG", path, line)
		}
		org = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return org, nil
}

// DatabasePath preserves PostgreSQL URLs and resolves SQLite paths using the
// same home-relative default on every operating system.
func DatabasePath(path string) (string, error) {
	if store.IsPostgres(path) {
		return path, nil
	}
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home directory: %w", err)
		}
		path = filepath.Join(home, ".local", "share", "ghscanscope", "ghscanscope.db")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", fmt.Errorf("create database directory: %w", err)
	}
	return path, nil
}

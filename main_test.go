package main

import (
	"runtime"
	"strings"
	"testing"
)

func TestVersionString(t *testing.T) {
	got := versionString("1.2.3")
	if !strings.Contains(got, "ghscanscope version v1.2.3") || !strings.Contains(got, runtime.GOOS+"/"+runtime.GOARCH) {
		t.Fatalf("unexpected version: %q", got)
	}
}

func TestSyncFlagsPrecedence(t *testing.T) {
	t.Setenv("GHSCAN_SCOPE_ORG", "Environment")
	t.Setenv("GHSCAN_SCOPE_REPO", "Environment/repo")
	t.Setenv("GHSCAN_SCOPE_DB", "environment.db")
	opts, db, err := syncFlags(nil)
	if err != nil || opts.Org != "Environment" || opts.Repo != "Environment/repo" || !opts.OrgSet || !opts.RepoSet || db != "environment.db" {
		t.Fatalf("environment defaults: %+v %q %v", opts, db, err)
	}
	opts, db, err = syncFlags([]string{"--org", "Flag", "--repo", "Flag/repo", "--db", "flag.db"})
	if err != nil || opts.Org != "Flag" || opts.Repo != "Flag/repo" || !opts.OrgSet || !opts.RepoSet || db != "flag.db" {
		t.Fatalf("flag overrides: %+v %q %v", opts, db, err)
	}
	t.Setenv("GHSCAN_SCOPE_ORG", "")
	t.Setenv("GHSCAN_SCOPE_REPO", "")
	t.Setenv("GHSCAN_SCOPE_DB", "")
	opts, db, err = syncFlags(nil)
	if err != nil || opts.OrgSet || opts.RepoSet || opts.Org != "" || opts.Repo != "" || db != "" {
		t.Fatalf("directory defaults should be available: %+v %q %v", opts, db, err)
	}
	opts, _, err = syncFlags([]string{"--org=", "--repo="})
	if err != nil || !opts.OrgSet || !opts.RepoSet || opts.Org != "" || opts.Repo != "" {
		t.Fatalf("explicit empty flags: %+v %v", opts, err)
	}
}

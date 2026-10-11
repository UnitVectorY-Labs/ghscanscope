package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/app"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/config"
)

type filterValues []string

func (v *filterValues) String() string { return strings.Join(*v, ",") }
func (v *filterValues) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return fmt.Errorf("filter values must not be empty")
		}
		*v = append(*v, part)
	}
	return nil
}

func runFindings(ctx context.Context, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("findings", flag.ContinueOnError)
	org := fs.String("org", env("GHSCAN_SCOPE_ORG", ""), "repository owner (or GHSCAN_SCOPE_ORG)")
	repo := fs.String("repo", env("GHSCAN_SCOPE_REPO", ""), "OWNER/REPO, or REPO with --org (or GHSCAN_SCOPE_REPO); defaults to GitHub origin")
	db := fs.String("db", env("GHSCAN_SCOPE_DB", ""), "SQLite database path (default ~/.local/share/ghscanscope/ghscanscope.db; or GHSCAN_SCOPE_DB)")
	var levels, scanners filterValues
	fs.Var(&levels, "level", "priority: critical, high, medium, low, unknown; repeat or comma-separate")
	fs.Var(&scanners, "scanner", "exact scanner name, ignoring case; repeat or comma-separate")
	details := fs.Bool("details", false, "print full finding details even without filters")
	asJSON := fs.Bool("json", false, "emit full finding records as JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q; use flags to select a repository", fs.Arg(0))
	}
	opts := config.SyncOptions{Org: *org, Repo: *repo, OrgSet: os.Getenv("GHSCAN_SCOPE_ORG") != "", RepoSet: os.Getenv("GHSCAN_SCOPE_REPO") != ""}
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "org" {
			opts.OrgSet = true
		}
		if f.Name == "repo" {
			opts.RepoSet = true
		}
	})
	filters := app.FindingsOptions{Levels: levels, Scanners: scanners, Details: *details, JSON: *asJSON}
	if err := filters.Validate(); err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	_, fullName, err := config.ResolveFindings(ctx, dir, opts)
	if err != nil {
		return err
	}
	dbPath, err := config.DatabasePath(*db)
	if err != nil {
		return err
	}
	return app.RunFindings(ctx, dbPath, fullName, filters, out)
}

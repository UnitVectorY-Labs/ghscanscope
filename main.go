package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/app"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/config"
)

var Version = "dev"

var semverRe = regexp.MustCompile(`^\d+\.\d+\.\d+`)

func normalizedVersion(version string) string {
	if semverRe.MatchString(version) && !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

func versionString(version string) string {
	return fmt.Sprintf("ghscanscope version %s (%s, %s/%s)", normalizedVersion(version), runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  ghscanscope sync [--org ORG] [--repo OWNER/REPO] [--db PATH]")
	fmt.Fprintln(os.Stderr, "  ghscanscope findings [--org ORG] [--repo OWNER/REPO] [--db PATH] [--level LEVEL] [--scanner NAME] [--details] [--json]")
	fmt.Fprintln(os.Stderr, "  ghscanscope web [--db PATH] [--addr ADDRESS]")
	fmt.Fprintln(os.Stderr, "  ghscanscope version")
}

func syncFlags(args []string) (config.SyncOptions, string, error) {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	org := fs.String("org", env("GHSCAN_SCOPE_ORG", ""), "GitHub organization (or GHSCAN_SCOPE_ORG)")
	repo := fs.String("repo", env("GHSCAN_SCOPE_REPO", ""), "single OWNER/REPO (or GHSCAN_SCOPE_REPO)")
	db := fs.String("db", env("GHSCAN_SCOPE_DB", ""), "SQLite database path (default ~/.local/share/ghscanscope/ghscanscope.db; or GHSCAN_SCOPE_DB)")
	if err := fs.Parse(args); err != nil {
		return config.SyncOptions{}, "", err
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

	return opts, *db, nil
}

func main() {
	if Version == "dev" || Version == "" {
		if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			Version = bi.Main.Version
		}
	}
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var err error
	switch os.Args[1] {
	case "version":
		fmt.Println(versionString(Version))
		return
	case "sync":
		opts, dbFlag, parseErr := syncFlags(os.Args[2:])
		err = parseErr
		if err == nil {
			var dir, dbPath, resolvedOrg, resolvedRepo string
			dir, err = os.Getwd()
			if err == nil {
				resolvedOrg, resolvedRepo, err = config.ResolveSync(ctx, dir, opts)
			}
			if err == nil {
				dbPath, err = config.DatabasePath(dbFlag)
			}
			if err == nil {
				err = app.RunSync(ctx, dbPath, resolvedOrg, resolvedRepo, os.Stdout)
			}
		}
	case "web":
		fs := flag.NewFlagSet("web", flag.ContinueOnError)
		db := fs.String("db", env("GHSCAN_SCOPE_DB", ""), "SQLite database path (default ~/.local/share/ghscanscope/ghscanscope.db; or GHSCAN_SCOPE_DB)")
		addr := fs.String("addr", env("GHSCAN_SCOPE_ADDR", "127.0.0.1:8080"), "listen address (or GHSCAN_SCOPE_ADDR)")
		if err = fs.Parse(os.Args[2:]); err == nil {
			var dbPath string
			dbPath, err = config.DatabasePath(*db)
			if err == nil {
				err = app.RunWeb(ctx, dbPath, *addr, Version)
			}
		}
	case "findings":
		err = runFindings(ctx, os.Args[2:], os.Stdout)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

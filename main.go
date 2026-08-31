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
	fmt.Fprintln(os.Stderr, "  ghscanscope sync --org ORG [--repo OWNER/REPO] [--db PATH]")
	fmt.Fprintln(os.Stderr, "  ghscanscope web [--db PATH] [--addr ADDRESS]")
	fmt.Fprintln(os.Stderr, "  ghscanscope version")
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
		fs := flag.NewFlagSet("sync", flag.ContinueOnError)
		org := fs.String("org", env("GHSCAN_SCOPE_ORG", ""), "GitHub organization (or GHSCAN_SCOPE_ORG)")
		repo := fs.String("repo", env("GHSCAN_SCOPE_REPO", ""), "single OWNER/REPO (or GHSCAN_SCOPE_REPO)")
		db := fs.String("db", env("GHSCAN_SCOPE_DB", ".ghscanscope.db"), "SQLite database path (or GHSCAN_SCOPE_DB)")
		if err = fs.Parse(os.Args[2:]); err == nil {
			if strings.TrimSpace(*org) == "" {
				fmt.Fprintln(os.Stderr, "sync: --org is required")
				fs.Usage()
				os.Exit(2)
			}
			err = app.RunSync(ctx, *db, *org, *repo, os.Stdout)
		}
	case "web":
		fs := flag.NewFlagSet("web", flag.ContinueOnError)
		db := fs.String("db", env("GHSCAN_SCOPE_DB", ".ghscanscope.db"), "SQLite database path (or GHSCAN_SCOPE_DB)")
		addr := fs.String("addr", env("GHSCAN_SCOPE_ADDR", "127.0.0.1:8080"), "listen address (or GHSCAN_SCOPE_ADDR)")
		if err = fs.Parse(os.Args[2:]); err == nil {
			err = app.RunWeb(ctx, *db, *addr, Version)
		}
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

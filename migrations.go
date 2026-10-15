package main

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/config"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/console"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
)

func runMigrate(ctx context.Context, args []string, out io.Writer) error {
	fs := console.FlagSet("migrate")
	db := fs.String("db", env("GHSCAN_SCOPE_DB", ""), "SQLite path or postgres:// connection URL (or GHSCAN_SCOPE_DB)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: ghscanscope migrate [--db PATH_OR_URL] up|down|status [VERSION]")
		fmt.Fprintln(fs.Output(), "up applies pending upgrades; down reverts one migration. VERSION selects a target (0 removes all application tables). status prints the version. Back up before downgrading. Place flags before the action.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 || fs.NArg() > 2 {
		return fmt.Errorf("use migrate [--db PATH_OR_URL] up|down|status [VERSION]")
	}
	action := fs.Arg(0)
	if action != "up" && action != "down" && action != "status" {
		return fmt.Errorf("migration action must be up, down, or status")
	}
	var target *int64
	if fs.NArg() == 2 {
		version, err := strconv.ParseInt(fs.Arg(1), 10, 64)
		if err != nil || version < 0 || version > store.LatestVersion {
			return fmt.Errorf("migration version must be between 0 and %d", store.LatestVersion)
		}
		target = &version
	}
	if action == "status" && target != nil {
		return fmt.Errorf("status does not accept a target version")
	}
	path, err := config.DatabasePath(*db)
	if err != nil {
		return err
	}
	s, err := store.Connect(ctx, path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer s.Close()
	version, err := s.Migrate(ctx, action, target)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "Database version: %d\n", version)
	return err
}

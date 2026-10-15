package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/console"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/github"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/syncer"
	webui "github.com/UnitVectorY-Labs/ghscanscope/internal/web"
)

// RunSync keeps progress separate from callers that only need the summary.
func RunSync(ctx context.Context, dbPath, org, repo string, out io.Writer) error {
	return RunSyncWithProgress(ctx, dbPath, org, repo, out, io.Discard)
}

func RunSyncWithProgress(ctx context.Context, dbPath, org, repo string, out, progressOut io.Writer) error {
	return runSync(ctx, dbPath, org, repo, out, progressOut, nil)
}

func runSync(ctx context.Context, dbPath, org, repo string, out, progressOut io.Writer, client *github.Client) error {
	s, err := store.OpenContext(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer s.Close()
	p := console.NewProgress(progressOut)
	p.Update("Starting sync", 0, -1)
	if client == nil {
		client = github.New()
	}
	client.OnPage = func(path string, pages, items int) {
		phase := "Downloading repositories"
		if strings.Contains(path, "/code-scanning/alerts") {
			phase = "Downloading alerts"
		}
		p.Update(phase, items, -1)
	}
	result, err := (&syncer.Syncer{Store: s, GitHub: client, OnProgress: p.Update}).Sync(ctx, org, repo)
	progressErr := p.Finish(err)
	if err != nil {
		return fmt.Errorf("sync %s: %w", org, err)
	}
	if progressErr != nil {
		return fmt.Errorf("write sync progress: %w", progressErr)
	}
	return console.SyncSummary(out, result.Repositories, result.Alerts, org, result.Duration.Round(time.Millisecond).String())
}

func RunWeb(ctx context.Context, dbPath, addr, version string) error {
	s, err := store.OpenContext(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer s.Close()
	h := webui.New(s, github.New(), version)
	server := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 5 * time.Second, ErrorLog: log.New(console.Writer{Out: os.Stderr, Palette: console.New(), Code: console.Red}, "", log.LstdFlags)}
	errCh := make(chan error, 1)
	go func() {
		log.Print(console.New().Color(console.Cyan, fmt.Sprintf("ghscanscope web listening on http://%s", addr)))
		errCh <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

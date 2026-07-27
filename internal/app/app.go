package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/github"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
	"github.com/UnitVectorY-Labs/ghscanscope/internal/syncer"
	webui "github.com/UnitVectorY-Labs/ghscanscope/internal/web"
)

func RunSync(ctx context.Context, dbPath, org, repo string, out io.Writer) error {
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer s.Close()
	result, err := (&syncer.Syncer{Store: s, GitHub: github.New()}).Sync(ctx, org, repo)
	if err != nil {
		return fmt.Errorf("sync %s: %w", org, err)
	}
	fmt.Fprintf(out, "Synced %d repositories and %d open alerts for %s in %s\n", result.Repositories, result.Alerts, org, result.Duration.Round(time.Millisecond))
	return nil
}

func RunWeb(ctx context.Context, dbPath, addr string) error {
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer s.Close()
	h := webui.New(s, github.New())
	server := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 5 * time.Second}
	errCh := make(chan error, 1)
	go func() { log.Printf("ghscanscope web listening on http://%s", addr); errCh <- server.ListenAndServe() }()
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

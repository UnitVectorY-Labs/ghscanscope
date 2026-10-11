package console

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestColorPolicy(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if got := New().Label("Title"); got != "Title" {
		t.Fatalf("empty NO_COLOR must disable color: %q", got)
	}
	t.Setenv("NO_COLOR", "1")
	if got := New().Priority("Critical"); got != "Critical" {
		t.Fatalf("NO_COLOR: %q", got)
	}
	if err := os.Unsetenv("NO_COLOR"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(New().Priority("Critical"), "\x1b[1;31m") {
		t.Fatal("default color missing")
	}
	var b bytes.Buffer
	w := Writer{Out: &b, Palette: New(), Code: Cyan}
	n, err := w.Write([]byte("help\n"))
	if err != nil || n != 5 || !strings.Contains(b.String(), "\x1b[36mhelp\n") {
		t.Fatalf("writer: %d %v %q", n, err, b.String())
	}
}

func TestProgressLogCountsAndFailure(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	for _, failure := range []bool{false, true} {
		var b bytes.Buffer
		p := NewProgress(&b)
		p.Update("Downloading alerts", 0, -1)
		p.Update("Downloading alerts", 100, -1)
		p.Update("Caching alerts", 0, 2)
		p.Update("Caching alerts", 1, 2)
		if !failure {
			p.Update("Caching alerts", 2, 2)
		}
		var err error
		if failure {
			err = errors.New("cancelled")
		}
		if err := p.Finish(err); err != nil {
			t.Fatal(err)
		}
		text := b.String()
		if strings.ContainsAny(text, "\r\x1b") || !strings.Contains(text, "100 received") || !strings.Contains(text, "50% 1/2") {
			t.Fatalf("progress: %q", text)
		}
		if failure {
			if !strings.Contains(text, "Sync failed") || strings.Contains(text, "Sync complete") || strings.Contains(text, "100% 2/2") {
				t.Fatal(text)
			}
		} else if !strings.Contains(text, "100% 2/2") || !strings.Contains(text, "Sync complete") {
			t.Fatal(text)
		}
	}
}

func TestEmptyProgressAndThrottling(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	var b bytes.Buffer
	p := NewProgress(&b)
	p.Update("Empty", 0, 0)
	for i := 0; i <= 1000; i++ {
		p.Update("Large", i, 1000)
	}
	if err := p.Finish(nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "100% 0/0") || !strings.Contains(b.String(), "100% 1000/1000") || strings.Count(b.String(), "Large") > 21 {
		t.Fatalf("incorrect empty or throttled progress: %s", b.String())
	}
}

// Observe animation from the ticker without repeatedly sending fake progress.
type observedWriter struct {
	bytes.Buffer
	writes chan struct{}
}

func (w *observedWriter) Write(data []byte) (int, error) {
	n, err := w.Buffer.Write(data)
	select {
	case w.writes <- struct{}{}:
	default:
	}
	return n, err
}
func (w *observedWriter) WriteString(value string) (int, error) { return w.Write([]byte(value)) }

func TestTerminalProgressAnimationStops(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	w := &observedWriter{writes: make(chan struct{}, 10)}
	p := newProgress(w, true)
	p.Update("Downloading repositories", 0, -1)
	// The first write is the phase change; the second comes from animation.
	for i := 0; i < 2; i++ {
		select {
		case <-w.writes:
		case <-time.After(2 * time.Second):
			p.Finish(errors.New("timeout"))
			t.Fatal("terminal animation did not advance")
		}
	}
	p.Update("Caching repositories", 2, 2)
	if err := p.Finish(nil); err != nil {
		t.Fatal(err)
	}
	text := w.String()
	if !strings.Contains(text, "\r") || !strings.Contains(text, "Sync complete\n") || !strings.Contains(text, "100% 2/2") || strings.Contains(text, "\x1b") {
		t.Fatalf("terminal output: %q", text)
	}
	select {
	case <-p.stopped:
	default:
		t.Fatal("animation goroutine still running")
	}
}

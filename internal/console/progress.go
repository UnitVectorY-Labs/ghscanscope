package console

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
)

// Progress shows counted work when a total is known, or activity and downloaded
// item counts when GitHub pagination has not yet revealed a total.
type Progress struct {
	out         io.Writer
	palette     Palette
	terminal    bool
	mu          sync.Mutex
	phase       string
	done, total int
	frame       int
	lastPhase   string
	lastBucket  int
	width       int
	stop        chan struct{}
	stopped     chan struct{}
	err         error
}

func NewProgress(out io.Writer) *Progress {
	terminal := false
	if f, ok := out.(interface{ Fd() uintptr }); ok {
		terminal = isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return newProgress(out, terminal)
}

func newProgress(out io.Writer, terminal bool) *Progress {
	p := &Progress{out: out, palette: New(), terminal: terminal, lastBucket: -1, stop: make(chan struct{}), stopped: make(chan struct{})}
	if terminal {
		go func() {
			defer close(p.stopped)
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-p.stop:
					return
				case <-ticker.C:
					p.mu.Lock()
					if p.phase != "" {
						p.frame++
						p.render()
					}
					p.mu.Unlock()
				}
			}
		}()
	} else {
		close(p.stopped)
	}
	return p
}

// Update uses total=-1 for unknown totals. Known empty phases use total=0.
func (p *Progress) Update(phase string, done, total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.phase, p.done, p.total = phase, done, total
	bucket := 0
	if total > 0 {
		bucket = done * 20 / total
	} else if total < 0 {
		bucket = done
	}
	// Terminal frames redraw on a ticker. Logs show phase changes and at most
	// twenty counted updates per phase, without carriage returns or animation.
	if phase != p.lastPhase || done == total || (!p.terminal && bucket != p.lastBucket) {
		p.render()
		p.lastPhase, p.lastBucket = phase, bucket
	}
}

func (p *Progress) render() {
	const barWidth = 20
	bar := strings.Repeat("░", barWidth)
	count := "waiting"
	if p.total >= 0 {
		percent := 100
		if p.total > 0 {
			percent = min(100, max(0, p.done*100/p.total))
		}
		filled := percent * barWidth / 100
		bar = p.palette.Color(Cyan, strings.Repeat("━", filled)) + p.palette.Color(Dim, strings.Repeat("░", barWidth-filled))
		count = fmt.Sprintf("%3d%% %d/%d", percent, p.done, p.total)
	} else {
		if p.terminal {
			position := p.frame % (barWidth - 2)
			bar = p.palette.Color(Dim, strings.Repeat("░", position)) + p.palette.Color(Cyan, "━━━") + p.palette.Color(Dim, strings.Repeat("░", barWidth-position-3))
		} else {
			bar = p.palette.Color(Dim, bar)
		}
		if p.done > 0 {
			count = fmt.Sprintf("%d received", p.done)
		}
	}
	line := fmt.Sprintf("  [%s] %s  %s", bar, p.palette.Label(p.phase), count)
	if p.terminal {
		// Padding erases a longer previous frame without ANSI cursor commands.
		width := barWidth + 7 + len([]rune(p.phase)) + len([]rune(count))
		padding := max(0, p.width-width)
		p.width = width
		p.write("\r" + line + strings.Repeat(" ", padding))
	} else {
		p.write(line + "\n")
	}
}

func (p *Progress) write(text string) {
	if p.err != nil {
		return
	}
	_, p.err = io.WriteString(p.out, text)
}

// Finish stops animation and leaves a complete line even on failure/cancellation.
// It must be called exactly once after all progress producers have returned.
func (p *Progress) Finish(err error) error {
	close(p.stop)
	<-p.stopped
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.terminal {
		p.write("\r" + strings.Repeat(" ", p.width) + "\r")
	}
	message := p.palette.Color(Green, "Sync complete")
	if err != nil {
		message = p.palette.Color(Red, "Sync failed")
	}
	p.write("  " + message + "\n")
	return p.err
}

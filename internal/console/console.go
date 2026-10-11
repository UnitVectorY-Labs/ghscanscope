// Package console provides shared presentation for human-readable CLI output.
package console

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	Bold   = "1;34"
	Cyan   = "36"
	Green  = "32"
	Yellow = "33"
	Red    = "31"
	Dim    = "2"
)

type Palette struct{ Enabled bool }

// NO_COLOR presence disables color, including an explicitly empty value.
// JSON callers bypass this package so machine-readable output stays unchanged.
func New() Palette {
	_, disabled := os.LookupEnv("NO_COLOR")
	return Palette{Enabled: !disabled}
}

func (p Palette) Color(code, value string) string {
	if !p.Enabled || value == "" {
		return value
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
}
func (p Palette) Label(value string) string { return p.Color(Bold, value) }
func (p Palette) Priority(value string) string {
	code := Dim
	switch strings.ToLower(value) {
	case "critical":
		code = "1;31"
	case "high":
		code = Red
	case "medium":
		code = Yellow
	case "low":
		code = Cyan
	}
	return p.Color(code, value)
}

// Writer colors flag help and parsing diagnostics without changing their text.
type Writer struct {
	Out     io.Writer
	Palette Palette
	Code    string
}

func (w Writer) Write(data []byte) (int, error) {
	text := w.Palette.Color(w.Code, string(data))
	n, err := io.WriteString(w.Out, text)
	if err != nil {
		return 0, err
	}
	if n != len(text) {
		return 0, io.ErrShortWrite
	}
	return len(data), nil
}
func FlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(Writer{Out: os.Stderr, Palette: New(), Code: Cyan})
	return fs
}

func SyncSummary(out io.Writer, repositories, alerts int, org, duration string) error {
	p := New()
	_, err := fmt.Fprintf(out, "%s %s repositories and %s open alerts for %s in %s\n",
		p.Color(Green, "Synced"), p.Color(Bold, fmt.Sprint(repositories)), p.Color(Bold, fmt.Sprint(alerts)), p.Color(Cyan, org), p.Color(Dim, duration))
	return err
}

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/UnitVectorY-Labs/ghscanscope/internal/store"
)

type FindingsOptions struct {
	Levels, Scanners []string
	Details, JSON    bool
}

func (o FindingsOptions) Validate() error {
	for _, level := range o.Levels {
		switch strings.ToLower(level) {
		case "critical", "high", "medium", "low", "unknown":
		default:
			return fmt.Errorf("invalid level %q: use critical, high, medium, low, or unknown", level)
		}
	}
	return nil
}

type Finding struct {
	Number           int64     `json:"number"`
	Priority         string    `json:"priority"`
	ReportedSeverity string    `json:"reported_severity"`
	SeveritySource   string    `json:"severity_source"`
	Scanner          string    `json:"scanner"`
	ScannerVersion   string    `json:"scanner_version"`
	ScannerGUID      string    `json:"scanner_guid"`
	RuleID           string    `json:"rule_id"`
	RuleName         string    `json:"rule_name"`
	RuleDescription  string    `json:"rule_description"`
	Tags             []string  `json:"tags"`
	Message          string    `json:"message"`
	Path             string    `json:"path"`
	StartLine        int       `json:"start_line"`
	EndLine          int       `json:"end_line"`
	StartColumn      int       `json:"start_column"`
	EndColumn        int       `json:"end_column"`
	Ref              string    `json:"ref"`
	CommitSHA        string    `json:"commit_sha"`
	AnalysisKey      string    `json:"analysis_key"`
	Environment      string    `json:"environment"`
	Category         string    `json:"category"`
	URL              string    `json:"url"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type FindingsReport struct {
	Repository         string         `json:"repository"`
	RepositoryURL      string         `json:"repository_url"`
	RepositoryCachedAt time.Time      `json:"repository_cached_at"`
	TotalOpen          int            `json:"total_open"`
	Matching           int            `json:"matching"`
	ByLevel            map[string]int `json:"by_level"`
	ByScanner          map[string]int `json:"by_scanner"`
	Findings           []Finding      `json:"findings"`
}

// RunFindings reads the same current open alerts as the dashboard. It does not
// fetch GitHub data or infer scan coverage from an empty result.
func RunFindings(ctx context.Context, dbPath, repository string, opts FindingsOptions, out io.Writer) error {
	if err := opts.Validate(); err != nil {
		return err
	}
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("cached database unavailable: %w; run ghscanscope sync first", err)
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer s.Close()
	repos, err := s.Repositories(ctx)
	if err != nil {
		return err
	}
	report := FindingsReport{ByLevel: map[string]int{"Critical": 0, "High": 0, "Medium": 0, "Low": 0, "Unknown": 0}, ByScanner: map[string]int{}, Findings: []Finding{}}
	for _, r := range repos {
		if strings.EqualFold(r.FullName, repository) {
			report.Repository, report.RepositoryURL, report.RepositoryCachedAt = r.FullName, r.URL, r.UpdatedAt
			break
		}
	}
	if report.Repository == "" {
		return fmt.Errorf("repository %s is not cached; run ghscanscope sync --repo %s first", repository, repository)
	}
	alerts, err := s.RepositoryAlerts(ctx, repository)
	if err != nil {
		return err
	}
	report.TotalOpen = len(alerts)
	rank := map[string]int{"Critical": 0, "High": 1, "Medium": 2, "Low": 3, "Unknown": 4}
	sort.SliceStable(alerts, func(i, j int) bool {
		if rank[alerts[i].Priority] != rank[alerts[j].Priority] {
			return rank[alerts[i].Priority] < rank[alerts[j].Priority]
		}
		return alerts[i].GitHubID < alerts[j].GitHubID
	})
	for _, a := range alerts {
		if !includes(opts.Levels, a.Priority) || !includes(opts.Scanners, a.Tool) {
			continue
		}
		report.ByLevel[a.Priority]++
		report.ByScanner[a.Tool]++
		tags := a.Tags
		if tags == nil {
			tags = []string{}
		}
		report.Findings = append(report.Findings, Finding{
			Number: a.GitHubID, Priority: a.Priority, ReportedSeverity: a.ReportedSeverity, SeveritySource: a.SeveritySource,
			Scanner: a.Tool, ScannerVersion: a.ToolVersion, ScannerGUID: a.ToolGUID, RuleID: a.RuleID, RuleName: a.RuleName,
			RuleDescription: a.RuleDescription, Tags: tags, Message: a.Message, Path: a.Path, StartLine: a.StartLine, EndLine: a.EndLine,
			StartColumn: a.StartColumn, EndColumn: a.EndColumn, Ref: a.Ref, CommitSHA: a.CommitSHA, AnalysisKey: a.AnalysisKey,
			Environment: a.Environment, Category: a.Category, URL: a.URL, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
		})
	}
	report.Matching = len(report.Findings)
	if opts.JSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}
	return writeFindings(out, report, opts.Details || len(opts.Levels) > 0 || len(opts.Scanners) > 0)
}

func includes(values []string, value string) bool {
	if len(values) == 0 {
		return true
	}
	for _, v := range values {
		if strings.EqualFold(v, value) {
			return true
		}
	}
	return false
}

func writeFindings(out io.Writer, r FindingsReport, details bool) error {
	// Buffer text so output errors are returned consistently with JSON errors.
	var b strings.Builder
	fmt.Fprintf(&b, "Repository: %s\nRepository cached at: %s\nOpen findings: %d matching / %d total\n", r.Repository, r.RepositoryCachedAt.UTC().Format(time.RFC3339Nano), r.Matching, r.TotalOpen)
	fmt.Fprintf(&b, "Levels: critical=%d high=%d medium=%d low=%d unknown=%d\n", r.ByLevel["Critical"], r.ByLevel["High"], r.ByLevel["Medium"], r.ByLevel["Low"], r.ByLevel["Unknown"])
	scanners := make([]string, 0, len(r.ByScanner))
	for name := range r.ByScanner {
		scanners = append(scanners, name)
	}
	sort.Strings(scanners)
	for _, name := range scanners {
		fmt.Fprintf(&b, "Scanner: %s (%d)\n", name, r.ByScanner[name])
	}
	if r.Matching == 0 {
		fmt.Fprintln(&b, "No matching cached open findings. This does not establish scan coverage.")
	}
	if details {
		for _, f := range r.Findings {
			fmt.Fprintf(&b, "\n[%s] #%d %s: %s\n", f.Priority, f.Number, f.Scanner, f.RuleID)
			fields := []struct{ name, value string }{
				{"Rule name", f.RuleName}, {"Description", f.RuleDescription}, {"Message", f.Message}, {"Path", f.Path},
				{"Reported severity", f.ReportedSeverity}, {"Severity source", f.SeveritySource}, {"Tags", strings.Join(f.Tags, ", ")},
				{"Ref", f.Ref}, {"Commit", f.CommitSHA}, {"Analysis key", f.AnalysisKey}, {"Environment", f.Environment}, {"Category", f.Category},
				{"Scanner version", f.ScannerVersion}, {"Scanner GUID", f.ScannerGUID}, {"URL", f.URL},
			}
			for _, field := range fields {
				if field.value != "" {
					fmt.Fprintf(&b, "  %s: %s\n", field.name, strings.ReplaceAll(field.value, "\n", "\n    "))
				}
			}
			if f.Path == "" {
				fmt.Fprintln(&b, "  Location: not reported")
			}
			fmt.Fprintf(&b, "  Lines: %d-%d; columns: %d-%d\n  Created: %s\n  Updated: %s\n", f.StartLine, f.EndLine, f.StartColumn, f.EndColumn, f.CreatedAt.UTC().Format(time.RFC3339Nano), f.UpdatedAt.UTC().Format(time.RFC3339Nano))
		}
	}
	_, err := io.WriteString(out, b.String())
	return err
}

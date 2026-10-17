package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Markdown consumes only the same display model as the HTML template. Never
// serialize Page: it also contains filter options and other non-content data.
func renderMarkdown(r *http.Request, page Page) string {
	m := markdownWriter{request: r}
	fmt.Fprintf(&m.Builder, "---\ntitle: %q\nurl: %q\n---\n\n# %s\n\n", page.Title, m.absolute(r.URL.RequestURI()), markdownText(page.Title))
	if page.ErrorMessage != "" {
		m.paragraph(page.ErrorMessage)
		m.related(page)
		return m.String()
	}
	m.paragraph(page.Notice)
	if page.Stats.LastSync != nil {
		m.field("Last synced", markdownTime(*page.Stats.LastSync))
	}
	if len(page.Chips) > 0 {
		m.WriteString("Filters: ")
		for i, chip := range page.Chips {
			if i > 0 {
				m.WriteString("; ")
			}
			m.WriteString(markdownText(chip.Label))
		}
		m.WriteString("\n\n")
	}
	if page.Stats.Totals != nil && page.View != "alert" {
		m.WriteString("Counts: matching / total before filters.\n\n")
		total := *page.Stats.Totals
		m.field("Open occurrences", countPair(page.Stats.Open, total.Open))
		if page.View == "repositories" {
			m.field("Repositories", countPair(len(page.Repositories), total.TotalRepositories))
		} else {
			m.field("Affected repositories", countPair(page.Stats.AffectedRepositories, total.AffectedRepositories))
		}
		m.priorities(page.Stats, total)
		m.WriteByte('\n')
	}
	switch page.View {
	case "alerts":
		m.field("Issue groups", countPair(len(page.Groups), page.TotalGroups))
		for _, group := range page.Groups {
			fmt.Fprintf(&m.Builder, "\n## %s\n\n", m.link(group.Name, group.URL))
			m.field("Scanner", group.Tool)
			m.field("Rule", group.RuleID)
			m.paragraph(group.Description)
			m.field("Affected repositories", countPair(group.RepositoryCount, group.Totals.AffectedRepositories))
			m.field("Open occurrences", countPair(group.AlertCount, group.Totals.Open))
			m.priorities(groupStats(group), group.Totals)
			m.field("Alert updated", markdownTime(group.UpdatedAt))
		}
		if len(page.Groups) == 0 {
			m.paragraph("No issue groups match these filters.")
		}
	case "repositories":
		m.paragraph("No open alerts does not establish scanning coverage.")
		m.repositories(page, page.Repositories)
	case "repository":
		repo := page.Repository
		m.paragraph(repo.Description)
		m.field("Visibility", repo.Visibility)
		m.field("Language", repo.Language)
		m.occurrences(page)
	case "rule":
		m.field("Scanner", page.RuleTool)
		m.field("Rule", page.RuleID)
		m.paragraph(page.RuleDescription)
		if page.RuleDescription == "" {
			m.paragraph("The scanner did not provide a rule description. Read individual finding messages for the reported evidence.")
		}
		m.field("Example finding in this scope", page.RuleMessage)
		m.field("Tags", strings.Join(page.RuleTags, ", "))
		m.paragraph("Occurrences share a rule; each location still needs individual review.")
		m.WriteString("## Affected repositories\n\n")
		m.repositories(page, page.RuleRepositories)
		m.occurrences(page)
	case "alert":
		a := *page.Alert
		m.field("Status", "Open")
		m.field("GitHub alert", strconv.FormatInt(a.GitHubID, 10))
		m.field("Priority", a.Priority)
		m.field("Scanner", a.Tool)
		m.field("Scanner version", a.ToolVersion)
		m.field("Rule", a.RuleID)
		m.WriteString("\n## Finding message\n\n")
		if a.Message == "" {
			m.paragraph("The scanner did not provide a finding message.")
		} else {
			m.paragraph(a.Message)
		}
		fmt.Fprintf(&m.Builder, "## Affected location\n\n%s\n\n", m.link(lineRange(a), codeURL(a)))
		m.field("Ref", a.Ref)
		m.field("Commit", a.CommitSHA)
		m.WriteString("\n## About this issue\n\n")
		if a.RuleDescription == "" {
			m.paragraph("Rule description not provided by the scanner.")
		} else {
			m.paragraph(a.RuleDescription)
		}
		m.field("Tags", strings.Join(a.Tags, ", "))
		m.field("First reported", markdownTime(a.CreatedAt))
		m.field("Alert updated", markdownTime(a.UpdatedAt))
		m.WriteString("\n## Scan details and priority source\n\n")
		m.field("Reported severity", a.ReportedSeverity)
		if a.ReportedSeverity != "" {
			m.field("Severity source", a.SeveritySource)
		}
		m.field("Analysis key", a.AnalysisKey)
		m.field("Environment", a.Environment)
		m.field("Category", a.Category)
	}
	m.related(page)
	return m.String()
}

type markdownWriter struct {
	strings.Builder
	request *http.Request
}

func markdownText(value string) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
	// Scanner text is displayed as text in HTML too. Escape syntax so it cannot
	// inject HTML, headings, images, or links into the alternate representation.
	value = strings.NewReplacer("&", "&amp;", "\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "<", "&lt;", ">", "&gt;", "#", "\\#", "!", "\\!", "|", "\\|").Replace(value)
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		// Escape list markers and setext headings only at the start of a line;
		// keep punctuation in timestamps, paths, and prose readable.
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "+") || strings.HasPrefix(line, "=") {
			lines[i] = "\\" + line
			continue
		}
		j := 0
		for j < len(line) && line[j] >= '0' && line[j] <= '9' {
			j++
		}
		if j > 0 && j < len(line) && (line[j] == '.' || line[j] == ')') {
			lines[i] = line[:j] + "\\" + line[j:]
		}
	}
	return strings.Join(lines, "  \n")
}

func (m *markdownWriter) field(label, value string) {
	if value != "" {
		fmt.Fprintf(&m.Builder, "%s: %s  \n", label, markdownText(value))
	}
}

func (m *markdownWriter) paragraph(value string) {
	if value != "" {
		fmt.Fprintf(&m.Builder, "%s\n\n", markdownText(value))
	}
}

func (m *markdownWriter) absolute(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || raw == "" {
		return ""
	}
	scheme := "http"
	if m.request.TLS != nil {
		scheme = "https"
	}
	// Do not trust forwarded headers supplied by arbitrary clients. A reverse
	// proxy should preserve Host. TLS termination requires a configured public
	// origin in a future extension to produce absolute HTTPS links correctly.
	base := &url.URL{Scheme: scheme, Host: m.request.Host}
	u = base.ResolveReference(u)
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return strings.NewReplacer("<", "%3C", ">", "%3E", "\"", "%22", "\\", "%5C").Replace(u.String())
}

func (m *markdownWriter) link(label, raw string) string {
	if target := m.absolute(raw); target != "" {
		return "[" + markdownText(label) + "](<" + target + ">)"
	}
	return markdownText(label)
}

func countPair(shown, total int) string { return fmt.Sprintf("%d / %d", shown, total) }

func markdownTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func (m *markdownWriter) priorities(shown, total Stats) {
	m.field("Priorities", fmt.Sprintf("Critical %s; High %s; Medium %s; Low %s; Unknown %s", countPair(shown.Critical, total.Critical), countPair(shown.High, total.High), countPair(shown.Medium, total.Medium), countPair(shown.Low, total.Low), countPair(shown.Other, total.Other)))
}

func (m *markdownWriter) repositories(page Page, repos []RepoView) {
	for _, repo := range repos {
		fmt.Fprintf(&m.Builder, "### %s\n\n", m.link(repo.FullName, pageLink(page, "/repositories/"+strconv.FormatInt(repo.ID, 10), nil)))
		if page.View == "repositories" {
			m.field("Organization", repo.Org)
			m.field("Visibility", repo.Visibility)
			m.field("Language", repo.Language)
			m.paragraph(repo.Description)
		}
		m.field("Open occurrences", countPair(repo.AlertCount, repo.Totals.Open))
		m.priorities(Stats{Critical: repo.Critical, High: repo.High, Medium: repo.Medium, Low: repo.Low, Other: repo.Other}, repo.Totals)
		m.WriteByte('\n')
	}
	if len(repos) == 0 {
		m.paragraph("No repositories match these filters.")
	}
}

func (m *markdownWriter) occurrences(page Page) {
	m.WriteString("\n## Open findings\n\n")
	for _, a := range page.Alerts {
		fmt.Fprintf(&m.Builder, "### %s\n\n", m.link(findingName(a), "/alerts/"+strconv.FormatInt(a.ID, 10)+"?from="+url.QueryEscape(page.CurrentURL)))
		if page.View == "rule" {
			fmt.Fprintf(&m.Builder, "Repository: %s  \n", m.link(a.Repository, pageLink(page, "/repositories/"+strconv.FormatInt(a.RepositoryID, 10), nil)))
			m.field("GitHub alert", strconv.FormatInt(a.GitHubID, 10))
		}
		m.field("Priority", a.Priority)
		m.field("Scanner", a.Tool)
		m.field("Message", a.Message)
		m.field("Location", lineRange(a))
		m.field("Ref", a.Ref)
		m.field("Alert updated", markdownTime(a.UpdatedAt))
		m.WriteByte('\n')
	}
	if len(page.Alerts) == 0 {
		m.paragraph("No open findings match these filters.")
	}
}

func (m *markdownWriter) related(page Page) {
	m.WriteString("\n## Related\n\n")
	seen := map[string]bool{}
	add := func(label, target string) {
		if target != "" && !seen[m.absolute(target)] {
			seen[m.absolute(target)] = true
			fmt.Fprintf(&m.Builder, "- %s\n", m.link(label, target))
		}
	}
	if page.View == "alert" || page.View == "repository" || page.View == "rule" {
		add("Back to "+page.BackLabel, page.BackURL)
	}
	if page.Alert != nil {
		a := page.Alert
		add(a.Repository, pageLink(page, "/repositories/"+strconv.FormatInt(a.RepositoryID, 10), nil))
		add("Explore this issue across repositories", pageLink(page, "/rules", url.Values{"tool": {a.Tool}, "rule": {a.RuleID}}))
		add("Alert on GitHub", a.URL)
		if a.Path != "" {
			add("Source at reported revision", codeURL(*a))
		}
	}
	if page.Repository != nil {
		add("Repository on GitHub", page.Repository.URL)
	}
	add("All results before filters", page.ResetURL)
	add("Issues across repositories", "/alerts")
	add("Repositories", "/repositories")
}

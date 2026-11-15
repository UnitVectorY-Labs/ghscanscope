package web

import (
	"sort"
	"strconv"
	"strings"
)

// findingsCommand only exports drill-down views. Repository metadata filters
// select repositories here; search is represented by the matching alert numbers.
func findingsCommand(page Page) string {
	var repositories []string
	alerts := page.Alerts
	switch page.View {
	case "alert":
		return "ghscanscope findings --repo " + shellArg(page.Alert.Repository) + " --number " + strconv.FormatInt(page.Alert.GitHubID, 10) + " --verbose"
	case "repository":
		repositories = []string{page.Repository.FullName}
	case "rule":
		for _, repo := range page.RuleRepositories {
			repositories = append(repositories, repo.FullName)
		}
	default:
		return ""
	}
	// Empty filtered views have no selection to export.
	if len(alerts) == 0 && len(page.Filters) > 0 {
		return ""
	}
	sort.Strings(repositories)
	var commands []string
	for _, repo := range repositories {
		command := "ghscanscope findings --repo " + shellArg(repo)
		// The CLI's repeated string filters split commas, so unusual scanner/rule
		// names containing commas also require an exact alert-number selection.
		exact := len(page.Filters["q"]) > 0 || strings.Contains(page.RuleTool+page.RuleID, ",")
		for _, key := range []string{"tool", "rule"} {
			for _, value := range page.Filters[key] {
				exact = exact || strings.Contains(value, ",")
			}
		}
		if exact {
			var numbers []int64
			for _, a := range alerts {
				if a.Repository == repo {
					numbers = append(numbers, a.GitHubID)
				}
			}
			if len(numbers) == 0 {
				continue
			}
			sort.Slice(numbers, func(i, j int) bool { return numbers[i] < numbers[j] })
			var values []string
			for _, n := range numbers {
				values = append(values, strconv.FormatInt(n, 10))
			}
			command += " --number " + strings.Join(values, ",")
		} else {
			if page.View == "rule" {
				command += " --scanner " + shellArg(page.RuleTool) + " --rule " + shellArg(page.RuleID)
			}
			for _, filter := range []struct{ key, flag string }{{"priority", "level"}, {"tool", "scanner"}, {"rule", "rule"}} {
				for _, value := range page.Filters[filter.key] {
					command += " --" + filter.flag + " " + shellArg(value)
				}
			}
			if len(page.Filters["kev"]) > 0 {
				command += " --kev"
			}
		}
		commands = append(commands, command+" --verbose")
	}
	return strings.Join(commands, "\n")
}

// shellArg quotes POSIX shell arguments, including scanner-controlled text.
func shellArg(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_./:@%+=,-", r)
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

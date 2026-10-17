package web

import (
	"mime"
	"net/http"
	"strconv"
	"strings"
)

type mediaPreference struct {
	quality     float64
	specificity int
}

// prefersMarkdown applies the most specific matching media range to each
// representation (RFC 9110 section 12.5.1). Wildcards alone never opt into
// Markdown. Equal explicit preferences retain the existing HTML default.
// If neither representation is acceptable we retain HTML, as RFC 9110 permits
// a server to disregard Accept instead of returning 406.
func prefersMarkdown(r *http.Request) bool {
	html := mediaPreference{specificity: -1}
	markdown := mediaPreference{specificity: -1}
	explicitMarkdown := false
	for _, item := range splitMediaRanges(strings.Join(r.Header.Values("Accept"), ",")) {
		media, params, err := mime.ParseMediaType(strings.TrimSpace(item))
		if err != nil {
			continue
		}
		quality := 1.0
		if q, ok := params["q"]; ok {
			if !validQuality(q) {
				continue
			}
			quality, _ = strconv.ParseFloat(q, 64)
		}
		delete(params, "q")
		// Both renderers offer only a UTF-8 representation, without variants.
		parameterSpecificity := 0
		if charset, ok := params["charset"]; ok && strings.EqualFold(charset, "utf-8") {
			delete(params, "charset")
			parameterSpecificity = 1
		}
		if len(params) != 0 {
			continue
		}
		specificity := 4 + parameterSpecificity
		if media == "text/*" {
			specificity = 2 + parameterSpecificity
		} else if media == "*/*" {
			specificity = parameterSpecificity
		}
		update := func(p *mediaPreference) {
			if specificity > p.specificity || (specificity == p.specificity && quality > p.quality) {
				*p = mediaPreference{quality, specificity}
			}
		}
		if media == "text/html" || media == "text/*" || media == "*/*" {
			update(&html)
		}
		if media == "text/markdown" || media == "text/*" || media == "*/*" {
			update(&markdown)
			if media == "text/markdown" && quality > 0 {
				explicitMarkdown = true
			}
		}
	}
	return explicitMarkdown && markdown.quality > 0 &&
		(markdown.quality > html.quality || (markdown.quality == html.quality && html.specificity < 4))
}

// Commas inside quoted media parameters are not list delimiters.
func splitMediaRanges(value string) []string {
	var parts []string
	start := 0
	quoted, escaped := false, false
	for i, c := range value {
		if escaped {
			escaped = false
		} else if quoted && c == '\\' {
			escaped = true
		} else if c == '"' {
			quoted = !quoted
		} else if c == ',' && !quoted {
			parts = append(parts, value[start:i])
			start = i + 1
		}
	}
	return append(parts, value[start:])
}

func validQuality(q string) bool {
	if len(q) == 0 || (q[0] != '0' && q[0] != '1') {
		return false
	}
	if len(q) == 1 {
		return true
	}
	if q[1] != '.' || len(q) > 5 {
		return false
	}
	for _, digit := range q[2:] {
		if digit < '0' || digit > '9' || (q[0] == '1' && digit != '0') {
			return false
		}
	}
	return true
}

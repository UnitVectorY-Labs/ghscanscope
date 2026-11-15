package web

import (
	"html"
	"strconv"
	"strings"
	"testing"
)

func TestDrillDownCommands(t *testing.T) {
	f := newFixture(t)
	repo := "/repositories/" + strconv.FormatInt(f.repositoryID, 10)
	for _, tc := range []struct{ path, want string }{
		{repo, "ghscanscope findings --repo acme/one --verbose"},
		{repo + "?priority=High&tool=CodeQL", "ghscanscope findings --repo acme/one --level High --scanner CodeQL --verbose"},
		{repo + "?q=untrusted", "ghscanscope findings --repo acme/one --number 7 --verbose"},
		{"/rules?tool=CodeQL&rule=go/xss", "ghscanscope findings --repo acme/one --scanner CodeQL --rule go/xss --verbose\nghscanscope findings --repo acme/two --scanner CodeQL --rule go/xss --verbose"},
		{"/rules?tool=CodeQL&rule=go/xss&repo=acme/one", "ghscanscope findings --repo acme/one --scanner CodeQL --rule go/xss --verbose"},
		{"/alerts/" + strconv.FormatInt(f.alertID, 10), "ghscanscope findings --repo acme/one --number 7 --verbose"},
	} {
		body := html.UnescapeString(get(t, f.handler, tc.path))
		if !strings.Contains(body, "<code class=\"mono\">"+tc.want+"</code>") {
			t.Errorf("%s missing command %q", tc.path, tc.want)
		}
		if !strings.Contains(body, "Copy findings command") {
			t.Errorf("%s missing copy button", tc.path)
		}
	}
	for _, path := range []string{"/alerts", "/repositories", repo + "?q=nonexistent", "/rules?tool=CodeQL&rule=go/xss&q=nonexistent"} {
		if strings.Contains(get(t, f.handler, path), "<section class=\"panel findings-command\"") {
			t.Errorf("unexpected command on %s", path)
		}
	}
}

func TestShellArg(t *testing.T) {
	for _, tc := range []struct{ value, want string }{
		{"CodeQL", "CodeQL"},
		{"Semgrep OSS", "'Semgrep OSS'"},
		{"a'$(touch /tmp/unsafe)", "'a'\"'\"'$(touch /tmp/unsafe)'"},
		{"", "''"},
	} {
		if got := shellArg(tc.value); got != tc.want {
			t.Errorf("shellArg(%q)=%q, want %q", tc.value, got, tc.want)
		}
	}
}

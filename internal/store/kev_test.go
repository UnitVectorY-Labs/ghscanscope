package store

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestKEVMatching(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, err = s.DB.Exec("INSERT INTO kev(cve,payload) VALUES (?,?)", "CVE-2023-4863", `{"cveID":"CVE-2023-4863","vulnerabilityName":"WebP"}`)
	if err != nil {
		t.Fatal(err)
	}
	alerts := []Alert{
		{RuleID: "cve-2023-4863", Message: "Repeated CVE-2023-4863"},
		{Message: "CVE-2023-48630 and CVE-2023-9999"},
	}
	if err := s.annotateKEV(context.Background(), alerts); err != nil {
		t.Fatal(err)
	}
	if len(alerts[0].KEVs) != 1 || len(alerts[1].KEVs) != 0 {
		t.Fatalf("unexpected matches: %+v", alerts)
	}
}

// Serve fixtures through the production fetch path without contacting CISA.
type kevTransport func(*http.Request) (*http.Response, error)

func (f kevTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestIngestKEVStrictJSONAndAtomicity(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	body := `{"count":1,"vulnerabilities":[{"cveID":"cve-2023-4863","vendorProject":"Vendor","vulnerabilityName":"WebP"}]}`
	http.DefaultTransport = kevTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	if n, err := s.IngestKEV(context.Background()); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	for _, invalid := range []string{
		`{"count":1,"count":0,"vulnerabilities":[]}`,
		`{"count":1,"vulnerabilities":[{"cveID":"CVE-2023-9999"}]} {}`,
		"{\"count\":1,\"vulnerabilities\":[{\"cveID\":\"CVE-2023-9999\",\"notes\":\"\xff\"}]}",
		`{"count":2,"vulnerabilities":[{"cveID":"CVE-2023-9999"}]}`,
		`{"count":2,"vulnerabilities":[{"cveID":"CVE-2023-9999"},{"cveID":"invalid"}]}`,
	} {
		body = invalid
		if _, err := s.IngestKEV(context.Background()); err == nil {
			t.Fatalf("accepted %q", body)
		}
		alerts := []Alert{{RuleID: "CVE-2023-4863"}}
		if err := s.annotateKEV(context.Background(), alerts); err != nil {
			t.Fatal(err)
		}
		if len(alerts[0].KEVs) != 1 || alerts[0].KEVs[0].Vendor != "Vendor" {
			t.Fatalf("previous catalog lost: %+v", alerts)
		}
	}
}

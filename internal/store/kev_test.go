package store

import (
	"context"
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

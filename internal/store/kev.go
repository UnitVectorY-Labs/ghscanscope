package store

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const KEVSource = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"

type KEV struct {
	Vendor         string `json:"vendorProject"`
	Product        string `json:"product"`
	Description    string `json:"shortDescription"`
	Ransomware     string `json:"knownRansomwareCampaignUse"`
	Notes          string `json:"notes"`
	CVE            string `json:"cveID"`
	Name           string `json:"vulnerabilityName"`
	DateAdded      string `json:"dateAdded"`
	RequiredAction string `json:"requiredAction"`
	DueDate        string `json:"dueDate"`
}

func upKEV(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "CREATE TABLE kev (cve TEXT PRIMARY KEY, payload TEXT NOT NULL)")
	return err
}
func downKEV(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "DROP TABLE kev")
	return err
}

var cvePattern = regexp.MustCompile(`(?i)\bCVE-[0-9]{4}-[0-9]{4,}\b`)

// IngestKEV replaces the cached catalog only after a complete, validated download.
func (s *Store) IngestKEV(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, KEVSource, nil)
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("CISA catalog: HTTP %d", resp.StatusCode)
	}
	var feed struct {
		Count           int   `json:"count"`
		Vulnerabilities []KEV `json:"vulnerabilities"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return 0, err
	}
	if err = json.Unmarshal(body, &feed); err != nil {
		return 0, err
	}
	if feed.Count == 0 || feed.Count != len(feed.Vulnerabilities) {
		return 0, fmt.Errorf("invalid CISA catalog count")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "DELETE FROM kev"); err != nil {
		return 0, err
	}
	for _, v := range feed.Vulnerabilities {
		if cvePattern.FindString(v.CVE) != v.CVE {
			return 0, fmt.Errorf("invalid CVE %q", v.CVE)
		}
		b, err := json.Marshal(v)
		if err != nil {
			return 0, err
		}
		if _, err = tx.ExecContext(ctx, s.sql("INSERT INTO kev(cve,payload) VALUES (?,?)"), strings.ToUpper(v.CVE), string(b)); err != nil {
			return 0, err
		}
	}
	return feed.Count, tx.Commit()
}
func (s *Store) annotateKEV(ctx context.Context, alerts []Alert) error {
	rows, err := s.query(ctx, "SELECT cve,payload FROM kev")
	if err != nil {
		return err
	}
	defer rows.Close()
	catalog := map[string]KEV{}
	for rows.Next() {
		var id, p string
		if err := rows.Scan(&id, &p); err != nil {
			return err
		}
		var v KEV
		if err := json.Unmarshal([]byte(p), &v); err != nil {
			return err
		}
		catalog[id] = v
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range alerts {
		a := &alerts[i]
		seen := map[string]bool{}
		for _, id := range cvePattern.FindAllString(strings.ToUpper(strings.Join([]string{a.RuleID, a.RuleName, a.RuleDescription, a.Message, strings.Join(a.Tags, " ")}, " ")), -1) {
			if v, ok := catalog[id]; ok && !seen[id] {
				a.KEVs = append(a.KEVs, v)
				seen[id] = true
			}
		}
	}
	return nil
}

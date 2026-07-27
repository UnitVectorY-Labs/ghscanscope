package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	HTTP           *http.Client
	BaseURL, Token string
}

type Repository struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	FullName    string  `json:"full_name"`
	HTMLURL     string  `json:"html_url"`
	Description *string `json:"description"`
	Visibility  string  `json:"visibility"`
	Private     bool    `json:"private"`
	Archived    bool    `json:"archived"`
	Language    *string `json:"language"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type Alert struct {
	Number    int64     `json:"number"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	HTMLURL   string    `json:"html_url"`
	Tool      struct {
		Name string `json:"name"`
	} `json:"tool"`
	Rule struct {
		ID, Name, Severity string
		SecuritySeverity   string `json:"security_severity_level"`
	} `json:"rule"`
	Repository         Repository `json:"repository"`
	MostRecentInstance struct {
		Location struct {
			Path        string `json:"path"`
			StartLine   int    `json:"start_line"`
			EndLine     int    `json:"end_line"`
			StartColumn int    `json:"start_column"`
			EndColumn   int    `json:"end_column"`
		} `json:"location"`
	} `json:"most_recent_instance"`
}

func Token() string {
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		return token
	}
	if token := strings.TrimSpace(os.Getenv("GH_TOKEN")); token != "" {
		return token
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "gh", "auth", "token", "--hostname", "github.com").Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}
	return ""
}

func New() *Client {
	return &Client{HTTP: &http.Client{Timeout: 30 * time.Second}, BaseURL: "https://api.github.com", Token: Token()}
}

func (c *Client) get(ctx context.Context, path string, out any) (http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "ghscanscope")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var message struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(body, &message)
		if message.Message == "" {
			message.Message = strings.TrimSpace(string(body))
		}
		return resp.Header, fmt.Errorf("GitHub API %s: %s", resp.Status, message.Message)
	}
	return resp.Header, json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) paginate(ctx context.Context, path string, newPage func() any, appendPage func(any)) error {
	for page := 1; ; page++ {
		separator := "?"
		if strings.Contains(path, "?") {
			separator = "&"
		}
		target := fmt.Sprintf("%s%sper_page=100&page=%d", path, separator, page)
		value := newPage()
		_, err := c.get(ctx, target, value)
		if err != nil {
			return err
		}
		appendPage(value)
		length := 0
		switch v := value.(type) {
		case *[]Repository:
			length = len(*v)
		case *[]Alert:
			length = len(*v)
		default:
			return errors.New("unsupported pagination type")
		}
		if length < 100 {
			return nil
		}
	}
}

func (c *Client) OrganizationRepositories(ctx context.Context, org string) ([]Repository, error) {
	var all []Repository
	err := c.paginate(ctx, "/orgs/"+url.PathEscape(org)+"/repos?type=all", func() any { return &[]Repository{} }, func(v any) { all = append(all, *v.(*[]Repository)...) })
	return all, err
}
func (c *Client) Repository(ctx context.Context, owner, repo string) (Repository, error) {
	var r Repository
	_, err := c.get(ctx, "/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(repo), &r)
	return r, err
}
func (c *Client) OrganizationAlerts(ctx context.Context, org string) ([]Alert, error) {
	var all []Alert
	err := c.paginate(ctx, "/orgs/"+url.PathEscape(org)+"/code-scanning/alerts?state=open", func() any { return &[]Alert{} }, func(v any) { all = append(all, *v.(*[]Alert)...) })
	return all, err
}
func (c *Client) RepositoryAlerts(ctx context.Context, owner, repo string) ([]Alert, error) {
	var all []Alert
	err := c.paginate(ctx, "/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(repo)+"/code-scanning/alerts?state=open", func() any { return &[]Alert{} }, func(v any) { all = append(all, *v.(*[]Alert)...) })
	return all, err
}

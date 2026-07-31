// Package docs searches and fetches Bolna's public documentation
// (https://www.bolna.ai/docs), via the llms.txt index Bolna publishes
// specifically for machine consumption — no scraping, no undocumented
// endpoint. This is deliberately separate from package api: it's
// unauthenticated (docs are public) and talks to a different host than the
// Bolna REST API.
package docs

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const baseURL = "https://www.bolna.ai/docs"

// Entry is one page listed in Bolna's llms.txt index.
type Entry struct {
	Title       string
	Path        string // relative to baseURL, no leading slash, no ".md"
	Description string
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

// indexLine matches an llms.txt bullet: "- [Title](URL): Description".
var indexLine = regexp.MustCompile(`^-\s*\[([^\]]+)\]\(([^)]+)\)(?::\s*(.*))?$`)

// FetchIndex retrieves and parses Bolna's llms.txt — every doc page's
// title, path, and one-line description.
func FetchIndex() ([]Entry, error) {
	body, err := get(baseURL + "/llms.txt")
	if err != nil {
		return nil, fmt.Errorf("fetching docs index: %w", err)
	}

	var entries []Entry
	for _, line := range strings.Split(body, "\n") {
		m := indexLine.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		entries = append(entries, Entry{
			Title:       m[1],
			Path:        pathFromURL(m[2]),
			Description: m[3],
		})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no entries found in llms.txt — Bolna may have changed its format")
	}
	return entries, nil
}

// Search ranks Entries by how many of the query's words appear in the
// title, description, or path (title matches count for more), returning
// only entries with at least one match, best matches first.
func Search(entries []Entry, query string) []Entry {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil
	}

	type scored struct {
		Entry
		score int
	}
	var results []scored
	for _, e := range entries {
		title := strings.ToLower(e.Title)
		desc := strings.ToLower(e.Description)
		path := strings.ToLower(e.Path)
		score := 0
		for _, t := range terms {
			if strings.Contains(title, t) {
				score += 3
			}
			if strings.Contains(desc, t) {
				score++
			}
			if strings.Contains(path, t) {
				score++
			}
		}
		if score > 0 {
			results = append(results, scored{e, score})
		}
	}

	sort.SliceStable(results, func(i, j int) bool { return results[i].score > results[j].score })

	out := make([]Entry, len(results))
	for i, r := range results {
		out[i] = r.Entry
	}
	return out
}

// FetchPage retrieves one doc page's raw Markdown. path may be a bare page
// path (e.g. "build-with-ai/mcp-tool-list"), a path with a ".md" suffix, or
// a full bolna.ai/docs URL — all three are normalized to the same request.
func FetchPage(path string) (string, error) {
	p := pathFromURL(path)
	if p == "" {
		return "", fmt.Errorf("empty doc path")
	}
	body, err := get(baseURL + "/" + p + ".md")
	if err != nil {
		return "", fmt.Errorf("fetching doc page %q: %w", p, err)
	}
	return body, nil
}

// pathFromURL strips a bolna.ai/docs URL (or bare "/docs/..." path) down to
// the bare page path: no scheme/host, no leading "/docs/", no ".md".
func pathFromURL(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "/docs/"); i >= 0 {
		s = s[i+len("/docs/"):]
	}
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimSuffix(s, ".md")
	return s
}

func get(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/markdown, text/plain, */*")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("reaching bolna.ai: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return string(raw), nil
}

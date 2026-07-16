// Package api is a thin, typed client for the Bolna REST API
// (https://api.bolna.ai), covering exactly the surface exposed by Bolna's
// MCP tool list: agents, calls/executions, phone numbers, batches, account.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const DefaultBaseURL = "https://api.bolna.ai"

// Client talks to the Bolna REST API on behalf of one API key.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

// New builds a Client against the production Bolna API.
func New(apiKey string) *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError represents a non-2xx response from the Bolna API. Status, the raw
// message the API returned, and (for 429s) the Retry-After value are all
// preserved so callers can render an actionable message.
type APIError struct {
	Status     int
	Message    string
	RetryAfter string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("bolna API error (HTTP %d): %s", e.Status, e.Message)
}

// Friendly renders a human-actionable message for the common cases, mirroring
// the error-handling contract Bolna's own MCP server uses (401/403 invalid
// key, 404 not-found with a hint, 422 validation, 429 rate limit).
func (e *APIError) Friendly(hint string) string {
	switch e.Status {
	case 401, 403:
		return "Bolna API key invalid or expired. Run `bolna login` to set a valid key."
	case 404:
		if hint != "" {
			return fmt.Sprintf("Not found (HTTP 404): %s. %s", e.Message, hint)
		}
		return fmt.Sprintf("Not found (HTTP 404): %s", e.Message)
	case 422:
		return fmt.Sprintf("Validation error (HTTP 422): %s", e.Message)
	case 429:
		if e.RetryAfter != "" {
			return fmt.Sprintf("Rate limited by the Bolna API (HTTP 429): %s. Retry after %s seconds.", e.Message, e.RetryAfter)
		}
		return fmt.Sprintf("Rate limited by the Bolna API (HTTP 429): %s", e.Message)
	default:
		return e.Error()
	}
}

type requestOptions struct {
	method string
	query  url.Values
	body   any
}

func (c *Client) do(path string, opts requestOptions, out any) error {
	if c.APIKey == "" {
		return fmt.Errorf("no Bolna API key configured — run `bolna login` or set BOLNA_API_KEY")
	}

	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return fmt.Errorf("building request URL: %w", err)
	}
	if opts.query != nil {
		u.RawQuery = opts.query.Encode()
	}

	method := opts.method
	if method == "" {
		method = http.MethodGet
	}

	var bodyReader io.Reader
	if opts.body != nil {
		encoded, err := json.Marshal(opts.body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("reaching the Bolna API: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			Status:     resp.StatusCode,
			Message:    extractMessage(raw),
			RetryAfter: resp.Header.Get("Retry-After"),
		}
	}

	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decoding Bolna API response: %w", err)
	}
	return nil
}

// extractMessage pulls the most useful string out of a Bolna error body,
// which inconsistently uses "message", "detail" (auth errors), or "error".
func extractMessage(raw []byte) string {
	if len(raw) == 0 {
		return "no further details were returned by the Bolna API"
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err == nil {
		for _, key := range []string{"message", "detail", "error"} {
			if s, ok := body[key].(string); ok && s != "" {
				return s
			}
		}
	}
	return string(raw)
}

// paginate applies client-side pagination to endpoints that return a bare
// array (list_agents, list_batches don't paginate server-side).
func paginate[T any](items []T, pageNumber, pageSize int) []T {
	if pageNumber < 1 {
		pageNumber = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	start := (pageNumber - 1) * pageSize
	if start >= len(items) {
		return []T{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

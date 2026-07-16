package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL, APIKey: "test-key", HTTP: srv.Client()}
}

func TestListAgents(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/agent/all" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing/incorrect Authorization header: %q", r.Header.Get("Authorization"))
		}
		agents := []AgentSummary{
			{ID: "a1", AgentName: "Agent One", AgentStatus: "active", CreatedAt: "2026-01-01"},
			{ID: "a2", AgentName: "Agent Two", AgentStatus: "inactive", CreatedAt: "2026-01-02"},
		}
		json.NewEncoder(w).Encode(agents)
	})

	agents, err := client.ListAgents(0, 0)
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0].ID != "a1" {
		t.Errorf("expected first agent id a1, got %s", agents[0].ID)
	}
}

func TestListAgentsClientSidePagination(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		agents := make([]AgentSummary, 25)
		for i := range agents {
			agents[i] = AgentSummary{ID: string(rune('a' + i))}
		}
		json.NewEncoder(w).Encode(agents)
	})

	page1, err := client.ListAgents(1, 10)
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(page1) != 10 {
		t.Fatalf("expected 10 agents on page 1, got %d", len(page1))
	}

	page3, err := client.ListAgents(3, 10)
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(page3) != 5 {
		t.Fatalf("expected 5 agents on page 3 (25 total), got %d", len(page3))
	}
}

func TestAPIErrorMessageExtraction(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"message field", 422, `{"message":"agent_name is required"}`, "agent_name is required"},
		{"detail field (auth errors)", 403, `{"detail":"Unrecognized access token"}`, "Unrecognized access token"},
		{"error field", 500, `{"error":"boom"}`, "boom"},
		{"empty body", 500, ``, "no further details were returned by the Bolna API"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			})
			_, err := client.GetUserInfo()
			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("expected *APIError, got %T: %v", err, err)
			}
			if apiErr.Status != tc.status {
				t.Errorf("expected status %d, got %d", tc.status, apiErr.Status)
			}
			if apiErr.Message != tc.want {
				t.Errorf("expected message %q, got %q", tc.want, apiErr.Message)
			}
		})
	}
}

func TestAPIErrorFriendlyMessages(t *testing.T) {
	cases := []struct {
		status int
		hint   string
		want   string
	}{
		{401, "", "Bolna API key invalid or expired. Run `bolna login` to set a valid key."},
		{403, "", "Bolna API key invalid or expired. Run `bolna login` to set a valid key."},
		{404, "try X", "Not found (HTTP 404): nope. try X"},
		{422, "", "Validation error (HTTP 422): bad field"},
		{429, "", "Rate limited by the Bolna API (HTTP 429): slow down"},
	}
	for _, tc := range cases {
		err := &APIError{Status: tc.status, Message: map[int]string{404: "nope", 422: "bad field", 429: "slow down"}[tc.status]}
		got := err.Friendly(tc.hint)
		if got != tc.want {
			t.Errorf("status %d: expected %q, got %q", tc.status, tc.want, got)
		}
	}
}

func TestListAgentExecutionsDefaultsToLast7Days(t *testing.T) {
	var gotFrom, gotTo string
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotFrom = r.URL.Query().Get("from")
		gotTo = r.URL.Query().Get("to")
		json.NewEncoder(w).Encode(ExecutionsPage{})
	})

	_, err := client.ListAgentExecutions(ListExecutionsInput{AgentID: "a1"})
	if err != nil {
		t.Fatalf("ListAgentExecutions: %v", err)
	}
	if gotFrom == "" || gotTo == "" {
		t.Fatalf("expected from/to query params to be set, got from=%q to=%q", gotFrom, gotTo)
	}
}

func TestNoAPIKeyError(t *testing.T) {
	client := &Client{BaseURL: "http://unused.invalid", APIKey: "", HTTP: http.DefaultClient}
	_, err := client.GetUserInfo()
	if err == nil {
		t.Fatal("expected an error when no API key is configured")
	}
}

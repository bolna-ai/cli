package api

import (
	"fmt"
	"net/url"
	"time"
)

// StartCallInput mirrors POST /call's body.
type StartCallInput struct {
	AgentID              string         `json:"agent_id"`
	RecipientPhoneNumber string         `json:"recipient_phone_number"`
	FromPhoneNumber      string         `json:"from_phone_number,omitempty"`
	UserData             map[string]any `json:"user_data,omitempty"`
}

// StartCall places a real outbound call and spends account balance.
func (c *Client) StartCall(input StartCallInput) (map[string]any, error) {
	var result map[string]any
	err := c.do("/call", requestOptions{method: "POST", body: input}, &result)
	return result, err
}

// TelephonyData is the trimmed telephony summary attached to an execution.
type TelephonyData struct {
	ToNumber   string `json:"to_number,omitempty"`
	FromNumber string `json:"from_number,omitempty"`
	Provider   string `json:"provider,omitempty"`
}

// ExecutionSummary is one row of GET /v2/agent/{id}/executions.
type ExecutionSummary struct {
	ID                   string         `json:"id"`
	Status               string         `json:"status"`
	ConversationDuration *float64       `json:"conversation_duration"`
	CreatedAt            string         `json:"created_at"`
	TelephonyData        *TelephonyData `json:"telephony_data,omitempty"`
}

// ExecutionsPage is the paginated envelope GET /v2/agent/{id}/executions
// returns — a real paginated response, not a bare array like list_agents.
type ExecutionsPage struct {
	Data       []ExecutionSummary `json:"data"`
	Total      int                `json:"total"`
	HasMore    bool               `json:"has_more"`
	PageNumber int                `json:"page_number"`
	PageSize   int                `json:"page_size"`
}

// ListExecutionsInput configures GET /v2/agent/{id}/executions. From/To
// default to the last 7 days if left zero (the Bolna API rejects windows
// wider than 7 days), matching the Bolna MCP server's default.
type ListExecutionsInput struct {
	AgentID    string
	From       time.Time
	To         time.Time
	PageNumber int
	PageSize   int
}

func (c *Client) ListAgentExecutions(input ListExecutionsInput) (*ExecutionsPage, error) {
	to := input.To
	if to.IsZero() {
		to = time.Now().UTC()
	}
	from := input.From
	if from.IsZero() {
		from = to.Add(-7 * 24 * time.Hour)
	}
	pageNumber := input.PageNumber
	if pageNumber < 1 {
		pageNumber = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	query := url.Values{}
	query.Set("from", from.Format(time.RFC3339))
	query.Set("to", to.Format(time.RFC3339))
	query.Set("page_number", fmt.Sprintf("%d", pageNumber))
	query.Set("page_size", fmt.Sprintf("%d", pageSize))

	var page ExecutionsPage
	err := c.do(
		"/v2/agent/"+url.PathEscape(input.AgentID)+"/executions",
		requestOptions{query: query},
		&page,
	)
	return &page, err
}

// Execution is the full call record from GET /executions/{id}: status,
// transcript, telephony data, cost, extracted data. Kept generic since the
// shape (extracted_data especially) varies by agent/disposition config.
type Execution map[string]any

func (e Execution) str(key string) string {
	if v, ok := e[key].(string); ok {
		return v
	}
	return ""
}

func (e Execution) ID() string     { return e.str("id") }
func (e Execution) Status() string { return e.str("status") }

// Transcript returns the conversation transcript if present, either as a
// plain string or (depending on API version) a list of {role, content} turns
// serialized back into a readable block.
func (e Execution) Transcript() string {
	if s, ok := e["transcript"].(string); ok {
		return s
	}
	return ""
}

func (c *Client) GetExecution(executionID string) (Execution, error) {
	var execution Execution
	err := c.do("/executions/"+url.PathEscape(executionID), requestOptions{}, &execution)
	return execution, err
}

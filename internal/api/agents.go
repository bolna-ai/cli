package api

import "net/url"

// AgentSummary is the trimmed shape returned by GET /v2/agent/all.
type AgentSummary struct {
	ID          string `json:"id"`
	AgentName   string `json:"agent_name"`
	AgentStatus string `json:"agent_status"`
	CreatedAt   string `json:"created_at"`
}

// Agent is a full agent configuration (GET /v2/agent/{id}, POST /v2/agent
// response, PATCH response). Its shape varies by provider/task config, so it
// is kept as a generic JSON document rather than modeled field-by-field —
// callers pull out display fields defensively via the Field/Agent helpers.
type Agent map[string]any

func (a Agent) str(key string) string {
	if v, ok := a[key].(string); ok {
		return v
	}
	return ""
}

func (a Agent) Name() string   { return a.str("agent_name") }
func (a Agent) Status() string { return a.str("agent_status") }
func (a Agent) ID() string     { return a.str("id") }

// WelcomeMessage returns agent_welcome_message if present.
func (a Agent) WelcomeMessage() string { return a.str("agent_welcome_message") }

// SystemPrompt returns the first task's system_prompt from the top-level
// agent_prompts map (e.g. agent_prompts.task_1.system_prompt) — confirmed
// against the live GET /v2/agent/{id} response, which mirrors the
// {agent_config, agent_prompts} shape used to create the agent rather than
// nesting prompts under tasks[].tools_config. Prefers "task_1" but falls
// back to any task key present, since a second (summarization) task may
// come first in map iteration order.
func (a Agent) SystemPrompt() string {
	prompts, ok := a["agent_prompts"].(map[string]any)
	if !ok {
		return ""
	}
	if task1, ok := prompts["task_1"].(map[string]any); ok {
		if raw, ok := task1["system_prompt"].(string); ok && raw != "" {
			return raw
		}
	}
	for _, v := range prompts {
		if task, ok := v.(map[string]any); ok {
			if raw, ok := task["system_prompt"].(string); ok && raw != "" {
				return raw
			}
		}
	}
	return ""
}

// AgentConfig returns the agent's current nested agent_config object from a
// GET /v2/agent/{id} response, or nil if absent. Used by `update` to patch
// from the full existing config rather than a destructive partial.
func (a Agent) AgentConfig() map[string]any {
	m, _ := a["agent_config"].(map[string]any)
	return m
}

// AgentPrompts returns the agent's current nested agent_prompts object (e.g.
// {task_1: {...}, task_2: {...}}) from a GET response, or nil if absent.
func (a Agent) AgentPrompts() map[string]any {
	m, _ := a["agent_prompts"].(map[string]any)
	return m
}

// CreateAgentInput mirrors POST /v2/agent's body. agent_config and
// agent_prompts are passed through as generic maps deliberately: the full
// v2 schema is large and provider-dependent, and re-modeling it field by
// field risks silently dropping fields the API accepts.
type CreateAgentInput struct {
	AgentConfig  map[string]any `json:"agent_config"`
	AgentPrompts map[string]any `json:"agent_prompts"`
}

// UpdateAgentInput mirrors PATCH /v2/agent/{id}'s body; both fields optional.
type UpdateAgentInput struct {
	AgentConfig  map[string]any `json:"agent_config,omitempty"`
	AgentPrompts map[string]any `json:"agent_prompts,omitempty"`
}

// ListAgents fetches every agent on the account (GET /v2/agent/all does not
// paginate server-side). Pass pageNumber/pageSize > 0 to slice the result
// client-side for JSON-mode output parity with the MCP tool; pass 0 for both
// to get everything (used by the TUI, which paginates in-memory).
func (c *Client) ListAgents(pageNumber, pageSize int) ([]AgentSummary, error) {
	var agents []AgentSummary
	if err := c.do("/v2/agent/all", requestOptions{}, &agents); err != nil {
		return nil, err
	}
	if pageNumber > 0 && pageSize > 0 {
		return paginate(agents, pageNumber, pageSize), nil
	}
	return agents, nil
}

// GetAgent fetches the full configuration of one agent.
func (c *Client) GetAgent(agentID string) (Agent, error) {
	var agent Agent
	if err := c.do("/v2/agent/"+url.PathEscape(agentID), requestOptions{}, &agent); err != nil {
		return nil, err
	}
	return agent, nil
}

// CreateAgent creates a new agent and returns the API's raw response
// (typically includes the new agent's id and status).
func (c *Client) CreateAgent(input CreateAgentInput) (map[string]any, error) {
	var result map[string]any
	err := c.do("/v2/agent", requestOptions{method: "POST", body: input}, &result)
	return result, err
}

// UpdateAgent applies a partial patch to an existing agent.
func (c *Client) UpdateAgent(agentID string, input UpdateAgentInput) (map[string]any, error) {
	var result map[string]any
	err := c.do("/v2/agent/"+url.PathEscape(agentID), requestOptions{method: "PATCH", body: input}, &result)
	return result, err
}

// DeleteAgent permanently deletes an agent and its history.
func (c *Client) DeleteAgent(agentID string) (map[string]any, error) {
	var result map[string]any
	err := c.do("/v2/agent/"+url.PathEscape(agentID), requestOptions{method: "DELETE"}, &result)
	return result, err
}

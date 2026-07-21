package cli

import (
	"testing"

	"github.com/bolna-ai/cli/internal/api"
)

// fullAgent is a GET /v2/agent/{id} response with a complete pipeline, in the
// nested {agent_config, agent_prompts} shape update patches from.
func fullAgent() api.Agent {
	return api.Agent{
		"agent_name":            "Old Name",
		"agent_welcome_message": "Hi there",
		"agent_config": map[string]any{
			"agent_name":            "Old Name",
			"agent_welcome_message": "Hi there",
			"tasks":                 []any{"telephony", "llm"},
			"llm":                   map[string]any{"model": "gpt-4"},
			"synthesizer":           map[string]any{"voice": "aria"},
		},
		"agent_prompts": map[string]any{
			"task_1": map[string]any{"system_prompt": "answer calls"},
			"task_2": map[string]any{"system_prompt": "summarize the call"},
		},
	}
}

// A rename must carry the whole existing agent_config back, so a replacing
// PATCH can't drop tasks/llm/synthesizer. This is the core data-loss guard.
func TestBuildAgentUpdatePreservesConfigOnRename(t *testing.T) {
	input, changed := buildAgentUpdate(fullAgent(), "New Name", "", "", "")

	if input.AgentConfig["agent_name"] != "New Name" {
		t.Errorf("agent_name = %v, want New Name", input.AgentConfig["agent_name"])
	}
	for _, key := range []string{"tasks", "llm", "synthesizer", "agent_welcome_message"} {
		if _, ok := input.AgentConfig[key]; !ok {
			t.Errorf("rename dropped agent_config.%s", key)
		}
	}
	if _, ok := changed["agent_name"]; !ok {
		t.Error("expected agent_name in the change summary")
	}
}

// Updating task_1's prompt must not drop the task_2 (summarization) prompt.
func TestBuildAgentUpdatePreservesSiblingPrompts(t *testing.T) {
	input, _ := buildAgentUpdate(fullAgent(), "", "", "new task_1 prompt", "")

	prompts := input.AgentPrompts
	task1 := prompts["task_1"].(map[string]any)
	if task1["system_prompt"] != "new task_1 prompt" {
		t.Errorf("task_1 prompt = %v, want updated", task1["system_prompt"])
	}
	task2, ok := prompts["task_2"].(map[string]any)
	if !ok {
		t.Fatal("task_2 prompt was dropped")
	}
	if task2["system_prompt"] != "summarize the call" {
		t.Errorf("task_2 prompt = %v, want preserved", task2["system_prompt"])
	}
}

// The builder must not mutate the caller's fetched agent (it copies before
// writing), so the change summary's before-values stay accurate.
func TestBuildAgentUpdateDoesNotMutateCurrent(t *testing.T) {
	current := fullAgent()
	buildAgentUpdate(current, "New Name", "", "changed prompt", "")

	if current.AgentConfig()["agent_name"] != "Old Name" {
		t.Error("builder mutated current agent_config")
	}
	origTask1 := current.AgentPrompts()["task_1"].(map[string]any)
	if origTask1["system_prompt"] != "answer calls" {
		t.Error("builder mutated current agent_prompts")
	}
}

// No-op / unchanged values produce an empty payload and empty summary, so we
// never PATCH (and never risk a clobber) when nothing actually changed.
func TestBuildAgentUpdateNoChanges(t *testing.T) {
	input, changed := buildAgentUpdate(fullAgent(), "Old Name", "Hi there", "answer calls", "")

	if len(changed) != 0 {
		t.Errorf("expected no changes, got %v", changed)
	}
	if input.AgentConfig != nil {
		t.Errorf("expected nil AgentConfig, got %v", input.AgentConfig)
	}
	if input.AgentPrompts != nil {
		t.Errorf("expected nil AgentPrompts, got %v", input.AgentPrompts)
	}
}

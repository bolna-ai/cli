package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bolna-ai/cli/internal/api"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

func newAgentsCmd(a *appCtx) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agents",
		Aliases: []string{"agent"},
		Short:   "Manage Bolna voice AI agents",
	}
	cmd.AddCommand(
		newAgentsListCmd(a),
		newAgentsViewCmd(a),
		newAgentsCreateCmd(a),
		newAgentsUpdateCmd(a),
		newAgentsDeleteCmd(a),
	)
	return cmd
}

func newAgentsListCmd(a *appCtx) *cobra.Command {
	var page, pageSize int
	var quiet bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List agents — ID, name, status, created date",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := a.ClientOrLogin()
			if err != nil {
				return err
			}
			agents, err := client.ListAgents(page, pageSize)
			if err != nil {
				return friendlyAPIErr(err, "")
			}
			headers := []string{"ID", "NAME", "STATUS", "CREATED"}
			rows := make([][]string, len(agents))
			for i, ag := range agents {
				rows[i] = []string{ag.ID, ag.AgentName, ag.AgentStatus, ag.CreatedAt}
			}
			return a.renderList(headers, rows, 0, 2, agents, quiet)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "page number (table/csv/json alike)")
	cmd.Flags().IntVar(&pageSize, "page-size", 50, "page size, max 50")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "print only agent IDs, one per line (for scripting)")
	return cmd
}

func newAgentsViewCmd(a *appCtx) *cobra.Command {
	var asPNG string
	cmd := &cobra.Command{
		Use:   "view <agent-id>",
		Short: "Full config of one agent — prompts, LLM, voice, telephony, tools",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := a.ClientOrLogin()
			if err != nil {
				return err
			}
			agent, err := client.GetAgent(args[0])
			if err != nil {
				return friendlyAPIErr(err, "Call `bolna agents list` to see valid agent IDs.")
			}
			if a.Format() == "json" {
				return printJSON(agent)
			}

			theme := a.Theme()
			var body string
			header := theme.Title.Render(orDash(agent.Name())) + "  " + theme.StatusColor(agent.Status()).Render(orDash(agent.Status()))
			body = header + "\n" + theme.Muted.Render("ID: "+agent.ID())
			if wm := agent.WelcomeMessage(); wm != "" {
				body += "\n\n" + theme.Subtitle.Render("Welcome message") + "\n" + wm
			}
			if sp := agent.SystemPrompt(); sp != "" {
				rendered, err := glamour.Render("**System prompt**\n\n"+sp, "auto")
				if err == nil {
					body += "\n" + rendered
				} else {
					body += "\n\n" + theme.Subtitle.Render("System prompt") + "\n" + sp
				}
			}
			card := theme.Card.Render(body)
			fmt.Println(card)

			if asPNG != "" {
				if err := renderCardToPNG(card, asPNG); err != nil {
					return fmt.Errorf("exporting PNG: %w", err)
				}
				fmt.Println(theme.Success.Render("✓ Saved snapshot to " + asPNG))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&asPNG, "png", "", "also render this card to a PNG file (via Freeze) for sharing")
	return cmd
}

func newAgentsCreateCmd(a *appCtx) *cobra.Command {
	var fromFile string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent, returns its ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := a.ClientOrLogin()
			if err != nil {
				return err
			}

			var input api.CreateAgentInput
			if fromFile != "" {
				raw, err := os.ReadFile(fromFile)
				if err != nil {
					return fmt.Errorf("reading %s: %w", fromFile, err)
				}
				if err := json.Unmarshal(raw, &input); err != nil {
					return fmt.Errorf("parsing %s: %w", fromFile, err)
				}
			} else {
				if !IsTTY() {
					return fmt.Errorf("interactive creation requires a terminal; pass --file with a JSON payload instead")
				}
				built, err := runCreateAgentWizard()
				if err != nil {
					return err
				}
				input = built
			}

			result, err := client.CreateAgent(input)
			if err != nil {
				return friendlyAPIErr(err, "")
			}
			if a.Format() == "json" {
				return printJSON(result)
			}
			theme := a.Theme()
			id, _ := result["agent_id"].(string)
			fmt.Println(theme.Success.Render("✓ Agent created") + "  " + theme.Muted.Render(id))
			return nil
		},
	}
	cmd.Flags().StringVar(&fromFile, "file", "", "path to a JSON file with {agent_config, agent_prompts} for full control instead of the wizard")
	return cmd
}

// buildAgentUpdate builds the PATCH body from the agent's CURRENT full config
// plus the requested changes. It starts from a copy of the existing nested
// agent_config / agent_prompts so the payload carries the complete object and
// only the intended fields differ — safe whether the API replaces or
// shallow-merges those top-level keys. A bare partial payload (the old
// behavior) would wipe everything else under a rename: tasks, voice, LLM, or a
// sibling task_2 prompt.
//
// ponytail: correctness relies on GET returning nested agent_config/agent_prompts
// (documented at api/agents.go:34). If GET omits agent_config, the base is empty
// and this degrades to the old partial payload for config-only edits — confirm
// the GET/PATCH shapes with one curl to close that gap. agent_prompts is
// confirmed nested (SystemPrompt reads it), so the task_2 fix holds regardless.
func buildAgentUpdate(current api.Agent, name, welcome, prompt, webhook string) (api.UpdateAgentInput, map[string]string) {
	changed := map[string]string{}

	config := copyMap(current.AgentConfig())
	configTouched := false
	if name != "" && name != current.Name() {
		config["agent_name"] = name
		changed["agent_name"] = fmt.Sprintf("%q → %q", current.Name(), name)
		configTouched = true
	}
	if welcome != "" && welcome != current.WelcomeMessage() {
		config["agent_welcome_message"] = welcome
		changed["agent_welcome_message"] = fmt.Sprintf("%q → %q", current.WelcomeMessage(), welcome)
		configTouched = true
	}
	if webhook != "" {
		config["webhook_url"] = webhook
		changed["webhook_url"] = webhook
		configTouched = true
	}

	var input api.UpdateAgentInput
	if configTouched {
		input.AgentConfig = config
	}
	if prompt != "" && prompt != current.SystemPrompt() {
		prompts := copyMap(current.AgentPrompts())
		task1 := copyMap(mapOf(prompts["task_1"]))
		task1["system_prompt"] = prompt
		prompts["task_1"] = task1
		input.AgentPrompts = prompts
		changed["system_prompt"] = "(updated — see diff above)"
	}
	return input, changed
}

// copyMap returns a shallow copy of m (nil → empty map). Shallow is enough here
// because callers overwrite whole top-level entries, never mutate nested values.
func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	return out
}

func mapOf(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func newAgentsUpdateCmd(a *appCtx) *cobra.Command {
	var name, welcome, prompt, webhook string
	var yes bool
	cmd := &cobra.Command{
		Use:   "update <agent-id>",
		Short: "Patch an agent's name, prompts, welcome message, webhook, or voice",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			client, err := a.ClientOrLogin()
			if err != nil {
				return err
			}
			current, err := client.GetAgent(agentID)
			if err != nil {
				return friendlyAPIErr(err, "Call `bolna agents list` to see valid agent IDs.")
			}

			if name == "" && welcome == "" && prompt == "" && webhook == "" {
				if !IsTTY() {
					return fmt.Errorf("no --name/--welcome/--prompt/--webhook given and stdin is not a terminal")
				}
				n, w, p, wh, err := runUpdateAgentWizard(current)
				if err != nil {
					return err
				}
				name, welcome, prompt, webhook = n, w, p, wh
			}

			input, changed := buildAgentUpdate(current, name, welcome, prompt, webhook)

			theme := a.Theme()
			if len(changed) == 0 {
				fmt.Println(theme.Muted.Render("Nothing to change."))
				return nil
			}

			fmt.Println(theme.Subtitle.Render("Pending changes:"))
			for field, diff := range changed {
				fmt.Printf("  %s %s\n", theme.Bold.Render(field+":"), diff)
			}

			if !yes && IsTTY() {
				confirmed := false
				if err := huh.NewForm(huh.NewGroup(
					huh.NewConfirm().Title("Apply these changes?").Value(&confirmed),
				)).Run(); err != nil {
					return err
				}
				if !confirmed {
					fmt.Println(theme.Muted.Render("Cancelled."))
					return nil
				}
			}

			result, err := client.UpdateAgent(agentID, input)
			if err != nil {
				return friendlyAPIErr(err, "Call `bolna agents list` to see valid agent IDs.")
			}
			if a.Format() == "json" {
				return printJSON(result)
			}
			fmt.Println(theme.Success.Render("✓ Agent updated"))
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "new agent name")
	cmd.Flags().StringVar(&welcome, "welcome", "", "new welcome message")
	cmd.Flags().StringVar(&prompt, "prompt", "", "new system prompt (task_1)")
	cmd.Flags().StringVar(&webhook, "webhook", "", "new webhook URL")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt (scripting)")
	return cmd
}

func newAgentsDeleteCmd(a *appCtx) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <agent-id>",
		Short: "Permanently delete an agent and its history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			client, err := a.ClientOrLogin()
			if err != nil {
				return err
			}
			theme := a.Theme()

			if !yes {
				agent, err := client.GetAgent(agentID)
				if err != nil {
					return friendlyAPIErr(err, "Call `bolna agents list` to see valid agent IDs.")
				}
				if !IsTTY() {
					return fmt.Errorf("refusing to delete without confirmation in a non-interactive session; pass --yes")
				}
				fmt.Println(theme.Danger.Render("⚠ This permanently deletes the agent and its call/batch history. This cannot be undone."))
				var typed string
				if err := huh.NewForm(huh.NewGroup(
					huh.NewInput().
						Title(fmt.Sprintf("Type the agent name (%s) to confirm deletion", agent.Name())).
						Value(&typed),
				)).Run(); err != nil {
					return err
				}
				if typed != agent.Name() {
					fmt.Println(theme.Muted.Render("Name didn't match — cancelled."))
					return nil
				}
			}

			result, err := client.DeleteAgent(agentID)
			if err != nil {
				return friendlyAPIErr(err, "Call `bolna agents list` to see valid agent IDs.")
			}
			if a.Format() == "json" {
				return printJSON(result)
			}
			fmt.Println(theme.Success.Render("✓ Agent deleted"))
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt (scripting)")
	return cmd
}

// runCreateAgentWizard walks the user through a minimal-but-real v2 agent
// payload (per Bolna's create-agent guidance): one conversation task with an
// OpenAI LLM, a chosen TTS/telephony provider pair, and a Deepgram
// transcriber. Advanced configs (multilingual, RAG, tools) are out of scope
// for the wizard — use `agents create --file` for those.
func runCreateAgentWizard() (api.CreateAgentInput, error) {
	var name, welcome, prompt, llmModel, ttsProvider, telephony string
	// Defaults are a known-good ElevenLabs voice (confirmed against the live
	// API): the synthesizer config is rejected with "requires 'voice' or
	// 'voice_id'" if either is missing, and "voice" alone (e.g. a bare voice
	// name with no matching voice_id) isn't enough — both fields, plus a
	// real model name, are required together.
	voiceName := "Nila"
	voiceID := "V9LCAAi4tTlqe9JadbCo"
	voiceModel := "eleven_turbo_v2_5"
	llmModel = "gpt-4.1-mini"
	ttsProvider = "elevenlabs"
	telephony = "twilio"

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Agent name").Value(&name).Validate(huh.ValidateNotEmpty()),
			huh.NewText().Title("Welcome message").Description("First line the agent says. Supports {variables}.").Value(&welcome),
			huh.NewText().Title("System prompt").Description("The agent's persona and instructions.").Value(&prompt).Validate(huh.ValidateNotEmpty()),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("LLM model").
				Options(
					huh.NewOption("OpenAI gpt-4.1-mini (fast, cheap)", "gpt-4.1-mini"),
					huh.NewOption("OpenAI gpt-4o (higher quality)", "gpt-4o"),
					huh.NewOption("OpenAI gpt-4o-mini", "gpt-4o-mini"),
				).Value(&llmModel),
			huh.NewSelect[string]().Title("Voice provider (TTS)").
				Options(
					huh.NewOption("ElevenLabs", "elevenlabs"),
					huh.NewOption("Sarvam (Indian languages)", "sarvam"),
					huh.NewOption("Cartesia", "cartesia"),
				).Value(&ttsProvider),
			huh.NewInput().Title("Voice name").Description("Defaults to a verified-working ElevenLabs voice; check your provider's dashboard for others.").Value(&voiceName),
			huh.NewInput().Title("Voice ID").Description("Required alongside the voice name by most providers.").Value(&voiceID),
			huh.NewInput().Title("Voice model").Value(&voiceModel),
			huh.NewSelect[string]().Title("Telephony provider").
				Options(
					huh.NewOption("Twilio", "twilio"),
					huh.NewOption("Plivo", "plivo"),
					huh.NewOption("Exotel", "exotel"),
					huh.NewOption("Vobiz", "vobiz"),
				).Value(&telephony),
		),
	).Run()
	if err != nil {
		return api.CreateAgentInput{}, err
	}

	agentConfig := map[string]any{
		"agent_name":            name,
		"agent_welcome_message": welcome,
		"agent_type":            "other",
		"tasks": []any{
			map[string]any{
				"task_type": "conversation",
				"tools_config": map[string]any{
					"llm_agent": map[string]any{
						"agent_type":      "simple_llm_agent",
						"agent_flow_type": "streaming",
						"llm_config": map[string]any{
							"provider":    "openai",
							"family":      "openai",
							"model":       llmModel,
							"max_tokens":  200,
							"temperature": 0.2,
						},
					},
					"synthesizer": map[string]any{
						"provider": ttsProvider,
						"provider_config": map[string]any{
							"voice":    voiceName,
							"voice_id": voiceID,
							"model":    voiceModel,
						},
						"stream":       true,
						"audio_format": "wav",
					},
					"transcriber": map[string]any{
						"provider":      "deepgram",
						"model":         "nova-3",
						"language":      "en",
						"stream":        true,
						"encoding":      "linear16",
						"sampling_rate": 16000,
						"endpointing":   400,
					},
					"input":  map[string]any{"provider": telephony, "format": "wav"},
					"output": map[string]any{"provider": telephony, "format": "wav"},
				},
				"toolchain": map[string]any{
					"execution": "parallel",
					"pipelines": []any{[]any{"transcriber", "llm", "synthesizer"}},
				},
			},
		},
	}

	return api.CreateAgentInput{
		AgentConfig:  agentConfig,
		AgentPrompts: map[string]any{"task_1": map[string]any{"system_prompt": prompt}},
	}, nil
}

func runUpdateAgentWizard(current api.Agent) (name, welcome, prompt, webhook string, err error) {
	name = current.Name()
	welcome = current.WelcomeMessage()
	prompt = current.SystemPrompt()

	err = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Agent name").Value(&name),
		huh.NewText().Title("Welcome message").Value(&welcome),
		huh.NewText().Title("System prompt").Value(&prompt),
		huh.NewInput().Title("Webhook URL").Description("leave empty to leave unchanged").Value(&webhook),
	)).Run()
	return
}

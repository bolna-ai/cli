# bolna

**🚧 Beta** — actively developed, interfaces and commands may still change.
Bug reports and feedback welcome.

The Bolna Voice AI CLI — and a full-screen terminal dashboard — for managing
agents, calls, phone numbers, and batches without leaving your terminal.

![demo](demo/demo.gif)

Every command mirrors a tool in [Bolna's MCP tool
list](https://www.bolna.ai/docs/build-with-ai/mcp-tool-list): agents, calls &
executions, phone numbers, batches, and account info. Run `bolna` with no
arguments in a terminal and you get a "mission control" dashboard instead of
a wall of JSON. Pipe it, script it, or run it in CI, and it behaves like a
normal, quiet, JSON-friendly CLI.

Built entirely on the [Charm](https://charm.land) stack - Bubble Tea, Bubbles,
Lip Gloss, Huh, Glamour, and Harmonica.

## Features

- **Full-screen dashboard** (`bolna`, no args) — bordered "mission control"
  frame (a steel-blue → periwinkle gradient border, matching the ASCII
  wordmark on mcp.bolna.ai) over agents, calls, numbers, batches, and
  account, with a spring-physics splash animation on startup (Harmonica).
- **Command palette** — press `:` anywhere in the dashboard for a fuzzy
  jump-to-agent/section launcher (Bubbles `list` with built-in filtering).
- **Start a call from the dashboard** — press `s` from an agent's detail
  screen: phone number prompt, a confirmation showing your real wallet
  balance, then a live animated waveform that polls the call until it ends
  and shows the transcript.
- **Frictionless first run** — no key configured yet? Any command (including
  bare `bolna`) offers to log you in right there instead of dead-ending on an
  error, then continues straight into what you asked for.
- **Scriptable everywhere** — every command works headless. `-o/--output
  table|json|csv` picks the format (table is the default, human-friendly
  one); `list` commands also take `-q/--quiet` to print bare IDs, one per
  line, for piping into `xargs` or other scripts. TTY auto-detection means
  piping/CI output is always plain text, never raw ANSI.
- **Interactive wizards** for the write operations that matter — `agents
  create`, `agents update` (with a before/after diff you must confirm),
  `agents delete` (type-the-name-to-confirm), and `call start` (shows the
  wallet balance and requires confirmation before it spends money).
- **`bolna doctor`** — an animated, spinner-driven checklist for config,
  keychain, network, and API key health.
- **Theming** — three built-in Lip Gloss palettes, all shades of blue by
  design (Bolna's own brand blue — the default — plus Tokyo Night and Nord;
  no violet/purple options), all `AdaptiveColor`-based so they look right in
  light or dark terminals. Cycle with `t` in the dashboard.
- **`--png` snapshot export** on `agents view`, via the
  [Freeze](https://github.com/charmbracelet/freeze) CLI, for dropping a
  styled agent card into Slack or docs.
- **OS keychain-backed auth** — API keys never touch a plaintext config file.
- **Lightweight** — a single static Go binary (~16MB stripped), no runtime,
  no separate interpreter.

## Install

**Via `go install`** (needs [Go](https://go.dev) installed):

```sh
go install github.com/bolna-ai/cli/cmd/bolna@latest
```

**Build from source:**

```sh
git clone https://github.com/bolna-ai/cli
cd cli
go build -o bolna ./cmd/bolna
```

Then move the `bolna` binary onto your `PATH` (e.g. `mv bolna /usr/local/bin/`).

If macOS warns `"bolna" cannot be opened because the developer cannot be
verified` the first time you run a binary you built or downloaded, either
right-click the file → **Open** → confirm once, or run:

```sh
xattr -d com.apple.quarantine bolna
```

## Usage

The fastest path is just:

```sh
bolna
```

First run, no key configured yet? It'll offer to log you in right there
(prompts for your API key, validates it, stores it in the OS keychain), then
drops you straight into the full-screen dashboard — no separate setup step
required.

Or, explicitly:

```sh
bolna login          # stores your API key in the OS keychain
bolna doctor          # sanity-check config, keychain, network, API key
bolna agents list      # or just `bolna` for the full dashboard
```

`BOLNA_API_KEY` always overrides the keychain — handy for CI, where the
inline login prompt is skipped automatically (non-TTY):

```sh
BOLNA_API_KEY=sk-... bolna agents list -o json
BOLNA_API_KEY=sk-... bolna agents list -q | xargs -I{} bolna agents view {}
```

### Commands

Run `bolna help` (or `bolna --help`) for the full grouped list with every
command's one-line description, or `bolna help <command>` / `bolna
<command> --help` for that command's own flags — `--help`/`-h` works
consistently at every level, including nested subcommands (e.g. `bolna
agents create --help`), and always shows that command's local flags
alongside the inherited global ones.

| Command | MCP tool equivalent | Notes |
|---|---|---|
| `bolna login` / `logout` / `whoami` | `get_user_info` | Huh-based login form, OS keychain storage |
| `bolna agents list` | `list_agents` | |
| `bolna agents view <id>` | `get_agent` | `--png` exports a styled snapshot |
| `bolna agents create` | `create_agent` | Interactive wizard, or `--file config.json` |
| `bolna agents update <id>` | `update_agent` | Shows a diff, requires confirmation |
| `bolna agents delete <id>` | `delete_agent` | Type-the-agent-name-to-confirm |
| `bolna call start <agent-id>` | `start_outbound_call` | Shows wallet balance, requires confirmation |
| `bolna calls list <agent-id>` | `list_agent_executions` | Defaults to the last 7 days |
| `bolna calls view <execution-id>` | `get_execution` | Transcript rendered via Glamour |
| `bolna numbers list` | `list_phone_numbers` | |
| `bolna batches list <agent-id>` | `list_batches` | |
| `bolna doctor` | — | Config/keychain/network/API-key health checks |
| `bolna completion [bash\|zsh\|fish\|powershell]` | — | Shell completions (via Cobra) |

Every command supports `-o/--output table|json|csv` (`--json` is a shorthand
for `-o json`), `--no-color`, `--profile <name>`, and `-v/--verbose`. Every
`list` command additionally supports `-q/--quiet` for bare-ID output.

### Dashboard keybindings

| Key | Action |
|---|---|
| `:` | Open the command palette (fuzzy jump to any agent or section) |
| `1` / `2` / `3` | Jump to Agents / Numbers / Account |
| `enter` | Drill into the selected row |
| `c` / `b` / `s` | From an agent's detail view: calls / batches / start a call |
| `esc` | Back one screen |
| `r` | Refresh the current screen |
| `t` | Cycle color theme |
| `q` / `ctrl+c` | Quit |

Browsing is read-only. Editing agents (create/update/delete) goes through the
dedicated `bolna agents` commands above, which have their own wizards and
confirmation flows — that stays out of the dashboard so a live "mission
control" view is never one stray keystroke away from a destructive edit.
Starting a call is the one write action built into the dashboard itself.

# bolna

The Bolna Voice AI CLI — and a full-screen terminal dashboard — for managing
agents, calls, phone numbers, and batches without leaving your terminal.

![demo](demo/demo.gif)

Every command mirrors a tool in [Bolna's MCP tool
list](https://www.bolna.ai/docs/build-with-ai/mcp-tool-list): agents, calls &
executions, phone numbers, batches, and account info. Run `bolna` with no
arguments in a terminal and you get a "mission control" dashboard instead of
a wall of JSON. Pipe it, script it, or run it in CI, and it behaves like a
normal, quiet, JSON-friendly CLI.

Built entirely on the [Charm](https://charm.land) stack — Bubble Tea, Bubbles,
Lip Gloss, Huh, Glamour, and Harmonica.

## Features

- **Full-screen dashboard** (`bolna`, no args) — bordered "mission control"
  frame (a steel-blue → periwinkle gradient border, matching the ASCII
  wordmark on mcp.bolna.ai) over agents, calls, numbers, batches, and
  account, with a spring-physics splash animation on startup (Harmonica).
- **Command palette** — press `:` anywhere in the dashboard for a fuzzy
  jump-to-agent/section launcher (Bubbles `list` with built-in filtering).
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
  light or dark terminals. Cycle
  with `t` in the dashboard.
- **`--png` snapshot export** on `agents view`, via the
  [Freeze](https://github.com/charmbracelet/freeze) CLI, for dropping a
  styled agent card into Slack or docs.
- **OS keychain-backed auth** — API keys never touch a plaintext config file.
- **Lightweight** — a single static Go binary (~16MB stripped), no runtime,
  no separate interpreter. `-o csv`/`-o json` use only the standard library —
  no extra dependency was added to support them.

## Install

```sh
go install github.com/bolna-ai/bolna-cli/cmd/bolna@latest
```

Or build from source:

```sh
git clone https://github.com/bolna-ai/bolna-cli
cd bolna-cli
go build -o bolna ./cmd/bolna
```

A Homebrew tap and signed/notarized macOS binaries ship via
[GoReleaser](https://goreleaser.com) — see [Releasing](#releasing) below.

## Quickstart

The fastest path is just:

```sh
bolna
```

First run, no key configured yet? It'll offer to log you in right there
(prompts for your API key, validates it, stores it in the OS keychain), then
drops you straight into the dashboard — no separate setup step required.

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

## Commands

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

## Dashboard keybindings

| Key | Action |
|---|---|
| `:` | Open the command palette (fuzzy jump to any agent or section) |
| `1` / `2` / `3` | Jump to Agents / Numbers / Account |
| `enter` | Drill into the selected row |
| `c` / `b` | From an agent's detail view: calls / batches |
| `esc` | Back one screen |
| `r` | Refresh the current screen |
| `t` | Cycle color theme |
| `q` / `ctrl+c` | Quit |

The dashboard is intentionally **read-only** — writes (create/update/delete
an agent, start a call) go through the dedicated commands above, which have
their own confirmation flows. This keeps a live "mission control" view from
ever being one stray keystroke away from a destructive action.

## Development

```sh
go build ./...
go vet ./...
go test ./...
gofmt -l .          # should print nothing
```

Project layout:

```
cmd/bolna/            entrypoint
internal/api/          Bolna REST API client (one file per resource group)
internal/auth/         OS keychain-backed credential storage
internal/config/       small on-disk settings file (theme, profiles)
internal/cli/          Cobra command tree
internal/tui/          Bubble Tea dashboard, doctor, splash, command palette
internal/tui/styles/   shared Lip Gloss theming
```

## Releasing

Tag pushes (`vX.Y.Z`) trigger `.github/workflows/release.yml`, which runs
[GoReleaser](https://goreleaser.com) (`.goreleaser.yaml`) to build signed,
notarized macOS binaries (universal amd64+arm64) plus Linux and Windows
builds, and cut a (draft) GitHub release.

### macOS notarization

Signing/notarization uses GoReleaser's `notarize.macos` pipe, which wraps
[quill](https://github.com/anchore/quill) — it runs on any CI runner (no
macOS host required) and works with free/OSS GoReleaser, unlike the Pro-only
native `codesign`/`notarytool` pipe.

You'll need, before the first real (non-snapshot) release:

1. A **paid Apple Developer Program membership**.
2. A **Developer ID Application** certificate + private key, exported as a
   `.p12` and base64-encoded → GitHub secret `MACOS_SIGN_P12`, plus its
   export password → `MACOS_SIGN_PASSWORD`.
3. An **App Store Connect API key** (Keys → Users and Access in App Store
   Connect): the Issuer ID → `MACOS_NOTARY_ISSUER_ID`, the Key ID →
   `MACOS_NOTARY_KEY_ID`, and the downloaded `.p8` file, base64-encoded →
   `MACOS_NOTARY_KEY`.

Until those secrets exist, `goreleaser release --clean` still runs and
produces unsigned binaries — the `enabled` field in `.goreleaser.yaml`'s
`notarize.macos` block is gated on `MACOS_SIGN_P12` being set. Test the whole
pipeline locally with:

```sh
goreleaser release --snapshot --clean
```

### Homebrew tap

`.goreleaser.yaml` has a commented-out `brews:` block. Uncomment it once a
`bolna-ai/homebrew-tap` repo exists, and add a `HOMEBREW_TAP_GITHUB_TOKEN`
secret (a GitHub token with write access to that repo).

## Demo GIF

`demo/demo.tape` is a [VHS](https://github.com/charmbracelet/vhs) script.
Regenerate `demo/demo.gif` with:

```sh
vhs demo/demo.tape
```

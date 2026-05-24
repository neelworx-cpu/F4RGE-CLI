# 4RGED

<p align="center">
    <strong>4RGED</strong><br />
    <a href="https://github.com/neelworx-cpu/F4RGE-CLI/releases"><img src="https://img.shields.io/github/release/neelworx-cpu/F4RGE-CLI" alt="Latest Release"></a>
    <a href="https://github.com/neelworx-cpu/F4RGE-CLI/actions"><img src="https://github.com/neelworx-cpu/F4RGE-CLI/actions/workflows/build.yml/badge.svg" alt="Build Status"></a>
</p>

<p align="center">Your F4RGE coding agent in the terminal.<br />Your tools, your code, and your workflows, governed by your F4RGE account.</p>

## Features

- **Managed Models:** use the models enabled for your F4RGE organization
- **Flexible:** switch allowed product models mid-session while preserving context
- **Session-Based:** maintain multiple work sessions and contexts per project
- **LSP-Enhanced:** 4RGED uses LSPs for additional context, just like you do
- **Extensible:** add capabilities via MCPs (`http`, `stdio`, and `sse`)
- **Works Everywhere:** first-class support in every terminal on macOS, Linux, Windows (PowerShell and WSL), Android, FreeBSD, OpenBSD, and NetBSD
- **Industrial Grade:** built on proven terminal UI foundations and tuned for F4RGE agentic workflows

## Installation

Install 4RGED with the F4RGE install script:

```bash
curl https://4rged.ai/install -fsS | bash
```

On Windows PowerShell:

```powershell
iwr https://4rged.ai/install.ps1 -useb | iex
```

The installer detects your OS and architecture, downloads the release artifact,
verifies its checksum, installs `4rged` into a user-writable location, and
prints the next command to run.

Or download a packaged release:

- [Packages][releases] are available in Debian and RPM formats once releases are published
- [Binaries][releases] are available for Linux, macOS, Windows, FreeBSD, OpenBSD, and NetBSD

[releases]: https://github.com/neelworx-cpu/F4RGE-CLI/releases

Release maintainers can generate installer artifacts with:

```bash
scripts/package-release.sh <version>
```

The script writes archives and `.sha256` files to `dist/cli/`, matching the
installer route naming convention.


> [!WARNING]
> Productivity may increase when using 4RGED and you may find yourself nerd
> sniped when first using the application.

## Getting Started

The customer path is managed by F4RGE. Users should install the CLI, sign in,
and start working. They should not need to choose raw providers, paste API keys,
or configure Azure/OpenAI/Anthropic/Gemini credentials locally.

```bash
4rged login
4rged status
4rged
```

Target first-run flow:

1. `4rged` detects that no F4RGE session is available.
2. The TUI shows a F4RGE sign-in prompt.
3. `4rged login` starts a browser/device login against F4RGE Auth.
4. F4RGE resolves the user, organization, entitlements, model catalog, and
   effective policy.
5. 4RGED starts chat with the managed default model, usually `Auto`.

Managed model families:

- `Auto` - F4RGE-managed routing, recommended.
- `GPT` - fast general coding and planning.
- `Claude` - deep reasoning and long-horizon edits.
- `Gemini` - large-context analysis.
- `4RGE 2.0` - F4RGE tuned agent model.
- `4RGE 1.5` - cost-efficient F4RGE model.

Useful setup checks:

```bash
4rged status
4rged doctor
4rged models
```

Provider keys, Azure deployments, and private model routes are configured by
admins in F4RGE. They are not configured in the customer CLI.

## Configuration

> [!TIP]
> 4RGED ships with a builtin `4rged-config` skill for configuring itself. In
> many cases you can simply ask 4RGED to configure itself.

4RGED runs great with no configuration. That said, if you do need or want to
customize 4RGED, configuration can be added either local to the project itself,
or globally, with the following priority:

1. `.4rged.json`
2. `4rged.json`
3. `$HOME/.config/4rged/4rged.json`

Configuration itself is stored as a JSON object:

```json
{
  "this-setting": { "this": "that" },
  "that-setting": ["ceci", "cela"]
}
```

As an additional note, 4RGED also stores ephemeral data, such as application
state, in one additional location:

```bash
# Unix
$HOME/.local/share/4rged/4rged.json

# Windows
%LOCALAPPDATA%\4rged\4rged.json
```

> [!TIP]
> You can override the user and data config locations by setting:
>
> - `F4RGED_GLOBAL_CONFIG`
> - `F4RGED_GLOBAL_DATA`

### LSPs

4RGED can use LSPs for additional context to help inform its decisions, just
like you would. LSPs can be added manually like so:

```json
{
  "$schema": "https://4rged.app/cli.json",
  "lsp": {
    "go": {
      "command": "gopls",
      "env": {
        "GOTOOLCHAIN": "go1.24.5"
      }
    },
    "typescript": {
      "command": "typescript-language-server",
      "args": ["--stdio"]
    },
    "nix": {
      "command": "nil"
    }
  }
}
```

### MCPs

4RGED also supports Model Context Protocol (MCP) servers through three transport
types: `stdio` for command-line servers, `http` for HTTP endpoints, and `sse`
for Server-Sent Events.

Shell-style value expansion (`$VAR`, `${VAR:-default}`, `$(command)`, quoting,
nesting) works in `command`, `args`, `env`, `headers`, and `url`, so
file-based secrets work out of the box. You can use values like `"$TOKEN"`
or `"$(cat /path/to/secret/token)"`. Expansion runs through 4RGED's embedded
shell, so the same syntax works on every supported system, Windows included.

Unset variables expand to the empty string by default, matching bash. For
required credentials, use `${VAR:?message}` so an unset variable fails loudly
at load time with `message` instead of silently resolving to empty:

```json
{ "api_key": "${CODEBERG_TOKEN:?set CODEBERG_TOKEN}" }
```

Headers (both MCP `headers` and provider `extra_headers`) whose value
resolves to the empty string are dropped from the outgoing request rather
than sent as `Header:`. That keeps optional env-gated headers like
`"OpenAI-Organization": "$OPENAI_ORG_ID"` clean when the variable is unset.

Provider `extra_body` is a non-expanding JSON passthrough; put env-driven
values in `extra_headers` or the provider's `api_key` / `base_url`, all of
which do expand.

> **Security note:** `4rged.json` is trusted code. Any `$(...)` in it runs at
> load time with your shell's privileges, before the UI appears. Don't launch
> 4RGED in a directory whose `4rged.json` you haven't reviewed.

```json
{
  "$schema": "https://4rged.app/cli.json",
  "mcp": {
    "filesystem": {
      "type": "stdio",
      "command": "node",
      "args": ["/path/to/mcp-server.js"],
      "timeout": 120,
      "disabled": false,
      "disabled_tools": ["some-tool-name"],
      "env": {
        "NODE_ENV": "production"
      }
    },
    "github": {
      "type": "http",
      "url": "https://api.githubcopilot.com/mcp/",
      "timeout": 120,
      "disabled": false,
      "disabled_tools": ["create_issue", "create_pull_request"],
      "headers": {
        "Authorization": "Bearer $GH_PAT"
      }
    },
    "streaming-service": {
      "type": "sse",
      "url": "https://example.com/mcp/sse",
      "timeout": 120,
      "disabled": false,
      "headers": {
        "API-Key": "$(echo $API_KEY)"
      }
    }
  }
}
```

### Hooks

4RGED has preliminary support for hooks. For details, see
[the hook guide](./docs/hooks/).

### Ignoring Files

4RGED respects `.gitignore` files by default, but you can also create a
`.4rgedignore` file to specify additional files and directories that 4RGED
should ignore. This is useful for excluding files that you want in version
control but don't want 4RGED to consider when providing context.

The `.4rgedignore` file uses the same syntax as `.gitignore` and can be placed
in the root of your project or in subdirectories.

### Allowing Tools

By default, 4RGED will ask you for permission before running tool calls. If
you'd like, you can allow tools to be executed without prompting you for
permissions. Use this with care.

```json
{
  "$schema": "https://4rged.app/cli.json",
  "permissions": {
    "allowed_tools": [
      "view",
      "ls",
      "grep",
      "edit",
      "mcp_context7_get-library-doc"
    ]
  }
}
```

You can also skip all permission prompts entirely by running 4RGED with the
`--yolo` flag. Be very, very careful with this feature.

### Disabling Built-In Tools

If you'd like to prevent 4RGED from using certain built-in tools entirely, you
can disable them via the `options.disabled_tools` list. Disabled tools are
completely hidden from the agent.

```json
{
  "$schema": "https://4rged.app/cli.json",
  "options": {
    "disabled_tools": ["bash", "sourcegraph"]
  }
}
```

To disable tools from MCP servers, see the [MCP config section](#mcps).

### Disabling Skills

If you'd like to prevent 4RGED from using certain skills entirely, you can
disable them via the `options.disabled_skills` list. Disabled skills are hidden
from the agent, including builtin skills and skills discovered from disk.

```json
{
  "$schema": "https://4rged.app/cli.json",
  "options": {
    "disabled_skills": ["4rged-config"]
  }
}
```

### Agent Skills

4RGED supports the [Agent Skills](https://agentskills.io) open standard for
extending agent capabilities with reusable skill packages. Skills are folders
containing a `SKILL.md` file with instructions that 4RGED can discover and
activate on demand.

The global paths we looks for skills are:

* `$F4RGED_SKILLS_DIR`
* `$XDG_CONFIG_HOME/agents/skills` or `~/.config/agents/skills/`
* `$XDG_CONFIG_HOME/4rged/skills` or `~/.config/4rged/skills/`
* `~/.agents/skills/`
* `~/.claude/skills/`
* On Windows, we _also_ look at
  * `%LOCALAPPDATA%\agents\skills\` or `%USERPROFILE%\AppData\Local\agents\skills\`
  * `%LOCALAPPDATA%\4rged\skills\` or `%USERPROFILE%\AppData\Local\4rged\skills\`
* Additional paths configured via `options.skills_paths`

On top of that, we _also_ load skills in your project from the following
relative paths:

* `.agents/skills`
* `.4rged/skills`
* `.claude/skills`
* `.cursor/skills`

```jsonc
{
  "$schema": "https://4rged.app/cli.json",
  "options": {
    "skills_paths": [
      "~/.config/4rged/skills", // Windows: "%LOCALAPPDATA%\\4rged\\skills",
      "./project-skills",
    ],
  },
}
```

You can get started with example skills from [anthropics/skills](https://github.com/anthropics/skills):

```bash
# Unix
mkdir -p ~/.config/4rged/skills
cd ~/.config/4rged/skills
git clone https://github.com/anthropics/skills.git _temp
mv _temp/skills/* . && rm -rf _temp
```

```powershell
# Windows (PowerShell)
mkdir -Force "$env:LOCALAPPDATA\4rged\skills"
cd "$env:LOCALAPPDATA\4rged\skills"
git clone https://github.com/anthropics/skills.git _temp
mv _temp/skills/* . ; rm -r -force _temp
```

#### User-Invocable Skills

Skills can be made invocable as commands from the commands palette (Ctrl+P). Add `user-invocable: true` to the skill's YAML frontmatter:

```yaml
---
name: my-skill
description: A skill that can be invoked as a command.
user-invocable: true
---
```

User-invocable skills appear in the commands palette with a `user:` or `project:` prefix:
- Skills from global directories show as `user:skill-name`
- Skills from project directories show as `project:skill-name`

When invoked, the skill's instructions are loaded into the conversation context.

To prevent the model from auto-triggering a skill (while still allowing user invocation), add `disable-model-invocation: true`:

```yaml
---
name: my-skill
description: Only invocable by users, not the model.
user-invocable: true
disable-model-invocation: true
---
```

Skills with `disable-model-invocation` won't appear in the model's available skills list but can still be invoked manually by users.

### Desktop notifications

4RGED sends desktop notifications when a tool call requires permission and when
the agent finishes its turn. They're only sent when the terminal window isn't
focused _and_ your terminal supports reporting the focus state.

```jsonc
{
  "$schema": "https://4rged.app/cli.json",
  "options": {
    "disable_notifications": false, // default
  },
}
```

To disable desktop notifications, set `disable_notifications` to `true` in your
configuration. On macOS, notifications currently lack icons due to platform
limitations.

### Initialization

When you initialize a project, 4RGED analyzes your codebase and creates
a context file that helps it work more effectively in future sessions.
By default, this file is named `AGENTS.md`, but you can customize the
name and location with the `initialize_as` option:

```json
{
  "$schema": "https://4rged.app/cli.json",
  "options": {
    "initialize_as": "AGENTS.md"
  }
}
```

This is useful if you prefer a different naming convention or want to
place the file in a specific directory (e.g., `4RGED.md` or
`docs/LLMs.md`). 4RGED will fill the file with project-specific context
like build commands, code patterns, and conventions it discovered during
initialization.

### Attribution Settings

By default, 4RGED adds attribution information to Git commits and pull requests
it creates. You can customize this behavior with the `attribution` option:

```json
{
  "$schema": "https://4rged.app/cli.json",
  "options": {
    "attribution": {
      "trailer_style": "co-authored-by",
      "generated_with": true
    }
  }
}
```

- `trailer_style`: Controls the attribution trailer added to commit messages
  (default: `assisted-by`)
  - `assisted-by`: Adds `Assisted-by: 4RGED:[ModelID]` as specified in [the convention](https://docs.kernel.org/process/coding-assistants.html#attribution)
  - `co-authored-by`: Adds `Co-Authored-By: 4RGED <4rged@4rged.app>`
  - `none`: No attribution trailer
- `generated_with`: When true (default), adds `💘 Generated with 4RGED` line to
  commit messages and PR descriptions

### Custom Providers

4RGED supports custom provider configurations for both OpenAI-compatible and
Anthropic-compatible APIs.

> [!NOTE]
> Note that we support two "types" for OpenAI. Make sure to choose the right one
> to ensure the best experience!
>
> - `openai` should be used when proxying or routing requests through OpenAI.
> - `openai-compat` should be used when using non-OpenAI providers that have OpenAI-compatible APIs.

#### OpenAI-Compatible APIs

Here’s an example configuration for Deepseek, which uses an OpenAI-compatible
API. Don't forget to set `DEEPSEEK_API_KEY` in your environment.

```json
{
  "$schema": "https://4rged.app/cli.json",
  "providers": {
    "deepseek": {
      "type": "openai-compat",
      "base_url": "https://api.deepseek.com/v1",
      "api_key": "$DEEPSEEK_API_KEY",
      "models": [
        {
          "id": "deepseek-chat",
          "name": "Deepseek V3",
          "cost_per_1m_in": 0.27,
          "cost_per_1m_out": 1.1,
          "cost_per_1m_in_cached": 0.07,
          "cost_per_1m_out_cached": 1.1,
          "context_window": 64000,
          "default_max_tokens": 5000
        }
      ]
    }
  }
}
```

#### Anthropic-Compatible APIs

Custom Anthropic-compatible providers follow this format:

```json
{
  "$schema": "https://4rged.app/cli.json",
  "providers": {
    "custom-anthropic": {
      "type": "anthropic",
      "base_url": "https://api.anthropic.com/v1",
      "api_key": "$ANTHROPIC_API_KEY",
      "extra_headers": {
        "anthropic-version": "2023-06-01"
      },
      "models": [
        {
          "id": "claude-sonnet-4-20250514",
          "name": "Claude Sonnet 4",
          "cost_per_1m_in": 3,
          "cost_per_1m_out": 15,
          "cost_per_1m_in_cached": 3.75,
          "cost_per_1m_out_cached": 0.3,
          "context_window": 200000,
          "default_max_tokens": 50000,
          "can_reason": true,
          "supports_attachments": true
        }
      ]
    }
  }
}
```

### Amazon Bedrock

4RGED currently supports running Anthropic models through Bedrock, with caching disabled.

- A Bedrock provider will appear once you have AWS configured, i.e. `aws configure`
- 4RGED also expects the `AWS_REGION` or `AWS_DEFAULT_REGION` to be set
- To use a specific AWS profile set `AWS_PROFILE` in your environment, i.e. `AWS_PROFILE=myprofile 4rged`
- Alternatively to `aws configure`, you can also just set `AWS_BEARER_TOKEN_BEDROCK`

### Vertex AI Platform

Vertex AI will appear in the list of available providers when `VERTEXAI_PROJECT` and `VERTEXAI_LOCATION` are set. You will also need to be authenticated:

```bash
gcloud auth application-default login
```

To add specific models to the configuration, configure as such:

```json
{
  "$schema": "https://4rged.app/cli.json",
  "providers": {
    "vertexai": {
      "models": [
        {
          "id": "claude-sonnet-4@20250514",
          "name": "VertexAI Sonnet 4",
          "cost_per_1m_in": 3,
          "cost_per_1m_out": 15,
          "cost_per_1m_in_cached": 3.75,
          "cost_per_1m_out_cached": 0.3,
          "context_window": 200000,
          "default_max_tokens": 50000,
          "can_reason": true,
          "supports_attachments": true
        }
      ]
    }
  }
}
```

### Local Models

Local models can also be configured via OpenAI-compatible API. Here are two common examples:

#### Ollama

```json
{
  "providers": {
    "ollama": {
      "name": "Ollama",
      "base_url": "http://localhost:11434/v1/",
      "type": "openai-compat",
      "models": [
        {
          "name": "Qwen 3 30B",
          "id": "qwen3:30b",
          "context_window": 256000,
          "default_max_tokens": 20000
        }
      ]
    }
  }
}
```

#### LM Studio

```json
{
  "providers": {
    "lmstudio": {
      "name": "LM Studio",
      "base_url": "http://localhost:1234/v1/",
      "type": "openai-compat",
      "models": [
        {
          "name": "Qwen 3 30B",
          "id": "qwen/qwen3-30b-a3b-2507",
          "context_window": 256000,
          "default_max_tokens": 20000
        }
      ]
    }
  }
}
```

## Logging

Sometimes you need to look at logs. Luckily, 4RGED logs all sorts of
stuff. Logs are stored in `./.4rged/logs/4rged.log` relative to the project.

The CLI also contains some helper commands to make perusing recent logs easier:

```bash
# Print the last 1000 lines
4rged logs

# Print the last 500 lines
4rged logs --tail 500

# Follow logs in real time
4rged logs --follow
```

Want more logging? Run `4rged` with the `--debug` flag, or enable it in the
config:

```json
{
  "$schema": "https://4rged.app/cli.json",
  "options": {
    "debug": true,
    "debug_lsp": true
  }
}
```

## Provider Auto-Updates

By default, 4RGED automatically checks for the latest and greatest list of
providers and models from [Catwalk](https://github.com/charmbracelet/catwalk),
the open source 4RGED provider database. This means that when new providers and
models are available, or when model metadata changes, 4RGED automatically
updates your local configuration.

### Disabling automatic provider updates

For those with restricted internet access, or those who prefer to work in
air-gapped environments, this might not be want you want, and this feature can
be disabled.

To disable automatic provider updates, set `disable_provider_auto_update` into
your `4rged.json` config:

```json
{
  "$schema": "https://4rged.app/cli.json",
  "options": {
    "disable_provider_auto_update": true
  }
}
```

Or set the `F4RGED_DISABLE_PROVIDER_AUTO_UPDATE` environment variable:

```bash
export F4RGED_DISABLE_PROVIDER_AUTO_UPDATE=1
```

### Manually updating providers

Manually updating providers is possible with the `4rged update-providers`
command:

```bash
# Update providers remotely from Catwalk.
4rged update-providers

# Update providers from a custom Catwalk base URL.
4rged update-providers https://example.com/

# Update providers from a local file.
4rged update-providers /path/to/local-providers.json

# Reset providers to the embedded version, embedded at 4rged at build time.
4rged update-providers embedded

# For more info:
4rged update-providers --help
```

## Metrics

4RGED records pseudonymous usage metrics (tied to a device-specific hash),
which maintainers rely on to inform development and support priorities. The
metrics include solely usage metadata; prompts and responses are NEVER
collected.

Details on exactly what’s collected are in the source code ([here](https://github.com/neelworx-cpu/F4RGE-CLI/tree/main/internal/event)
and [here](https://github.com/neelworx-cpu/F4RGE-CLI/blob/main/internal/llm/agent/event.go)).

You can opt out of metrics collection at any time by setting the environment
variable by setting the following in your environment:

```bash
export F4RGED_DISABLE_METRICS=1
```

Or by setting the following in your config:

```json
{
  "options": {
    "disable_metrics": true
  }
}
```

4RGED also respects the [`DO_NOT_TRACK`](https://donottrack.sh/) convention
which can be enabled via `export DO_NOT_TRACK=1`.

## Q&A

### Why is clipboard copy and paste not working?

Installing an extra tool might be needed on Unix-like environments.

| Environment         | Tool                     |
| ------------------- | ------------------------ |
| Windows             | Native support           |
| macOS               | Native support           |
| Linux/BSD + Wayland | `wl-copy` and `wl-paste` |
| Linux/BSD + X11     | `xclip` or `xsel`        |

## Contributing

See the [contributing guide](https://github.com/neelworx-cpu/F4RGE-CLI?tab=contributing-ov-file#contributing).

## Whatcha Think?

4RGED is part of the F4RGE agentic developer platform. Follow the project and file issues in the [F4RGE-CLI repository](https://github.com/neelworx-cpu/F4RGE-CLI).

## License

[FSL-1.1-MIT](https://github.com/neelworx-cpu/F4RGE-CLI/raw/main/LICENSE.md)

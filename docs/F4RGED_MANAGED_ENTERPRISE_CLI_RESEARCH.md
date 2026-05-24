# 4RGED Managed Enterprise CLI Research

This is the working research and product-direction document for turning `4rged` from a user-configured, BYOK terminal agent into a managed F4RGE enterprise product.

## Executive Direction

`4rged` should not ship as a product where customers pick arbitrary providers, paste API keys, and route directly to OpenAI, Anthropic, Gemini, Azure, Bedrock, or community OpenAI-compatible endpoints.

The enterprise product should work like this:

1. A user installs `4rged` with one command.
2. The first run asks them to sign in to F4RGE.
3. F4RGE authenticates the user, resolves their organization, policy, entitlements, and device session.
4. The CLI receives a short-lived F4RGE session token and a signed/cached policy snapshot.
5. Model access is curated by F4RGE Web, not by local config.
6. Model calls go through a F4RGE-managed gateway/runtime layer where provider keys, Azure deployments, routing, usage metering, budgets, audit, and fallback behavior are controlled centrally.

The local CLI remains the terminal execution edge: repository access, shell commands, file edits, MCP tools, local state, and user approvals. F4RGE Web becomes the authority: identity, model catalog, provider routing, policy, prompt publication, session lifecycle, billing, usage, traces, and enterprise administration.

## What Comparable Products Do

### Cursor CLI

Cursor's CLI pattern is the clearest install/onboarding reference:

```bash
curl https://cursor.com/install -fsS | bash
agent login
agent status
agent
```

Important product lessons:

- Install is one command and defaults to an auto-updating binary in a user-writable location.
- Interactive users authenticate through a browser login flow.
- `status`/`whoami` tells the user which account and endpoint they are using.
- API keys exist mainly for automation, not normal interactive onboarding.
- The CLI uses the same account/subscription/model quota as the larger product.

### Claude Code

Claude Code's managed path is subscription-first:

- Users install the CLI and run `claude`.
- First interactive use opens a browser login.
- Teams and Enterprise users authenticate with accounts invited by their org admin.
- Enterprise features are attached to the account/org: SSO, domain capture, RBAC, compliance, managed settings, and policy.
- Provider/cloud-provider modes exist, but the clean product path is not "paste your Anthropic key"; it is "log into the managed service."

Important product lesson: a serious enterprise CLI treats model access as an entitlement and policy decision, not as user-owned local key setup.

### GitHub Copilot CLI

Copilot CLI uses GitHub identity and org policy:

- Interactive auth uses OAuth device flow.
- Enterprise/GHE users can authenticate against a hostname.
- If an org disables the CLI or the user lacks a license, access is denied.
- It can reuse existing GitHub CLI credentials, but explicit CLI login remains available.
- Model selection is presented as a product capability under Copilot entitlement, not provider-key setup.

Important product lesson: org policy gates should be enforced before the agent runs, and a blocked user should get a clear "your org/admin/license disabled this" message.

### Gemini CLI

Gemini CLI exposes three modes:

- Login with Google for the simplest user flow.
- Gemini API key for developer/BYOK usage.
- Vertex AI for enterprise cloud use.

Important product lesson: Gemini supports BYOK/cloud-provider paths because it is a platform tool, but the F4RGE enterprise product should intentionally choose the managed path. If we keep any BYOK-like behavior, it should be an internal/developer build or clearly unsupported customer escape hatch, not the default product experience.

## Current 4RGED State To Retire

The current fork still has the original open-provider shape:

- `README.md` says users should get an API key, start `4rged`, and enter it interactively.
- `README.md` documents many provider environment variables, including `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`, `AZURE_OPENAI_API_KEY`, Bedrock credentials, OpenRouter, Groq, and others.
- `internal/config/provider.go` loads providers from Catwalk/embedded provider catalogs and supports provider auto-update.
- `internal/config/load.go` merges known providers with local provider overrides, environment-expanded API keys, base URLs, headers, model lists, Azure settings, Vertex settings, and custom providers.
- `internal/ui/dialog/models.go` opens a "Switch Model" dialog where users select a provider/model.
- `internal/ui/dialog/api_key_input.go` asks users to "Enter your API key...", verifies it against the provider, and writes it to global configuration.
- `internal/ui/dialog/oauth.go` and `internal/cmd/login.go` still have platform-specific auth for Hyper and Copilot.
- `internal/cmd/models.go` lists enabled models from configured local providers.
- The TUI exposes model selection with `ctrl+m` and command-center entries.

That behavior is useful for an open-source or power-user terminal tool, but it is wrong for a managed enterprise F4RGE product. Customers should not need to understand provider families, deployment names, API base URLs, headers, token formats, or which model is available through which cloud route.

## Target Product Shape

### One-Line Install

Target customer install:

```bash
curl https://4rged.ai/install -fsS | bash
```

Alternative domains can be decided later, but the command should read like a first-party product command, not a GitHub release workaround.

Recommended install behavior:

- Detect OS and architecture.
- Download a signed release artifact.
- Verify checksum/signature.
- Install into `~/.local/bin/4rged` or a platform-appropriate user-writable location.
- Add PATH instructions when needed.
- Print next steps:

```text
4RGED installed.
Run: 4rged login
Then: 4rged
```

Optional future convenience:

```bash
curl https://4rged.ai/install -fsS | bash && 4rged login
```

### First-Run Flow

Target first-run behavior:

```text
$ 4rged

Welcome to 4RGED.
You need to sign in to continue.

Press Enter to open your browser, or copy this code:
ABCD-EFGH

https://auth.4rged.ai/device
```

After login:

```text
Signed in as neel@company.com
Organization: Acme Corp
Plan: Enterprise
Models: GPT, Claude, Gemini, 4RGE 2.0, 4RGE 1.5
Policy: Acme default developer policy
```

Then the CLI starts directly in chat. It should not immediately ask "choose provider/model."

### Login/Auth Model

Recommended authentication model:

- Interactive terminal: OAuth 2.0 device authorization flow.
- Browser-capable local environment: open browser automatically after showing the device code.
- Headless/SSH environment: show copyable code and URL.
- CI/automation: service token or scoped machine token, not a provider key.

Token storage:

- macOS: Keychain.
- Linux: Secret Service/libsecret where available, encrypted file fallback with strict permissions.
- Windows: Credential Manager.
- Store refresh/session material as F4RGE credentials only.
- Never store OpenAI, Anthropic, Gemini, or Azure keys on the customer machine.

Session contract should reuse or extend the F4RGE Web desktop session shape:

- `deviceId`
- `deviceLabel`
- `platform`
- `appVersion`
- `clientKind: "customer"`
- entitlements such as `desktop.models.read`, `desktop.policy.read`, `desktop.usage.write`, `desktop.traces.write`, `desktop.sessions.refresh`
- short-lived access token
- refresh/session token where appropriate
- revocation status
- policy version

Although this is a CLI, it should be treated as another F4RGE managed edge client beside Desktop. Avoid creating a totally separate identity system for `4rged`.

### Model Experience

Customers should see F4RGE product model names, not provider plumbing.

Suggested visible model families:

- `Auto` or `F4RGE Auto`
- `GPT`
- `Claude`
- `Gemini`
- `4RGE 2.0`
- `4RGE 1.5`

The backend can route those names to:

- Direct OpenAI models.
- Azure OpenAI deployments.
- Anthropic Messages API or enterprise provider route.
- Gemini route.
- F4RGE-owned Azure/OpenAI-backed deployments for `4RGE 2.0` and `4RGE 1.5`.

The local UI should not expose:

- provider API keys
- provider base URLs
- provider-specific headers
- Azure deployment names
- raw Catwalk provider IDs
- OpenAI-compatible custom provider setup
- Bedrock/Vertex credential configuration

The "model section" should become a managed model dialog:

- It shows only models allowed by effective org/user policy.
- It explains model roles in product language: "fast", "deep reasoning", "best for codebase-wide changes", "preview", "restricted".
- It can show availability, policy restrictions, and admin-managed defaults.
- It should not show "configured/unconfigured" provider states.
- It should not route the user into API key input.
- It should include a clear footer command/hint, for example: `ctrl+m models`, `ctrl+l login`, `ctrl+, settings`.

Recommended default:

- Keep an `Auto` mode as the normal enterprise default.
- Allow admins to set org/team/project defaults.
- Allow users to switch only among models their org allows.
- Persist the user preference as a F4RGE model catalog ID, not as `provider/model/api_key`.

### Current F4RGE Web Catalog Contract To Track

The CLI should align with the Web/Platform API work instead of creating a separate catalog shape.

Relevant current Web pieces:

- `packages/platform-contracts/src/index.ts` exports `F4rgeModelCatalogEntry`, `F4rgeModelAccessPolicy`, `F4rgeBudgetPolicy`, `F4rgeRepositoryPolicy`, and `F4rgeEffectivePolicy`.
- `apps/agents-api/src/app/api/models/route.ts` returns an effective catalog bundle for an authorized organization/team/project.
- `apps/agents-api/src/lib/runtime/modelCatalog.ts` seeds cloud/runtime models and resolves scoped model access through `getEffectiveModelCatalogBundle`.
- The current model endpoint maps bundle models to `id`, `provider`, `model`, `label`, `availability`, `capabilities`, `riskClass`, `metadata`, and `requestProfile`.

CLI-side alignment:

- Treat the Web model `id` as the persisted CLI model preference.
- Display `label`, not raw provider/model IDs.
- Use `availability`, `capabilities`, and `riskClass` for model dialog badges and disabled states.
- Cache the full effective catalog bundle, not just flattened model rows.
- Preserve `requestProfile` for gateway/runtime routing metadata, but do not expose provider plumbing to customers.
- Include `organizationId`, optional `teamId`, optional `projectId`, and `surface=cli` when the Web endpoint supports CLI scoping.
- Track `policy.version` or bundle version in `4rged status` and trace metadata.

Minimum CLI model-cache shape:

```json
{
  "organizationId": "org_...",
  "teamId": null,
  "projectId": null,
  "surface": "cli",
  "policyVersion": "policy_...",
  "catalogVersion": "catalog_...",
  "fetchedAt": "2026-05-23T00:00:00Z",
  "expiresAt": "2026-05-23T01:00:00Z",
  "models": [
    {
      "id": "4rge-2.0",
      "label": "4RGE 2.0",
      "availability": "available",
      "capabilities": ["agent", "tools", "code", "reasoning"],
      "riskClass": "standard",
      "requestProfile": {
        "apiFamily": "azure.chat",
        "providerModelId": "Kimi-K2.6",
        "deploymentName": "Kimi-K2.6"
      }
    }
  ],
  "blockedReasonsByModel": {}
}
```

Open contract gap: current Web route uses `surface: "cloud"` internally. CLI should either receive a first-class `surface: "cli"` option or share a generic managed-edge surface that Desktop and CLI can both consume.

### Managed Provider Gateway

The provider keys should live in F4RGE infrastructure, not customer terminals.

Target route:

```text
4rged CLI
  -> F4RGE Agent Gateway / Runtime API
  -> F4RGE model router
  -> OpenAI / Azure OpenAI / Anthropic / Gemini
```

Gateway responsibilities:

- Validate CLI session token.
- Resolve organization, user, team, repository, and policy.
- Enforce model allow/deny lists.
- Enforce budget/rate/concurrency limits.
- Resolve product model IDs to provider deployments.
- Inject managed prompts/policies where required.
- Perform provider calls using F4RGE-owned credentials.
- Normalize streaming events into the `4rged`/F4RGE runtime event contract.
- Classify provider errors into customer-safe messages.
- Record usage and traces.
- Support provider failover and model rollout without shipping new CLI builds.

For privacy-sensitive local tool calls, the CLI still executes locally and streams tool results/events to the model through the gateway as needed. The gateway should never require raw repository upload beyond what the user asks the agent to send as context.

## Relationship To F4RGE Web And Desktop

The F4RGE Desktop architecture already points toward a split-control model:

- Desktop/local client owns execution edge.
- Web control plane owns authority.
- Model catalog and policy come from Web.
- Prompts and traces are centrally managed.

`4rged` should follow the same model:

- Share F4RGE Web auth, org membership, model catalog, policy, and usage contracts.
- Share provider gateway/runtime adapter logic where possible.
- Share model catalog IDs with F4RGE Web's cloud agent model catalog.
- Share Desktop session or create a generalized "client session" contract that supports both Desktop and CLI.
- Treat CLI traces and usage as first-class F4RGE usage events.

Relevant existing platform pieces:

- F4RGE Web has a model catalog concept with providers, model IDs, display names, capabilities, risk classes, and runtime metadata.
- F4RGE Web has policy concepts for allowed/denied model IDs, model groups, budgets, repositories, and capabilities.
- F4RGE Web has desktop session contracts with entitlements and short-lived tokens.
- F4RGE Desktop architecture already specifies signed policy snapshots, catalog snapshots, prompt snapshots, trace ingestion, and local execution edge.

## Existing 4RGED Agentic Architecture Inventory

This section maps the inherited Charm/4RGED architecture so new F4RGE capabilities can be planned without discarding the valuable local agent edge. The current CLI is not just a chat UI around an LLM. It is already a local agent runtime with sessions, persisted message streams, local tool execution, LSP/MCP context, permission gates, hooks, skills, sub-agents, non-interactive runs, and a client/server workspace boundary.

The managed enterprise work should therefore change who controls identity, model access, policy, prompts, traces, and provider routing. It should not replace the whole local runtime unless there is a specific security or product reason.

### Runtime Topology

Main entry and CLI command wiring live in:

- `main.go`
- `internal/cmd/root.go`
- `internal/cmd/run.go`
- `internal/cmd/server.go`
- `internal/workspace/workspace.go`
- `internal/workspace/app_workspace.go`
- `internal/workspace/client_workspace.go`
- `internal/app/app.go`

The important runtime shape is:

1. Cobra/Fang starts `4rged`.
2. `internal/cmd/root.go` resolves working directory, data directory, debug flags, yolo mode, and optional session continuation.
3. `setupWorkspace` chooses either an in-process workspace or a client/server workspace.
4. The in-process path creates `app.App`, which wires SQLite, config, sessions, messages, history, permissions, file tracking, LSP, MCP, skills, events, and the agent coordinator.
5. The client/server path uses `workspace.ClientWorkspace`, which exposes the same frontend interface through an HTTP/client protocol.
6. The Bubble Tea TUI consumes only the `workspace.Workspace` interface rather than directly depending on `app.App`.

This is one of the strongest existing seams for enterprise integration. F4RGE can add auth, policy, catalog refresh, gateway routing, and trace upload behind the workspace/app boundary while keeping the TUI mostly intact.

### Local Execution Edge

The CLI currently owns the execution edge:

- It runs in the user's repository.
- It reads and writes local files.
- It executes shell commands.
- It starts LSP servers.
- It starts and talks to MCP servers.
- It tracks local session history and file access.
- It asks the user for approval before sensitive tool actions, unless yolo/auto-approval applies.

For the managed product, this is still the right split:

- F4RGE Web should own identity, org, catalog, policy, provider routing, prompts, usage, billing, and traces.
- `4rged` should own local repository access, tool execution, local approvals, terminal UI state, and privacy-preserving context selection.

The CLI should not become a thin remote terminal unless F4RGE intentionally wants cloud-executed agents. For terminal-first coding, local execution is a product advantage.

### Application Services

`internal/app/app.go` is the service composition root. It creates and connects:

- `session.Service` for session CRUD and session lifecycle events.
- `message.Service` for persisted message streams.
- `history.Service` for file history.
- `permission.Service` for tool approval requests.
- `filetracker.Service` for files read or touched by a session.
- `lsp.Manager` for language server clients.
- `skills.Manager` for Agent Skills discovery state.
- `agent.Coordinator` for agent execution.
- `pubsub.Broker` instances for UI-visible events.
- MCP initialization and shutdown.
- update checks.

This composition root is the likely place to add managed services:

- F4RGE auth/session client.
- F4RGE model catalog client.
- F4RGE policy snapshot client.
- F4RGE gateway provider configuration.
- F4RGE trace/usage uploader.
- enterprise update and minimum-version checker.

### Session And Message Persistence

Sessions live in `internal/session/session.go`. Messages live in `internal/message/message.go`.

The session model already supports:

- top-level chat sessions
- child sessions for generated titles
- child sessions for delegated task agents
- title
- parent session ID
- message count
- prompt tokens
- completion tokens
- estimated usage marker
- cost
- summary message ID
- todos
- create/update/delete events

The message model already supports:

- user, assistant, and tool messages
- content parts
- reasoning content
- tool call records
- tool result records
- finish reasons
- provider/model metadata on assistant messages
- debounced streaming updates for efficient terminal rendering
- synchronous flushing when reads need the latest streamed state

Managed enterprise implications:

- Keep local SQLite for responsive terminal UX and offline draft/session browsing.
- Add F4RGE run IDs and remote trace IDs beside local session/message IDs.
- Add organization/user/project/repository metadata to usage and trace events.
- Do not upload whole local SQLite databases by default.
- Trace upload should be policy-driven and redacted.
- Empty draft sessions should not be persisted until they contain meaningful user content.

### Agent Coordinator

The coordinator lives in `internal/agent/coordinator.go`.

It is responsible for:

- building the active coder agent
- building the task sub-agent
- refreshing selected models before each run
- resolving provider config and model config
- merging provider/model call options
- refreshing OAuth tokens for providers that support it
- constructing tool lists from configured agent permissions
- filtering built-in tools by `AllowedTools`
- filtering MCP tools by `AllowedMCP`
- wiring hooks around top-level tools
- exposing run, cancel, queue, summarize, model, and busy-state methods

The default agent model currently has two configured model roles:

- `large` for the main coding agent
- `small` for title generation, summarization, and lightweight sub-agent work

Managed enterprise implications:

- Keep the large/small role distinction, but bind it to F4RGE catalog model roles rather than provider/model strings.
- Add a managed `Auto` model role that resolves server-side through the F4RGE model router.
- Keep `Coordinator.Run` as the main call path, but make `UpdateModels` read a signed catalog/policy snapshot.
- Replace provider-specific auth refresh with F4RGE session refresh.
- Keep tool filtering locally, but seed it from effective F4RGE policy.

### SessionAgent Loop

The main agent loop lives in `internal/agent/agent.go`.

Key runtime behavior:

- Rejects empty prompts and missing session IDs.
- Queues prompts when a session is already busy.
- Builds the final system prompt and appends MCP server instructions.
- Creates the user message before model execution.
- Creates assistant messages at step preparation time.
- Streams text deltas into persisted assistant messages.
- Streams reasoning/thinking content and signatures when providers support them.
- Tracks tool input start, tool call completion, and tool results.
- Converts provider/tool events into the internal message content model.
- Handles cancellation and provider errors.
- Generates a title for the first message.
- Tracks usage and cost into the session.
- Auto-summarizes when context is close to the model window.
- Detects repeated tool call loops and stops.
- Sends user notifications when turns finish or re-authentication is needed.

This is the core asset to preserve. A F4RGE gateway provider should adapt to this loop rather than force the CLI to learn a completely new orchestration model.

### Provider And Model Layer

The current model/provider system lives mainly in:

- `internal/config/config.go`
- `internal/config/load.go`
- `internal/config/provider.go`
- `internal/agent/coordinator.go`
- `internal/cmd/models.go`
- `internal/ui/dialog/models.go`
- `internal/ui/dialog/api_key_input.go`

Current behavior:

- Uses Catwalk provider metadata plus embedded provider data.
- Can auto-update provider catalogs.
- Supports direct provider API keys.
- Supports OpenAI, Anthropic, Gemini, Azure, Bedrock, Vertex, OpenRouter, Vercel AI Gateway, Copilot, Hyper, and OpenAI-compatible providers.
- Merges provider defaults, user provider config, model config, extra headers, extra body, and provider-specific options.
- Resolves shell-expanded environment variables in provider API keys, base URLs, and headers.
- Lets users select provider/model pairs locally.
- Lets users enter provider API keys locally.

Managed enterprise change:

- Replace the customer-facing provider catalog with a F4RGE model catalog.
- Keep provider capability metadata internally, but do not expose provider plumbing.
- Add a F4RGE gateway provider adapter that satisfies the existing Fantasy language model abstraction.
- Store selected model preference as a F4RGE product model ID or model role, not `provider/model`.
- Remove or hide API key entry in customer builds.
- Remove or hide Catwalk provider update flows in customer builds.
- Keep local provider mode only for internal/dev builds if needed.

### Prompt And Context System

Prompt construction lives in:

- `internal/agent/prompts.go`
- `internal/agent/prompt/prompt.go`
- `internal/agent/templates/coder.md.tpl`
- `internal/agent/templates/task.md.tpl`
- `internal/agent/templates/initialize.md.tpl`

The prompt system injects:

- provider and model identifiers
- working directory
- platform
- date
- git status
- recent commits
- context files
- available Agent Skills XML
- configured project instructions

Default context file discovery includes Cursor, Claude, Gemini, AGENTS, and F4RGED naming variants. That means the CLI already supports project-level instruction files and repo-specific policy guidance.

Managed enterprise implications:

- F4RGE prompt snapshots should be layered into this prompt builder.
- Org/team/repository policy instructions should be injected as signed managed prompt fragments.
- User/project context files should remain local and transparent.
- Prompt version/channel should come from F4RGE Web policy: stable, beta, canary, internal.
- Prompt provenance should be included in trace metadata.

### Built-In Tools

Built-in tools are in `internal/agent/tools/`. The tool registry is assembled in `coordinator.buildTools`.

Important current tools:

- `bash`: execute shell commands in the workspace.
- `job_output` and `job_kill`: interact with background jobs.
- `view`: read files with optional LSP/filetracker integration.
- `edit`, `multiedit`, and `write`: modify files.
- `ls`, `glob`, `grep`, and `rg`: inspect the repository.
- `diagnostics`, `references`, and `lsp_restart`: use LSP state.
- `fetch`, `download`, `web_fetch`, `web_search`, and `sourcegraph`: retrieve external information.
- `todos`: manage session todos.
- `f4rged_info`: expose runtime, config, LSP, MCP, and skill context to the model.
- `f4rged_logs`: inspect the CLI log file.
- `list_mcp_resources` and `read_mcp_resource`: expose MCP resources.
- MCP tools generated from configured MCP servers.
- `agent`: delegate a task to a sub-agent.
- `agentic_fetch`: run a small-model fetch/search sub-agent.

Each tool is a `fantasy.AgentTool`, so the agent loop does not need to know implementation details. This makes tool policy a natural enterprise control point.

Managed enterprise implications:

- Keep local tools local.
- Add policy metadata to every tool: side-effect class, risk class, network use, filesystem scope, and whether admin policy can auto-approve it.
- Add trace events for tool start, approval, denial, result, duration, and error.
- Add org policy controls for disabled tools, allowed MCP servers, network tools, shell commands, and write scopes.
- Keep tool descriptions product-quality because they shape model behavior.

### Permission System

Permissions live in `internal/permission/permission.go`.

Current behavior:

- Tools request permission with session ID, tool call ID, tool name, action, params, and path.
- The permission service publishes requests to the UI.
- The user can grant once, grant persistently for the session/path/action, or deny.
- Allowed tools can bypass prompts.
- Yolo mode skips all permission prompts.
- Non-interactive sessions can be auto-approved.
- PreToolUse hooks can pre-approve a specific tool call.

Managed enterprise implications:

- This should become the local enforcement layer for F4RGE tool policy.
- Org policy should be able to disable yolo mode, require approval for specific tool classes, block commands, restrict write paths, and disable network tools.
- Local user approval should still exist even when org policy allows a tool.
- Policy denials should produce clear messages and trace events.
- Persistent local approvals should be scoped and expire according to policy.

### Hooks

Hooks live in:

- `internal/hooks/hooks.go`
- `internal/hooks/runner.go`
- `internal/hooks/input.go`
- `internal/agent/hooked_tool.go`

Current behavior:

- User-configured shell commands can fire on `PreToolUse`.
- Hooks can match tool names with regex.
- Hooks run with structured environment variables and JSON stdin payloads.
- Hook results can allow, deny, halt the whole turn, rewrite tool input, or append context.
- Hook metadata is attached to tool responses for UI/audit visibility.
- Hooks run only around top-level tools, not inside delegated sub-agent internals.

Managed enterprise implications:

- Hooks are powerful and should be treated as trusted local code.
- Customer builds may keep hooks, but enterprise admins need controls.
- F4RGE policy should decide whether hooks are allowed, which hook sources are trusted, and whether hook decisions can override local permission prompts.
- Hook decisions should be included in trace events.
- Managed policy should be able to provide organization-approved hooks later, but not silently execute remote shell policy without user/admin visibility.

### MCP Integration

MCP integration lives in `internal/agent/tools/mcp/`.

Current behavior:

- Supports stdio, HTTP, and SSE MCP servers.
- Initializes configured MCP clients in the background.
- Tracks state: disabled, starting, connected, error.
- Lists tools, prompts, and resources.
- Converts MCP tools into `fantasy.AgentTool` entries.
- Adds MCP server instructions into the system prompt.
- Supports MCP resource list/read tools.
- Supports enabled/disabled tool allowlists per MCP server.
- Resolves command, args, env, URL, and headers through the same config resolver.

Managed enterprise implications:

- MCP should remain a local extension system.
- F4RGE policy should control allowed MCP server types, allowed MCP tool names, network endpoints, and secret handling.
- Managed enterprise can provide a curated MCP catalog later.
- Trace upload should include MCP server/tool identifiers, not raw secrets.

### LSP Integration

LSP integration lives in `internal/lsp/`.

Current behavior:

- Loads default language server definitions.
- Merges user-configured LSPs.
- Lazily starts an LSP when a file path needs it.
- Avoids auto-starting overly generic commands unless explicitly configured.
- Tracks server state and diagnostic counts.
- Provides diagnostics, references, and restart tools to the agent.
- Sends LSP state to the UI.

Managed enterprise implications:

- Keep LSP local.
- Use LSP state as local context, not as a cloud dependency.
- Allow policy to disable auto-LSP in locked-down environments.
- Use LSP diagnostics as structured trace/context metadata when policy allows.

### Agent Skills

Skills live in `internal/skills/`.

Current behavior:

- Implements the Agent Skills open standard with `SKILL.md`.
- Discovers built-in skills.
- Discovers user skills from configured paths.
- Validates YAML frontmatter and body.
- Deduplicates skills, with user skills able to override built-ins.
- Supports disabled skills.
- Exposes skill metadata to prompts as XML.
- Tracks skill discovery state for diagnostics and UI.

Managed enterprise implications:

- Skills can become a strong F4RGE capability channel.
- F4RGE Web can later distribute signed managed skills by org/team/project.
- Customer policy should control whether user-defined skills are allowed.
- Skill provenance should be visible: built-in, user local, org-managed, marketplace, internal.
- Managed skills should be signed and versioned.

### Sub-Agents

Sub-agent features live in:

- `internal/agent/agent_tool.go`
- `internal/agent/agentic_fetch_tool.go`
- `internal/agent/coordinator.go`
- `internal/session/session.go`

Current behavior:

- The `agent` tool delegates a task to a task agent.
- Delegated runs create child/task sessions.
- `agentic_fetch` creates a focused fetch/search sub-agent using the small model.
- Sub-agents can use a narrower tool set.
- Some sub-agent sessions are auto-approved when created for controlled workflows.

Managed enterprise implications:

- Preserve sub-agents because they are the foundation for future parallel agent capabilities.
- Add policy for whether agents can delegate, how many sub-agents may run, and which tools sub-agents may use.
- Add trace hierarchy: parent run, child run, tool call ID, child session ID.
- Use model catalog roles for sub-agent model selection rather than raw small model config.

### UI And Command Surface

The interactive frontend is the Bubble Tea TUI in `internal/ui/`, backed by `workspace.Workspace`.

Current product surfaces include:

- chat composer
- command center
- model dialog
- API key input dialog
- sessions dialog
- split chat panes
- sidebar with session/model/file/LSP/MCP details
- permission prompts
- diff and tool result rendering
- theme and compact mode controls

Managed enterprise implications:

- Keep the TUI shell, command center, sessions, split chat, sidebar, permissions, and local tool UX.
- Replace model/provider dialog content with a managed model catalog dialog.
- Replace API key input with login/status/policy flows.
- Add account status, org, policy freshness, gateway status, trace upload status, and model entitlement states.
- Keep split chat local-first, but eventually let each pane map to a distinct F4RGE run/session if cloud session sync is added.

### Non-Interactive And Automation Mode

Non-interactive mode lives in `internal/cmd/run.go` and `app.RunNonInteractive`.

Current behavior:

- `4rged run` accepts prompt args or stdin.
- Can continue by session ID or continue the last session.
- Can override large/small model from flags.
- Streams output to stdout.
- Shows progress/spinner on stderr when appropriate.
- Auto-approves permissions for the non-interactive session.
- Supports both in-process and client/server workspace modes.

Managed enterprise implications:

- CI/automation should use F4RGE service tokens or scoped machine tokens, not provider keys.
- `--model` should accept managed model IDs or product aliases only.
- Auto-approval in automation must be policy-controlled.
- Non-interactive runs should emit machine-readable trace/run IDs.
- F4RGE Web should distinguish human interactive sessions from automation sessions.

### Local Client/Server Mode

The CLI already has a local server mode:

- `internal/cmd/server.go`
- `internal/server/`
- `internal/client/`
- `internal/proto/`
- `internal/workspace/client_workspace.go`

Current behavior:

- `F4RGED_CLIENT_SERVER=1` makes the CLI connect to a server instead of running everything in-process.
- The server owns workspace/app execution.
- The client workspace proxies sessions, messages, agent runs, config changes, permissions, LSP, MCP, and events.
- The TUI can talk to either local app or server through the same workspace interface.

Managed enterprise implications:

- This can become a local daemon foundation for faster startup, shared auth/session state, background sync, and multi-terminal attachment.
- It is not the same as the F4RGE cloud gateway. The local server should still be treated as customer-machine execution.
- A future cloud session sync protocol can reuse some message/session shapes, but should not assume all local state is uploaded.

### Telemetry, Usage, And Cost

Telemetry and usage touch:

- `internal/agent/event.go`
- `internal/event/`
- `internal/session/session.go`
- `internal/agent/usage_fallback.go`

Current behavior:

- Emits prompt sent/responded events.
- Emits token usage events.
- Tracks provider, model, reasoning effort, thinking mode, and yolo mode.
- Stores prompt tokens, completion tokens, cost, and estimated usage on sessions.
- Falls back to estimating usage when provider metadata is incomplete.

Managed enterprise implications:

- Replace generic/product telemetry with F4RGE enterprise usage events.
- Usage should be attributed to org, user, team, repository, CLI version, model catalog ID, provider route, and gateway request ID.
- Cost should come from the gateway/router when possible, not from local provider assumptions.
- Local estimates can remain for immediate UI feedback, but billing must be server-authoritative.

### Update And Install Surface

The current CLI already has update concepts and release packaging. The managed product needs to harden them:

- one-line install script
- signed artifacts
- checksum verification
- minimum supported version policy
- enterprise kill switch
- channel support: stable, beta, canary, internal
- `4rged status` and `4rged doctor` checks for update and policy freshness

The existing provider auto-update path should not be confused with product updates. Customer builds should not need Catwalk provider updates once the F4RGE model catalog is authoritative.

### Current Capabilities To Preserve

Preserve these as first-class F4RGE CLI capabilities:

- local repository execution
- session/message persistence
- streaming TUI event model
- split chat/session UI
- permission prompts
- file edit/read/write tooling
- shell and job tooling
- LSP diagnostics/references
- MCP tools/resources/prompts
- Agent Skills
- sub-agent delegation
- context file ingestion
- auto-summarization
- loop detection
- non-interactive run mode
- local client/server workspace abstraction

These are the product foundations. The enterprise effort should make them governed and managed, not remove them.

### Current Surfaces To Retire Or Gate

Retire from customer-facing enterprise builds:

- arbitrary provider API key onboarding
- local provider picker as the primary model UX
- OpenAI/Anthropic/Gemini/Azure/Bedrock env var docs for customers
- API key input dialog
- provider auto-update as a customer concept
- provider-specific login commands such as Copilot and Hyper as first-run UX
- raw provider/model IDs in the default model dialog
- custom OpenAI-compatible provider setup in normal customer docs

Gate behind internal/dev mode if still needed:

- Catwalk provider development
- custom provider definitions
- local BYOK provider calls
- direct provider debugging
- provider-specific OAuth experiments

### Managed Integration Attachment Points

The lowest-risk enterprise integration plan is additive first:

1. Add F4RGE Auth as a new managed auth service and command set.
2. Add a signed model catalog and policy snapshot client.
3. Add a F4RGE gateway provider adapter that implements the existing model abstraction.
4. Bind `large`, `small`, and `auto` model roles to F4RGE catalog entries.
5. Replace customer model dialog rows with managed product models.
6. Replace API key onboarding with login-first onboarding.
7. Wrap local permission service with F4RGE policy checks.
8. Emit F4RGE trace and usage events from the existing agent/message/tool callbacks.
9. Hide local provider/BYOK flows in customer builds.
10. Move prompt version/channel control into F4RGE Web while keeping local context files.

This path lets the CLI adopt enterprise control without rewriting its TUI, tool runtime, session store, or agent loop.

### Planning Risks

Key risks to resolve before implementation:

- The current `Config.IsConfigured()` means "has enabled provider"; managed CLI needs "has valid F4RGE session and effective catalog."
- Provider/model IDs are deeply present in messages, events, model dialog, and non-interactive flags; migration needs a product model ID abstraction.
- Current usage cost is local/provider-derived; enterprise billing must be gateway-authoritative.
- Hooks and MCP can execute or connect to arbitrary local resources; enterprise policy must decide how much freedom customers get.
- Yolo and non-interactive auto-approval are useful but dangerous in enterprise contexts.
- Existing prompt construction is local-template based; managed prompt snapshots need provenance, versioning, and conflict rules with local context files.
- Local client/server mode is useful but not currently an authenticated enterprise daemon.
- Trace upload must be redacted and policy-bound so local code is not uploaded unexpectedly.

## Enterprise Requirements

### Identity And Access

Required:

- Email/password and OAuth login through F4RGE Auth.
- Enterprise SSO support.
- Domain capture or verified domain routing.
- Org invitation/member resolution.
- Role-based access.
- Device/session revocation.
- `4rged status`, `4rged logout`, and `4rged login --force`.
- Clear blocked states: no org, invite pending, license missing, policy denied, device revoked.

### Admin Controls

Required:

- Org/team model allow/deny lists.
- Default model or auto-router policy.
- Budget caps and rate limits.
- Repository policy.
- Tool approval policy.
- Trace upload policy.
- Prompt/version channel policy: stable, beta, canary, internal.
- Minimum CLI version policy.
- Emergency kill switch for compromised releases or provider incidents.

### Security

Required:

- No customer provider keys in local config.
- No raw provider credentials in logs, traces, SQLite, or crash reports.
- Signed update artifacts and checksums.
- Secure local credential storage.
- Policy snapshots signed by F4RGE and TTL-bound.
- Fail closed when policy is stale beyond enterprise TTL.
- Explicit local approval boundaries for shell/file side effects.
- Redaction before trace upload.

### Observability And Billing

Required:

- Model request usage events.
- Agent run events.
- Tool invocation events.
- Policy denial events.
- Estimated cost per org/team/project/user/model.
- Provider health and fallback observability.
- Customer-facing usage page in F4RGE Web.
- Internal operator view for incident debugging.

## Product Commands

Recommended public CLI commands:

```bash
4rged
4rged login
4rged logout
4rged status
4rged update
4rged models
4rged policy
4rged doctor
```

Command behavior:

- `4rged`: starts chat; if not authenticated, starts login.
- `4rged login`: device/browser flow against F4RGE Auth.
- `4rged logout`: removes local F4RGE tokens and optionally revokes the session.
- `4rged status`: account, org, endpoint, CLI version, policy version, token expiry, model default.
- `4rged update`: updates the CLI or prints install-manager instructions.
- `4rged models`: lists effective F4RGE models available to the signed-in user.
- `4rged policy`: shows effective local policy summary and freshness.
- `4rged doctor`: checks network, auth, policy, gateway, credential store, shell integration, and update status.

Commands to de-emphasize or remove from customer builds:

- arbitrary provider update commands
- provider API key entry flows
- provider-specific login modes such as `login copilot`
- custom OpenAI-compatible provider setup
- docs encouraging `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, `AZURE_OPENAI_API_KEY`, or Bedrock credentials

## UI Changes Required

### Onboarding

Current onboarding opens model/provider selection. Replace that with F4RGE login.

New onboarding order:

1. Welcome.
2. Login.
3. Organization/policy resolution.
4. Optional "choose default model mode" from allowed F4RGE product models, or default to `Auto`.
5. Start chat.

### Model Dialog

Current model dialog should be redesigned from:

```text
Provider group -> configured/unconfigured -> provider model rows -> API key or OAuth prompt
```

to:

```text
Managed models -> allowed model rows -> policy/status metadata -> selected product model
```

Suggested row examples:

```text
Auto            F4RGE-managed routing, recommended
GPT             Fast general coding and planning
Claude          Deep reasoning and long-horizon edits
Gemini          Large-context analysis
4RGE 2.0        F4RGE tuned agent model, enterprise default
4RGE 1.5        Cost-efficient F4RGE model
```

If a model is blocked:

```text
Claude          Disabled by Acme policy
```

Do not open API key input from this dialog.

### Command Center

The command center should include:

- Sign in / switch account
- Account status
- Managed models
- Effective policy
- Toggle thinking, if allowed
- Trace upload status
- Open F4RGE dashboard

The bottom composer command hint can keep model access visible as a product command, but it should say something like `ctrl+m models`, not imply provider setup.

## Architecture Options

### Option A: Full Remote Model Gateway

The CLI never calls providers directly. It streams model requests through F4RGE Gateway.

Pros:

- Best enterprise control.
- No local provider secrets.
- Centralized budget, audit, routing, model rollout, and failover.
- Simplest customer support story.

Cons:

- Requires robust streaming gateway.
- Requires careful privacy boundaries for local context.
- Requires high availability.

Recommendation: choose this as the enterprise target.

### Option B: F4RGE-Issued Ephemeral Provider Tokens

The CLI obtains short-lived provider-scoped credentials from F4RGE and calls providers directly.

Pros:

- Lower gateway bandwidth.
- Provider streaming stays closer to the client.

Cons:

- Secrets still reach customer machines.
- Harder audit/failover.
- Provider-specific complexity leaks back into the CLI.
- More difficult to revoke instantly.

Recommendation: avoid for customer enterprise builds unless a provider specifically requires it.

### Option C: Hybrid Internal/Developer Mode

Customer builds use the F4RGE gateway. Internal/dev builds can still enable local providers and BYOK behind a build tag or environment flag.

Pros:

- Keeps power-user/debug flexibility.
- Allows local provider adapter development.

Cons:

- Risk of leaking unsupported UI/docs into customer builds.

Recommendation: acceptable only if gated clearly as internal/development mode and absent from customer onboarding/docs.

## Implementation Roadmap

The roadmap should keep the existing local agent runtime and progressively replace the customer-facing BYOK/provider layer with F4RGE-managed control. Competitor-parity features should land after auth, model catalog, policy, and gateway primitives exist, otherwise the CLI will keep teaching users the wrong mental model.

### Phase 1: Managed Foundation

Goal: make first run feel like a F4RGE enterprise product instead of an open-provider terminal tool.

Core work:

- Add first-party `4rged login`, `4rged logout`, `4rged status`, and `4rged doctor`.
- Implement F4RGE Auth device/browser login.
- Register the CLI as a F4RGE client session with device ID, platform, version, and client kind.
- Store only F4RGE tokens locally through secure credential storage.
- Resolve user, organization, license, entitlements, and effective policy after login.
- Add a F4RGE model catalog client.
- Add signed catalog and policy snapshot caching.
- Replace first-run provider picker with login-first onboarding.
- Replace `4rged models` output with effective F4RGE product models.
- Replace the TUI model dialog with managed model rows: `Auto`, `GPT`, `Claude`, `Gemini`, `4RGE 2.0`, and `4RGE 1.5`.

Primary code areas:

- `internal/cmd/login.go`
- `internal/cmd/logout.go`
- `internal/cmd/models.go`
- `internal/config/`
- `internal/workspace/`
- `internal/ui/dialog/models.go`
- `internal/ui/dialog/api_key_input.go`

Exit criteria:

- A new user can install, run `4rged`, sign in, see their account/org/policy status, and start chat without entering a provider key.
- The model dialog no longer presents provider setup as the default path.

### Phase 2: F4RGE Gateway Adapter

Goal: route model calls through F4RGE while preserving the existing agent loop, tools, sessions, and TUI.

Core work:

- Add a F4RGE gateway provider adapter behind the existing model abstraction.
- Map product model IDs and roles to gateway route IDs.
- Preserve `large` and `small` internal model roles while sourcing them from the F4RGE catalog.
- Add `Auto` as the default managed routing mode.
- Route main agent, summarization, title generation, task sub-agent, and agentic fetch through gateway-backed catalog roles.
- Normalize gateway streaming into the existing assistant text, reasoning, tool call, and usage events.
- Move billing/cost authority to the F4RGE gateway.
- Convert provider and gateway errors into customer-safe messages.
- Keep direct providers only behind an internal/development path while the gateway stabilizes.

Primary code areas:

- `internal/agent/coordinator.go`
- `internal/agent/agent.go`
- `internal/config/provider.go`
- `internal/config/load.go`
- new F4RGE gateway/auth/catalog package, likely under `internal/f4rge/` or `internal/agent/f4rge/`

Exit criteria:

- Normal agent runs use F4RGE gateway by default.
- No OpenAI, Anthropic, Gemini, Azure, or Bedrock keys are required on customer machines.
- Usage and trace events include F4RGE model catalog IDs and gateway request IDs.

### Phase 3: Enterprise Policy Over Local Tools

Goal: keep local power, but govern it with F4RGE policy.

Core work:

- Add policy checks before local permission prompts.
- Add tool metadata: side-effect class, filesystem scope, network use, shell use, MCP use, and risk class.
- Enforce org/team/user policy for shell, file writes, MCP, web/network tools, hooks, yolo mode, and non-interactive auto-approval.
- Add command allow/deny lists for shell execution.
- Add path write/read restrictions where policy requires them.
- Add approval levels: read-only, ask-first, auto-safe, and admin-disabled-yolo.
- Emit policy denial events.
- Show local policy state in `4rged status`, `4rged doctor`, and the command center.

Primary code areas:

- `internal/permission/`
- `internal/agent/hooked_tool.go`
- `internal/agent/tools/`
- `internal/ui/dialog/commands.go`
- `internal/workspace/workspace.go`

Exit criteria:

- A policy-denied tool never reaches execution.
- Local approvals still exist, but they cannot override admin policy.
- Enterprise admins can disable dangerous modes and tool classes.

### Phase 4: Competitor-Parity UX

Goal: match expected modern agent CLI capabilities after the managed foundation exists.

Core work:

- Add explicit Ask/read-only mode.
- Add Plan/spec mode.
- Add Review mode for local diffs without touching the working tree.
- Add session fork and better session resume flows.
- Add richer `4rged run` automation outputs: text, JSON, stream JSON, and final response formats.
- Add service-token or machine-token automation auth.
- Add worktree/sandbox mode for isolated edits.
- Add checkpoints, revert, and rewind for local edits.
- Add slash/command-center entries for account, models, policy, traces, doctor, review, fork, and mode switching.
- Add better artifacts from runs: changed files, commands run, tests run, logs, screenshots when browser support exists.

Competitor capabilities this phase addresses:

- Cursor-style ask/plan/agent modes, non-interactive output formats, worktrees, and rules.
- Factory-style exec mode, session fork, missions groundwork, skills, hooks, plugins, and structured outputs.
- Windsurf-style approval levels, allow/deny command lists, workflows, and checkpoints.
- Claude/Codex-style review mode, resume/fork, remote/background sessions, and managed settings.
- Gemini-style plan mode, policy engine, checkpointing, MCP auth, and sandboxing.

Primary code areas:

- `internal/cmd/run.go`
- `internal/cmd/session.go`
- `internal/ui/model/`
- `internal/ui/dialog/commands.go`
- `internal/session/`
- `internal/message/`
- `internal/permission/`
- `internal/agent/coordinator.go`

Exit criteria:

- Users can choose mode intentionally: ask, plan, agent, review.
- Automation can consume structured outputs safely.
- Edits can be isolated or reverted.

### Phase 5: F4RGE-Plus Capabilities

Goal: build capabilities that feel native to F4RGE and go beyond simple competitor parity.

Core work:

- Make split sessions a first-class product capability, not just a local layout.
- Add local-to-cloud handoff: continue a CLI run in F4RGE Web or Cloud Agents.
- Add cloud-to-local handoff: open a cloud run locally when repository access or local tools are needed.
- Add policy-controlled cloud session sync.
- Add managed signed skills and workflows distributed from F4RGE Web.
- Add mission mode: orchestrator plus worker agents with visible task decomposition.
- Add F4RGE Web trace viewer for CLI sessions, tool calls, policy decisions, and usage.
- Add org model routing and budget views.
- Add prompt channels: stable, beta, canary, internal.
- Share session identity and client-session concepts with F4RGE Desktop where possible.

Primary code areas:

- CLI: `internal/session/`, `internal/message/`, `internal/agent/`, `internal/workspace/`, `internal/ui/model/`
- F4RGE Web: auth/session API, model catalog API, prompt control, trace ingestion, cloud agent runtime, dashboard/console UI
- F4RGE Desktop: shared client session and model catalog contracts where reusable

Exit criteria:

- A CLI session can be represented in F4RGE Web with policy, trace, usage, model, and run metadata.
- Managed skills/workflows can be assigned by org/team/project.
- Multi-agent/mission work has a product surface, not just hidden sub-agent calls.

### Phase 6: Customer Build Hardening

Goal: ship a managed enterprise CLI that is supportable, secure, and not confused with the open-provider fork.

Core work:

- Hide or remove BYOK/provider setup from customer builds.
- Hide provider auto-update/Catwalk flows from customer docs and command help.
- Keep provider/BYOK mode only behind internal/dev build flags if still needed.
- Add signed installer and update artifacts.
- Add checksum verification.
- Add minimum version policy.
- Add emergency kill switch.
- Add session revocation.
- Add policy TTL fail-closed behavior.
- Add redacted trace upload.
- Add proxy, custom CA, and mTLS support for enterprise networks.
- Add admin-managed settings.
- Add `4rged doctor` checks for auth, policy, gateway, credential store, proxy, update status, shell, MCP, and LSP.

Primary code areas:

- `internal/cmd/`
- `internal/update/`
- `internal/config/`
- `internal/event/`
- release/install scripts
- F4RGE Web admin and control plane APIs

Exit criteria:

- Customer builds expose only F4RGE-managed onboarding and model selection.
- Admin policy can block stale, revoked, or unsupported clients.
- Support can diagnose common enterprise environment failures through `4rged doctor`.

### Recommended Build Order

Build Phase 1 and Phase 2 first. They define the managed product identity. Then add policy enforcement in Phase 3 before adding higher-autonomy features in Phase 4 and Phase 5.

The first implementation milestone should be:

1. F4RGE Auth login/status/logout.
2. F4RGE model catalog client and signed snapshot cache.
3. Login-first onboarding.
4. Managed model dialog.
5. Gateway-backed default model calls.
6. BYOK/provider surfaces hidden from customer onboarding.

## Open Product Decisions

1. Public install domain: `4rged.ai/install`.
2. Should the binary stay `4rged`, or do we also provide a friendlier alias such as `f4rge`?
3. Should first run auto-start login, or should it print `Run 4rged login`?
4. Should the default visible model be `Auto`, `4RGE 2.0`, or admin-configured?
5. Are GPT/Claude/Gemini visible model families, or should the UI only expose F4RGE product tiers?
6. Should any BYOK/custom provider mode remain in the customer binary, or only in internal builds?
7. Does CLI session registration reuse the existing Desktop session contract or become a generalized F4RGE client session?
8. What offline behavior is acceptable when policy/model catalog cannot refresh?
9. Should traces upload by default for enterprise customers, or follow org policy with explicit local visibility?
10. What is the minimum enterprise auth surface for v1: email login, OAuth providers, or full SSO/domain capture?

## Recommended Near-Term Decision

Build the managed enterprise path, not a polished BYOK path.

The next engineering milestone should be:

1. Add F4RGE Auth login/status/logout to `4rged`.
2. Replace first-run model selection with login-first onboarding.
3. Add a F4RGE-managed model catalog client.
4. Replace the model dialog content with allowed F4RGE product models.
5. Keep existing provider/BYOK code temporarily behind a development/internal path while the gateway comes online.

This lets the UI stop teaching customers the wrong mental model before the full gateway is complete.

## Reference Links

- Cursor CLI installation: `https://cursor.com/docs/cli/installation`
- Cursor CLI authentication: `https://cursor.com/docs/cli/reference/authentication`
- Claude Code authentication: `https://code.claude.com/docs/en/authentication`
- Claude Code quickstart: `https://code.claude.com/docs/en/quickstart.md`
- Claude Code enterprise deployment overview: `https://code.claude.com/docs/en/third-party-integrations`
- GitHub Copilot CLI authentication: `https://docs.github.com/en/copilot/how-tos/copilot-cli/set-up-copilot-cli/authenticate-copilot-cli`
- Gemini CLI authentication: `https://google-gemini.github.io/gemini-cli/docs/get-started/authentication.html`

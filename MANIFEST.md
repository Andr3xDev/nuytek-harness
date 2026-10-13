# Config Manifest

Portable source for my agent setup across machines.

## Installs

- Claude Code agents: `agents/` -> `~/.claude/agents/`
- Claude Code commands: `commands/` -> `~/.claude/commands/`
- Claude Code skills: `skills/` -> `~/.claude/skills/`
- Claude hooks: `hooks/*.sh` -> `~/.claude/hooks/`
- Claude global instructions: `configs/CLAUDE.md` -> `~/.claude/CLAUDE.md`
- Claude settings/hooks reference: `configs/claude-settings.json` -> `~/.claude/settings.json`
- Claude MCP registry: `configs/claude-mcp.json` -> merged into `~/.claude.json#mcpServers`
- Claude helper docs: `configs/common-sdd.md`, `configs/engram-protocol.md`, `configs/context7.md` -> `~/.claude/`
- CodeGraph helper: `scripts/codegraph-health.sh` -> `~/.local/bin/codegraph-health`
- Out-of-context launcher: `scripts/claude-out-context.sh` -> `~/.local/bin/claude-out-context`
- OpenCode config: `opencode/` -> `~/.config/opencode/`

## Not Included

- Secrets: `GITHUB_TOKEN`, `ZENHUB_API_TOKEN`, `ZENHUB_WORKSPACE_ID`, Linear auth, Claude cloud connector auth
- Runtime data: Claude history, project transcripts, paste cache, jobs, stats
- Dependency caches: `node_modules`, plugin caches, OpenCode sessions/logs
- Machine-specific cloud connectors: disconnect/connect at `https://claude.ai/customize/connectors`

## Required Commands

- `claude`
- `opencode`
- `git` and `node` for the upstream Caveman installer; `npx` for local MCPs
- `engram`
- `codegraph`
- `openspec`

## Helper Scripts

- `codegraph-health`: shared CodeGraph health/maintenance helper for Claude Code and OpenCode agents.
- `claude-out-context`: launches Claude Code with memory and tracker persistence disabled.

## Expected MCPs

- `context7`: `npx -y @upstash/context7-mcp`
- `codegraph`: `codegraph serve --mcp`
- `engram`: `engram mcp`
- `filesystem`: `npx -y @modelcontextprotocol/server-filesystem /home/andrex/dev /home/andrex/laburo/`
- `github`: `npx -y @modelcontextprotocol/server-github` with `GITHUB_TOKEN`
- `linear-server`: `https://mcp.linear.app/mcp`
- `zenhub` (optional): `npx -y mcp-remote https://api.zenhub.com/mcp` with `ZENHUB_API_TOKEN`, `ZENHUB_WORKSPACE_ID`

## Expected OpenCode Plugins

- `caveman`: installed for Claude Code and OpenCode by the upstream installer from current `main`; it modifies user-level configuration

## Refresh

Run `./scripts/refresh-from-local.sh` on the source machine after changing live config.
Review diff before committing.

Refresh owns exactly `agents/`, `commands/`, `skills/`, `hooks/`, `opencode/`, and
`configs/{CLAUDE.md,claude-settings.json,common-sdd.md,engram-protocol.md,context7.md}`.
It mirrors these paths from local config and prunes stale managed files. `configs/claude-mcp.json`,
`configs/generic-config.md`, `configs/mcp-servers.md`, docs, and scripts are repository-owned because
they have no safe one-to-one local source; refresh preserves them.

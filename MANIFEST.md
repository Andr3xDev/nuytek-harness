# Config Manifest

Portable source for my agent setup across machines.

## Installs

- Claude Code agents: `agents/` -> `~/.claude/agents/`
- Claude Code commands: `claude/commands/` -> `~/.claude/commands/`
- Claude Code skills: `skills/` -> `~/.claude/skills/`
- Claude hooks: `claude/hooks/` -> `~/.claude/hooks/`
- Claude sounds: `claude/sounds/` -> `~/.claude/sounds/`
- Claude global instructions: `claude/CLAUDE.md` -> `~/.claude/CLAUDE.md`
- Claude settings/hooks reference: `claude/claude-settings.json` -> `~/.claude/settings.json`
- Claude MCP registry: `claude/claude-mcp.json` -> merged into `~/.claude.json#mcpServers`
- Shared helper docs: `shared/common-sdd.md`, `shared/engram-protocol.md` -> `~/.claude/`; `shared/rules/context7.md` -> `~/.claude/rules/context7.md`
- CodeGraph helper: `scripts/codegraph-health.sh` -> `~/.local/bin/codegraph-health`
- Out-of-context launcher: `claude/scripts/claude-out-context.sh` -> `~/.local/bin/claude-out-context`

## Not Included

- Secrets: `GITHUB_TOKEN`, `ZENHUB_API_TOKEN`, `ZENHUB_WORKSPACE_ID`, Linear auth, Claude cloud connector auth
- Runtime data: Claude history, project transcripts, paste cache, jobs, stats
- Dependency caches: `node_modules`, plugin caches
- Machine-specific cloud connectors: disconnect/connect at `https://claude.ai/customize/connectors`

## Required Commands

- `bash` >= 4
- `claude`
- `node` and the `claude` CLI for the plugins component; `npx` for local MCPs
- `engram`
- `codegraph`
- `openspec`

## Helper Scripts

- `codegraph-health`: shared CodeGraph health/maintenance helper for Claude Code agents.
- `claude-out-context`: launches Claude Code with memory and tracker persistence disabled.

## Expected MCPs

- `context7`: `npx -y @upstash/context7-mcp`
- `codegraph`: `codegraph serve --mcp`
- `engram`: `engram mcp`
- `filesystem`: `npx -y @modelcontextprotocol/server-filesystem /home/andrex/dev /home/andrex/laburo/`
- `github`: `npx -y @modelcontextprotocol/server-github` with `GITHUB_TOKEN`
- `linear-server`: `https://mcp.linear.app/mcp`
- `zenhub` (optional): `npx -y mcp-remote https://api.zenhub.com/mcp` with `ZENHUB_API_TOKEN`, `ZENHUB_WORKSPACE_ID`

## Expected Plugins

- `plugins`: every plugin in `enabledPlugins` and marketplace in `extraKnownMarketplaces` of `claude/claude-settings.json` is installed or updated through `claude plugin`; enable/disable state is left to `settings.json`

## Pull

Run `./scripts/nuytek-harness pull` on the source machine after changing live config.
Review diff before committing.

Pull owns exactly `agents/`, `skills/`, `claude/{commands,hooks,sounds}/`,
`claude/{CLAUDE.md,claude-settings.json}`, and
`shared/{common-sdd.md,engram-protocol.md,rules/context7.md}`.
It mirrors these paths from local config and prunes stale managed files. `claude/claude-mcp.json`,
`claude/mcp-servers.md`, `claude/scripts/`, docs, and scripts are repository-owned because
they have no safe one-to-one local source; pull preserves them.

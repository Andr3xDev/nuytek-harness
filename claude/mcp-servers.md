# MCP Servers

MCP servers configured in this environment. Secrets/tokens omitted.

| Server | Type | Command / URL | Purpose |
|--------|------|----------------|---------|
| `engram` | stdio | `engram mcp` | Persistent memory across sessions (decisions, bugs, conventions) |
| `codegraph` | stdio | `codegraph serve --mcp` | Symbol/call-graph index per project (`.codegraph/`) |
| `context7` | stdio | `npx -y @upstash/context7-mcp` | Up-to-date library/framework docs |
| `github` | stdio | `npx -y @modelcontextprotocol/server-github` | Issues, PRs, repo search |
| `linear-server` | http | `https://mcp.linear.app/mcp` | Linear issues/projects/initiatives |
| `filesystem` | stdio | `npx -y @modelcontextprotocol/server-filesystem <dirs>` | Sandboxed filesystem access outside project root |

## Plugins (marketplace-installed)

| Plugin | Repo | Purpose |
|--------|------|---------|
| `caveman` | `juliusbrussee/caveman` | Ultra-compressed communication mode |
| `ponytail` | `DietrichGebert/ponytail` | Forces simplest working code (YAGNI ladder) |
| `engram` | `Gentleman-Programming/engram` | Persistent memory MCP + hooks |
| `claude-hud` | `jarrodwatts/claude-hud` | Statusline |

Both `caveman` and `ponytail` default to `full` intensity, pinned via:
- `~/.config/caveman/config.json`
- `~/.config/ponytail/config.json`

They operate on independent axes and are meant to run together: caveman governs *how the agent talks*, ponytail governs *how the agent codes*.

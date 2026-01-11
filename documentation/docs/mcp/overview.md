# MCP Integration Overview

m9m includes a built-in MCP (Model Context Protocol) server that enables **Claude Code** to orchestrate workflows using natural language conversations.

## What is MCP?

MCP (Model Context Protocol) is a protocol that allows AI assistants like Claude to interact with external tools and services. The m9m MCP server exposes workflow automation capabilities as tools that Claude can use.

## Key Capabilities

| Capability | Description |
|------------|-------------|
| **Natural Language Workflows** | Create workflows by describing what you want |
| **Quick Actions** | Execute HTTP requests, send messages, call AI APIs |
| **Full CRUD** | Create, read, update, delete workflows |
| **Debugging** | Investigate failed executions with detailed logs |
| **Custom Nodes** | Create JavaScript or REST-based plugins on the fly |

## Quick Start

### 1. Build the MCP Server

```bash
cd /path/to/m9m
go build -o mcp-server ./cmd/mcp-server
```

### 2. Configure Claude Code

Add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "m9m": {
      "command": "/path/to/mcp-server",
      "args": ["--data", "./data"]
    }
  }
}
```

### 3. Restart Claude Code

Restart the Claude Code application to load the new MCP server.

### 4. Start Automating

```
You: "Send a Slack message to #general saying hello"
Claude: [Uses the send_slack tool to send the message]
        Message sent successfully to #general!
```

## Server Modes

### Local Mode (Default)

Runs m9m locally with persistent storage:

=== "SQLite (Default)"
    ```bash
    ./mcp-server
    # Data stored in ./data/m9m.db
    ```

=== "PostgreSQL"
    ```bash
    ./mcp-server --postgres "postgres://user:pass@localhost/m9m"
    ```

### Cloud Mode

Connects to a remote m9m instance:

```bash
./mcp-server --api-url https://m9m.example.com
```

## Architecture

```mermaid
graph TB
    CC[Claude Code] -->|JSON-RPC| MCP[MCP Server]
    MCP --> TM{Mode?}
    TM -->|Local| LE[Local Engine]
    TM -->|Cloud| RC[Remote Client]
    LE --> DB[(SQLite/Postgres)]
    RC -->|HTTP| API[m9m API Server]
```

## Next Steps

- [Available Tools](tools.md) - Complete reference of all 37 MCP tools
- [Custom Plugins](plugins.md) - Create JavaScript and REST plugins
- [Examples](examples.md) - Real-world workflow examples

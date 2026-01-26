# CLI Nodes

CLI nodes enable secure execution of command-line tools within workflows. The primary use case is running **CLI-based AI agents** like OpenAI Codex CLI, Claude Code, Aider, and similar tools in a sandboxed environment.

## Overview

Modern AI coding assistants increasingly ship as CLI tools that can:
- Analyze and modify codebases
- Generate code from natural language
- Debug and fix issues
- Run automated refactoring

m9m's CLI Execute node allows you to orchestrate these tools within workflows while maintaining security through Linux sandboxing.

---

## CLI Execute Node

Execute any CLI command with configurable sandboxing, isolation, and resource limits.

### Type

```
n8n-nodes-base.cliExecute
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `command` | string | Yes | - | The command to execute |
| `args` | array | No | `[]` | Command arguments |
| `env` | object | No | `{}` | Environment variables (key-value pairs) |
| `workDir` | string | No | - | Working directory for execution |
| `shell` | boolean | No | `false` | Run command through shell |
| `sandboxEnabled` | boolean | No | `true` | Enable sandbox isolation |
| `isolationLevel` | string | No | `standard` | Isolation level |
| `networkAccess` | string | No | `host` | Network access mode |
| `timeout` | number | No | `60` | Timeout in seconds |
| `maxMemoryMB` | number | No | `512` | Max memory in MB |
| `inputFromPrevious` | boolean | No | `false` | Send input data to stdin |
| `outputFormat` | string | No | `text` | Output format |
| `additionalMounts` | array | No | `[]` | Additional filesystem mounts |

---

## Running CLI AI Agents

### Supported Agents

The CLI Execute node works with any CLI-based AI agent:

| Agent | Command | Description |
|-------|---------|-------------|
| **Claude Code** | `claude` | Anthropic's CLI coding assistant |
| **OpenAI Codex CLI** | `codex` | OpenAI's code generation CLI |
| **Aider** | `aider` | AI pair programming in terminal |
| **GitHub Copilot CLI** | `gh copilot` | GitHub's CLI assistant |
| **Continue** | `continue` | Open-source AI code assistant |
| **Cursor CLI** | `cursor` | Cursor editor's CLI mode |

### Example - Claude Code

Run Claude Code to analyze a codebase:

```json
{
  "name": "Analyze with Claude Code",
  "type": "n8n-nodes-base.cliExecute",
  "parameters": {
    "command": "claude",
    "args": [
      "--print",
      "--prompt", "Analyze this codebase and identify potential security vulnerabilities"
    ],
    "sandboxEnabled": true,
    "isolationLevel": "strict",
    "networkAccess": "host",
    "timeout": 300,
    "maxMemoryMB": 2048,
    "additionalMounts": [
      {
        "source": "/home/user/myproject",
        "destination": "/workspace",
        "readWrite": false
      }
    ],
    "env": {
      "ANTHROPIC_API_KEY": "={{ $credentials.anthropicApi.apiKey }}"
    }
  }
}
```

### Example - OpenAI Codex CLI

Generate code with Codex:

```json
{
  "name": "Generate with Codex",
  "type": "n8n-nodes-base.cliExecute",
  "parameters": {
    "command": "codex",
    "args": [
      "--quiet",
      "--approval-mode", "full-auto",
      "Add input validation to all API endpoints"
    ],
    "sandboxEnabled": true,
    "isolationLevel": "standard",
    "networkAccess": "host",
    "timeout": 600,
    "maxMemoryMB": 2048,
    "additionalMounts": [
      {
        "source": "/home/user/project",
        "destination": "/project",
        "readWrite": true
      }
    ],
    "workDir": "/project",
    "env": {
      "OPENAI_API_KEY": "={{ $credentials.openAiApi.apiKey }}"
    }
  }
}
```

### Example - Aider for Code Review

Use Aider to review and suggest improvements:

```json
{
  "name": "Code Review with Aider",
  "type": "n8n-nodes-base.cliExecute",
  "parameters": {
    "command": "aider",
    "args": [
      "--no-git",
      "--yes",
      "--message", "Review this code for best practices and suggest improvements"
    ],
    "sandboxEnabled": true,
    "isolationLevel": "standard",
    "networkAccess": "host",
    "timeout": 300,
    "additionalMounts": [
      {
        "source": "={{ $json.repoPath }}",
        "destination": "/repo",
        "readWrite": false
      }
    ],
    "workDir": "/repo"
  }
}
```

### Example - Multi-Agent Workflow

Chain multiple AI agents for complex tasks:

```json
{
  "nodes": [
    {
      "name": "Analyze with Claude",
      "type": "n8n-nodes-base.cliExecute",
      "parameters": {
        "command": "claude",
        "args": ["--print", "--prompt", "List all functions that need tests"],
        "outputFormat": "json",
        "additionalMounts": [{"source": "/project", "destination": "/workspace", "readWrite": false}]
      }
    },
    {
      "name": "Generate Tests with Codex",
      "type": "n8n-nodes-base.cliExecute",
      "parameters": {
        "command": "codex",
        "args": ["--quiet", "Generate unit tests for: {{ $json.stdout.functions }}"],
        "additionalMounts": [{"source": "/project", "destination": "/workspace", "readWrite": true}]
      }
    }
  ]
}
```

---

## Configuration Options

### Isolation Levels

| Level | Description | Use Case |
|-------|-------------|----------|
| `none` | No isolation | Development/debugging only |
| `minimal` | Basic filesystem isolation | Trusted tools |
| `standard` | + PID namespace, resource limits | **Recommended for most agents** |
| `strict` | + Network isolation, seccomp filtering | Untrusted code analysis |
| `paranoid` | + No host filesystem access | Maximum security |

### Network Access Modes

| Mode | Description | When to Use |
|------|-------------|-------------|
| `host` | Full network access | AI agents that need API access |
| `isolated` | No network | Offline analysis tools |
| `loopback` | Localhost only | Local services only |

### Output Formats

| Format | Description |
|--------|-------------|
| `text` | Raw text output (default) |
| `json` | Parse stdout as JSON object |
| `lines` | Split stdout into array of lines |

### Additional Mounts

Mount host directories into the sandbox:

```json
{
  "additionalMounts": [
    {
      "source": "/home/user/project",
      "destination": "/workspace",
      "readWrite": true
    },
    {
      "source": "/home/user/.config/claude",
      "destination": "/home/sandbox/.config/claude",
      "readWrite": false
    }
  ]
}
```

---

## Output Structure

Every execution returns:

```json
{
  "json": {
    "exitCode": 0,
    "stdout": "...",
    "stderr": "...",
    "killed": false,
    "killReason": "",
    "duration": 1500,
    "cpuTime": 1200,
    "maxMemory": 104857600
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `exitCode` | number | Process exit code (0 = success) |
| `stdout` | string/object/array | Command output (format depends on outputFormat) |
| `stderr` | string | Error output |
| `killed` | boolean | Whether process was terminated |
| `killReason` | string | Reason if killed (timeout, signal, etc.) |
| `duration` | number | Wall-clock time in milliseconds |
| `cpuTime` | number | CPU time used in milliseconds |
| `maxMemory` | number | Peak memory usage in bytes |

---

## Security

### Sandbox Technology

On Linux, m9m uses **bubblewrap** (bwrap) for sandboxing:

- **Namespace isolation**: PID, network, mount, user namespaces
- **Resource limits**: cgroups v2 for memory, CPU, process count
- **Syscall filtering**: Seccomp BPF for blocking dangerous operations
- **Filesystem isolation**: Controlled mounts with read-only defaults

### Prerequisites

```bash
# Ubuntu/Debian
sudo apt install bubblewrap

# Fedora/RHEL
sudo dnf install bubblewrap

# Arch Linux
sudo pacman -S bubblewrap
```

### Default Security Settings

| Setting | Default |
|---------|---------|
| Memory limit | 512 MB |
| Timeout | 60 seconds |
| Max processes | 50 |
| Network | Host (configurable) |
| Filesystem | Read-only mounts |

### Blocked Syscalls (strict/paranoid)

- `reboot`, `kexec_load` - System control
- `mount`, `umount` - Filesystem manipulation
- `pivot_root`, `chroot` - Container escape
- `init_module`, `delete_module` - Kernel modules
- `ptrace` - Process debugging
- `setns` - Namespace escape

---

## Best Practices

### For AI Agents

1. **Use appropriate isolation**: `standard` for most agents, `strict` for untrusted code
2. **Set reasonable timeouts**: AI agents may take minutes; use 300-600 seconds
3. **Allocate sufficient memory**: 1-2 GB for most AI agents
4. **Enable network access**: Most agents need API access to work
5. **Mount code read-only** when possible to prevent accidental modifications

### For Security

1. **Never disable sandbox** in production
2. **Use read-only mounts** unless writes are required
3. **Limit network access** when API calls aren't needed
4. **Set memory limits** to prevent runaway processes
5. **Review agent output** before applying changes to codebase

### For Reliability

1. **Handle non-zero exit codes** in your workflow
2. **Parse stderr** for error messages
3. **Check `killed` flag** to detect timeouts
4. **Use JSON output format** for structured data

---

## Troubleshooting

### Agent not found

```json
{"exitCode": 127, "stderr": "command not found: claude"}
```

**Solution**: Ensure the agent CLI is installed and in PATH, or use absolute path.

### Permission denied

```json
{"exitCode": 1, "stderr": "Permission denied"}
```

**Solution**: Check mount permissions or use `readWrite: true` for mounts that need write access.

### Timeout exceeded

```json
{"killed": true, "killReason": "timeout"}
```

**Solution**: Increase `timeout` parameter for long-running operations.

### Out of memory

```json
{"killed": true, "killReason": "signal 9"}
```

**Solution**: Increase `maxMemoryMB` parameter.

---

## Quick Reference

| Parameter | Recommended for AI Agents |
|-----------|--------------------------|
| `isolationLevel` | `standard` or `strict` |
| `networkAccess` | `host` (for API access) |
| `timeout` | `300` - `600` seconds |
| `maxMemoryMB` | `1024` - `2048` MB |
| `outputFormat` | `json` (for structured output) |

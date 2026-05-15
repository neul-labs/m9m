# m9m-cli

> **The n8n alternative without the bugs — faster, more reliable workflow automation.**

Python bindings for [m9m](https://github.com/neul-labs/m9m), a drop-in n8n alternative written in Go. 5–10× faster execution, 70% lower memory, deterministic runs, single static binary. No Node.js or JVM on the server — the engine is a Go binary that this package downloads and calls natively.

[![PyPI version](https://img.shields.io/pypi/v/m9m-cli.svg?style=flat-square)](https://pypi.org/project/m9m-cli/)
[![PyPI downloads](https://img.shields.io/pypi/dm/m9m-cli.svg?style=flat-square)](https://pypi.org/project/m9m-cli/)
[![Python versions](https://img.shields.io/pypi/pyversions/m9m-cli.svg?style=flat-square)](https://pypi.org/project/m9m-cli/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](https://github.com/neul-labs/m9m/blob/main/LICENSE)

```bash
pip install m9m-cli
```

---

## What you get

- **n8n-compatible workflow execution** — runs n8n workflow JSON, expressions, and credentials unchanged.
- **Single binary, auto-downloaded** — `m9m-cli` fetches the platform-native `m9m` binary on first use (macOS Intel + Apple Silicon, Linux AMD64 + ARM64, Windows AMD64). No Docker, no server.
- **Fully typed Python API** — `py.typed` marker, full type hints, autocomplete in mypy / pyright / Pylance.
- **Pythonic** — context-manager API, decorator-based custom node registration.
- **In-process or remote** — embed the engine, or point the client at a `m9m serve` instance.

---

## Install

```bash
pip install m9m-cli
# or
uv add m9m-cli
poetry add m9m-cli
```

The `m9m` binary is downloaded on first use to `~/.m9m/bin/m9m`. For offline / air-gapped use, see [Pinning and offline use](#pinning-and-offline-use) below.

Requires Python 3.8+.

---

## 30-second quickstart

```python
from m9m import WorkflowEngine, Workflow

with WorkflowEngine() as engine:
    workflow = Workflow.from_file("my-workflow.json")
    result = engine.execute(workflow, [
        {"json": {"email": "user@example.com", "name": "Alice"}}
    ])

    print(result.data)
    if result.error:
        print("Failed:", result.error)
```

Build a workflow programmatically:

```python
from m9m import WorkflowEngine, create_workflow

workflow = create_workflow({
    "name": "Email Notification",
    "nodes": [
        {"name": "Start", "type": "n8n-nodes-base.start", "parameters": {}},
        {
            "name": "Send Email",
            "type": "n8n-nodes-base.emailSend",
            "parameters": {
                "to": "={{ $json.email }}",
                "subject": "Welcome!",
                "text": "Hello {{ $json.name }}",
            },
        },
    ],
    "connections": {
        "Start": {"main": [[{"node": "Send Email", "type": "main", "index": 0}]]},
    },
})

result = WorkflowEngine().execute(workflow)
```

Register a custom node:

```python
from m9m import WorkflowEngine

engine = WorkflowEngine()

@engine.node("custom.uppercase", name="Uppercase")
def uppercase(input_data, params):
    return [
        {"json": {**item["json"], "text": item["json"]["text"].upper()}}
        for item in input_data
    ]
```

---

## API at a glance

```python
from m9m import (
    WorkflowEngine,    # load + execute workflows
    Workflow,          # workflow definition
    create_workflow,   # builder
    DataItem,          # node-to-node data
    ExecutionResult,   # execute() return
    CredentialManager, # credential storage
    version,           # binary version
    download_binary,   # pin a version
    get_binary_path,   # ~/.m9m/bin/m9m
)
```

Full reference: [docs.neullabs.com/m9m/sdk](https://docs.neullabs.com/m9m).

---

## Supported platforms

| OS | Architecture |
|---|---|
| macOS | Intel (AMD64), Apple Silicon (ARM64) |
| Linux | AMD64, ARM64 |
| Windows | AMD64 |

The correct binary is auto-selected by `platform.system()` and `platform.machine()` on first use.

---

## Why m9m? (the short version)

- **Faster** — sub-second cold start, 5–10× workflow execution, 75% smaller container than n8n.
- **More reliable** — single static binary, no Node.js heap leaks, deterministic execution, no npm transitive-dep CVEs in the runtime path.
- **Drop-in compatible** — n8n workflow JSON, expressions, and credentials run unchanged.

Full comparison + benchmark methodology: [github.com/neul-labs/m9m](https://github.com/neul-labs/m9m#why-m9m-vs-n8n).

---

## Pinning and offline use

### Pin the binary version

```python
from m9m import download_binary
download_binary("v0.2.1")
```

Or via environment variable before first use:

```bash
export M9M_VERSION=v0.2.1
```

### Air-gapped / offline

Stage the binary manually at `~/.m9m/bin/m9m`:

```bash
# Linux AMD64
curl -L -o ~/.m9m/bin/m9m \
  https://github.com/neul-labs/m9m/releases/latest/download/m9m-linux-amd64
chmod +x ~/.m9m/bin/m9m
```

| Variable | Meaning |
|---|---|
| `M9M_VERSION` | Pin a specific binary release tag |
| `M9M_BINARY_PATH` | Use a pre-staged binary, skip download entirely |

---

## FAQ

### Does this package bundle a separate runtime?
No — m9m is a Go binary. This package is a thin Python client that invokes it. There is no embedded interpreter, no FFI to a C extension at runtime.

### Where does the binary come from?
GitHub Releases: `https://github.com/neul-labs/m9m/releases`. Each release ships SHA-256 checksums; the downloader verifies them.

### Can I use this offline / air-gapped?
Yes. Set `M9M_BINARY_PATH=/path/to/m9m` to point at a pre-staged binary, or place it at `~/.m9m/bin/m9m`. See [Pinning and offline use](#pinning-and-offline-use).

### How do I pin a specific binary version?
`download_binary("v0.2.1")` once, or set `M9M_VERSION=v0.2.1` before first use.

### Does it run n8n workflows unchanged?
For 40+ built-in node types: yes. Community n8n nodes (`n8n-nodes-*`) and n8n Cloud–specific features are not supported. See the [migration guide](https://github.com/neul-labs/m9m/blob/main/docs/migration/from-n8n.md).

---

## Type hints

`m9m-cli` ships a `py.typed` marker. mypy, pyright, and Pylance get full autocomplete:

```python
from m9m import WorkflowEngine, Workflow, DataItem

engine: WorkflowEngine = WorkflowEngine()
workflow: Workflow = Workflow.from_file("workflow.json")
result = engine.execute(workflow, [DataItem(json={"key": "value"})])
```

---

## Links

- **Documentation:** [docs.neullabs.com/m9m](https://docs.neullabs.com/m9m)
- **Repository:** [github.com/neul-labs/m9m](https://github.com/neul-labs/m9m)
- **Changelog:** [Releases](https://github.com/neul-labs/m9m/releases)
- **Issues:** [GitHub Issues](https://github.com/neul-labs/m9m/issues)
- **Node.js SDK:** [`m9m-cli` on npm](https://www.npmjs.com/package/m9m-cli)

## License

MIT — see [LICENSE](https://github.com/neul-labs/m9m/blob/main/LICENSE).

---
title: Install m9m
description: Install m9m via Homebrew, curl, Go, Docker, npm, or pip. Prebuilt binaries for macOS, Linux, and Windows on AMD64 and ARM64.
keywords: m9m install, n8n alternative install, workflow automation install, Homebrew, Docker, npm m9m-cli, pip m9m-cli
---

# Install m9m

m9m ships as a single statically-linked Go binary. Pick whichever install path fits your environment — they all produce the same binary.

## Requirements

- **Operating system:** macOS, Linux, or Windows
- **Architecture:** AMD64 (x86_64) or ARM64 (aarch64)
- **Disk:** ~30 MB for the binary, plus storage for workflows and the job queue
- **Network:** internet access for the initial download (not needed at runtime)

## Install methods

### Installer script (any platform) — recommended

```bash
curl -fsSL https://raw.githubusercontent.com/neul-labs/m9m/main/install.sh | bash
```

Auto-detects OS and architecture, downloads the right binary, places it on your `$PATH`. Falls back to a local Go build if no prebuilt is available for your platform.

### Homebrew (macOS / Linux)

```bash
brew tap neul-labs/tap
brew install m9m
```

### Go

```bash
go install github.com/neul-labs/m9m/cmd/m9m@latest
```

Installs `m9m` to `$GOPATH/bin`. Requires Go 1.21+.

### Docker — GitHub Container Registry (releases)

```bash
docker run -p 8080:8080 ghcr.io/neul-labs/m9m:latest
```

With persistent data:

```bash
docker run -p 8080:8080 \
  -v m9m-data:/app/data \
  ghcr.io/neul-labs/m9m:latest
```

With custom configuration:

```bash
docker run -p 8080:8080 \
  -v ./config.yaml:/app/config/config.yaml \
  -v m9m-data:/app/data \
  ghcr.io/neul-labs/m9m:latest
```

### Docker — Docker Hub (CI builds)

```bash
docker run -p 8080:8080 neul-labs/m9m:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  m9m:
    image: ghcr.io/neul-labs/m9m:latest
    ports:
      - "8080:8080"
    volumes:
      - m9m-data:/app/data
    environment:
      - M9M_LOG_LEVEL=info
    restart: unless-stopped

volumes:
  m9m-data:
```

```bash
docker-compose up -d
```

### Python SDK

```bash
pip install m9m-cli
# or
uv add m9m-cli
poetry add m9m-cli
```

The `m9m-cli` package downloads the binary on first use. See the [PyPI page](https://pypi.org/project/m9m-cli/).

### Node.js SDK

```bash
npm install m9m-cli
# or
yarn add m9m-cli
pnpm add m9m-cli
bun add m9m-cli
```

The `m9m-cli` package downloads the binary on `npm install`. See the [npm page](https://www.npmjs.com/package/m9m-cli).

### Build from source

```bash
git clone https://github.com/neul-labs/m9m.git
cd m9m
make deps
make build
./m9m version
```

### Prebuilt binaries

Download directly from [GitHub Releases](https://github.com/neul-labs/m9m/releases).

=== "Linux AMD64"

    ```bash
    curl -LO https://github.com/neul-labs/m9m/releases/latest/download/m9m-linux-amd64
    chmod +x m9m-linux-amd64
    sudo mv m9m-linux-amd64 /usr/local/bin/m9m
    ```

=== "Linux ARM64"

    ```bash
    curl -LO https://github.com/neul-labs/m9m/releases/latest/download/m9m-linux-arm64
    chmod +x m9m-linux-arm64
    sudo mv m9m-linux-arm64 /usr/local/bin/m9m
    ```

=== "macOS AMD64 (Intel)"

    ```bash
    curl -LO https://github.com/neul-labs/m9m/releases/latest/download/m9m-darwin-amd64
    chmod +x m9m-darwin-amd64
    sudo mv m9m-darwin-amd64 /usr/local/bin/m9m
    ```

=== "macOS ARM64 (Apple Silicon)"

    ```bash
    curl -LO https://github.com/neul-labs/m9m/releases/latest/download/m9m-darwin-arm64
    chmod +x m9m-darwin-arm64
    sudo mv m9m-darwin-arm64 /usr/local/bin/m9m
    ```

=== "Windows AMD64"

    ```powershell
    Invoke-WebRequest `
      -Uri https://github.com/neul-labs/m9m/releases/latest/download/m9m-windows-amd64.exe `
      -OutFile m9m.exe
    ```

Each release ships SHA-256 checksums next to the binary — verify before installing in production.

## Verify installation

```bash
m9m version
m9m serve

# In another terminal
curl http://localhost:8080/health
```

## Data directories

m9m stores state under `~/.m9m/` by default:

| Platform | Location |
|---|---|
| Linux | `~/.m9m/` |
| macOS | `~/.m9m/` |
| Windows | `%USERPROFILE%\.m9m\` |

Layout:

```
~/.m9m/
├── data/
│   ├── m9m.db        # Workflow storage (SQLite)
│   └── queue.db      # Job queue (SQLite)
├── config/
│   └── config.yaml   # Configuration
└── logs/
    └── m9m.log       # Logs
```

Override with `M9M_DATA_DIR`, `M9M_CONFIG_DIR`, `M9M_LOG_DIR` — see [Environment Variables](../configuration/environment.md).

## Troubleshooting

| Symptom | Fix |
|---|---|
| `command not found: m9m` after Go install | Add `$(go env GOPATH)/bin` to your `$PATH`. |
| Postinstall download fails behind proxy (npm / pip) | Set `M9M_DOWNLOAD_BINARY=0` and stage `~/.m9m/bin/m9m` manually. |
| Docker container exits immediately | Check that port 8080 isn't already bound: `lsof -i :8080`. |
| `Permission denied` on Linux binary | `chmod +x` the binary; verify it's not on a `noexec` mount. |
| Homebrew tap not found | Confirm the tap with `brew tap` and re-run `brew tap neul-labs/tap`. |

## Next steps

- [Create your first workflow](first-workflow.md)
- [Migrate existing n8n workflows](../migrate-from-n8n.md)
- [Configure m9m](../configuration/index.md)
- [Deploy to production](../deployment/index.md)

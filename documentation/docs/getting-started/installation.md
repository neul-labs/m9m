# Installation

This guide covers all methods for installing m9m.

## Requirements

- **Go 1.21+** (for building from source)
- **Docker** (optional, for container deployment)
- **64-bit Linux, macOS, or Windows**

## Installation Methods

### Go Install (Recommended)

The simplest way to install m9m:

```bash
go install github.com/neul-labs/m9m/cmd/m9m@latest
```

This installs the `m9m` binary to your `$GOPATH/bin` directory.

### Docker

Run m9m in a container:

```bash
# Basic usage
docker run -p 8080:8080 ghcr.io/neul-labs/m9m:latest

# With persistent data
docker run -p 8080:8080 \
  -v m9m-data:/app/data \
  ghcr.io/neul-labs/m9m:latest

# With custom configuration
docker run -p 8080:8080 \
  -v ./config.yaml:/app/config/config.yaml \
  -v m9m-data:/app/data \
  ghcr.io/neul-labs/m9m:latest
```

### Docker Compose

Create a `docker-compose.yml`:

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

Run with:

```bash
docker-compose up -d
```

### Build from Source

Clone and build the project:

```bash
# Clone repository
git clone https://github.com/neul-labs/m9m.git
cd m9m

# Install dependencies
make deps

# Build
make build

# The binary is at ./m9m
./m9m version
```

### Binary Downloads

Download pre-built binaries from the [GitHub Releases](https://github.com/neul-labs/m9m/releases) page.

=== "Linux (amd64)"

    ```bash
    curl -LO https://github.com/neul-labs/m9m/releases/latest/download/m9m-linux-amd64
    chmod +x m9m-linux-amd64
    sudo mv m9m-linux-amd64 /usr/local/bin/m9m
    ```

=== "Linux (arm64)"

    ```bash
    curl -LO https://github.com/neul-labs/m9m/releases/latest/download/m9m-linux-arm64
    chmod +x m9m-linux-arm64
    sudo mv m9m-linux-arm64 /usr/local/bin/m9m
    ```

=== "macOS (amd64)"

    ```bash
    curl -LO https://github.com/neul-labs/m9m/releases/latest/download/m9m-darwin-amd64
    chmod +x m9m-darwin-amd64
    sudo mv m9m-darwin-amd64 /usr/local/bin/m9m
    ```

=== "macOS (arm64)"

    ```bash
    curl -LO https://github.com/neul-labs/m9m/releases/latest/download/m9m-darwin-arm64
    chmod +x m9m-darwin-arm64
    sudo mv m9m-darwin-arm64 /usr/local/bin/m9m
    ```

## Verify Installation

Check that m9m is installed correctly:

```bash
# Check version
m9m version

# Start the server
m9m serve

# In another terminal, check health
curl http://localhost:8080/health
```

## Data Directories

m9m stores data in the following locations by default:

| Platform | Location |
|----------|----------|
| Linux | `~/.m9m/` |
| macOS | `~/.m9m/` |
| Windows | `%USERPROFILE%\.m9m\` |

Directory structure:

```
~/.m9m/
├── data/
│   ├── m9m.db        # Workflow storage (SQLite)
│   └── queue.db      # Job queue (SQLite)
├── config/
│   └── config.yaml   # Configuration file
└── logs/
    └── m9m.log       # Log files
```

## Next Steps

- [Create your first workflow](first-workflow.md)
- [Configure m9m](../configuration/index.md)
- [Deploy to production](../deployment/index.md)

# Installation

This guide covers all the ways to install m9m on your system.

## System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 1 core | 2+ cores |
| Memory | 256MB | 512MB+ |
| Disk | 500MB | 1GB+ |
| OS | Linux, macOS, Windows | Linux |

## Installation Methods

### Docker (Recommended)

The easiest way to run m9m is with Docker:

```bash
docker run -d \
  --name m9m \
  -p 8080:8080 \
  -p 9090:9090 \
  -v m9m_data:/data \
  m9m/m9m:latest
```

!!! tip "Docker Compose"
    For production deployments, see the [Docker deployment guide](../deployment/docker.md) for a complete Docker Compose setup.

### Binary Download

Download pre-built binaries for your platform:

=== "Linux (amd64)"

    ```bash
    # Download
    wget https://github.com/m9m/m9m/releases/latest/download/m9m-linux-amd64

    # Make executable
    chmod +x m9m-linux-amd64

    # Move to PATH
    sudo mv m9m-linux-amd64 /usr/local/bin/m9m

    # Verify installation
    m9m --version
    ```

=== "Linux (arm64)"

    ```bash
    wget https://github.com/m9m/m9m/releases/latest/download/m9m-linux-arm64
    chmod +x m9m-linux-arm64
    sudo mv m9m-linux-arm64 /usr/local/bin/m9m
    m9m --version
    ```

=== "macOS (amd64)"

    ```bash
    wget https://github.com/m9m/m9m/releases/latest/download/m9m-darwin-amd64
    chmod +x m9m-darwin-amd64
    sudo mv m9m-darwin-amd64 /usr/local/bin/m9m
    m9m --version
    ```

=== "macOS (arm64)"

    ```bash
    wget https://github.com/m9m/m9m/releases/latest/download/m9m-darwin-arm64
    chmod +x m9m-darwin-arm64
    sudo mv m9m-darwin-arm64 /usr/local/bin/m9m
    m9m --version
    ```

=== "Windows"

    ```powershell
    # Download
    Invoke-WebRequest -Uri "https://github.com/m9m/m9m/releases/latest/download/m9m-windows-amd64.exe" -OutFile "m9m.exe"

    # Add to PATH or run directly
    .\m9m.exe --version
    ```

### Build from Source

Build m9m from source code:

```bash
# Prerequisites: Go 1.21+
go version

# Clone repository
git clone https://github.com/m9m/m9m.git
cd m9m

# Install dependencies
go mod download

# Build
make build

# Verify
./m9m --version
```

#### Development Build

For development with additional tools:

```bash
# Install development dependencies
make deps

# Build with race detector
go build -race -o m9m cmd/m9m/main.go

# Run tests
make test
```

## Verify Installation

After installation, verify m9m is working:

```bash
# Check version
m9m --version

# View help
m9m --help

# Start server
m9m serve
```

You should see output like:

```
m9m v1.0.0
Build: 2024-01-15T10:30:00Z
Go: go1.21.5
```

## Configuration

### Environment Variables

Configure m9m using environment variables:

```bash
# Server settings
export M9M_PORT=8080
export M9M_HOST=0.0.0.0

# Queue settings
export M9M_QUEUE_TYPE=redis
export M9M_QUEUE_URL=redis://localhost:6379

# Monitoring
export M9M_METRICS_PORT=9090
```

### Configuration File

Create a configuration file at `~/.m9m/config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

queue:
  type: "memory"
  max_workers: 10

monitoring:
  metrics_port: 9090
  enable_tracing: false

database:
  type: "sqlite"
  path: "~/.m9m/data.db"
```

## Systemd Service (Linux)

Run m9m as a systemd service:

```ini
# /etc/systemd/system/m9m.service
[Unit]
Description=m9m Workflow Automation
After=network.target

[Service]
Type=simple
User=m9m
Group=m9m
ExecStart=/usr/local/bin/m9m serve
Restart=always
RestartSec=5
Environment=M9M_PORT=8080

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable m9m
sudo systemctl start m9m
sudo systemctl status m9m
```

## Next Steps

- [Quick Start](quickstart.md) - Run your first workflow
- [Configuration](../reference/configuration.md) - Full configuration reference
- [Docker Deployment](../deployment/docker.md) - Production Docker setup

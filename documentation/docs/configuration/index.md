# Configuration Overview

m9m can be configured through configuration files, environment variables, and command-line flags.

## Configuration Methods

| Method | Priority | Use Case |
|--------|----------|----------|
| Command-line flags | Highest | Quick overrides |
| Environment variables | Medium | Container deployments |
| Configuration file | Lowest | Persistent settings |

## Configuration File Location

m9m looks for configuration in these locations (in order):

1. `./config.yaml` (current directory)
2. `~/.m9m/config/config.yaml` (user directory)
3. `/etc/m9m/config.yaml` (system-wide)

## Quick Start Configuration

Minimal configuration to get started:

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  type: "sqlite"
  sqlite:
    path: "./data/m9m.db"

queue:
  type: "sqlite"
  sqlite:
    path: "./data/queue.db"
```

## Configuration Sections

| Section | Description |
|---------|-------------|
| [Server](server.md) | Host, port, TLS settings |
| [Database](database.md) | Storage backend configuration |
| [Queue](queue.md) | Job queue settings |
| [Security](security.md) | Authentication, CORS |
| [Environment](environment.md) | Environment variables |

## Full Configuration Example

```yaml
# Server
server:
  host: "0.0.0.0"
  port: 8080
  tls:
    enabled: false

# Database
database:
  type: "sqlite"
  sqlite:
    path: "./data/m9m.db"

# Queue
queue:
  type: "sqlite"
  max_workers: 4
  sqlite:
    path: "./data/queue.db"

# Logging
logging:
  level: "info"
  format: "json"

# Security
security:
  jwt:
    enabled: true
    secret: "your-secret-key"
    expiration: "24h"
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"

# Execution
execution:
  timeout: "1h"
  retry:
    enabled: true
    max_attempts: 3
```

## Validating Configuration

Check your configuration:

```bash
m9m config validate
```

View effective configuration:

```bash
m9m config show
```

## Next Steps

- [Server Configuration](server.md)
- [Database Configuration](database.md)
- [Security Configuration](security.md)

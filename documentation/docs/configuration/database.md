# Database Configuration

Configure the storage backend for workflows, executions, and credentials.

## Supported Backends

| Backend | Use Case |
|---------|----------|
| SQLite | Development, single-node |
| PostgreSQL | Production, multi-node |

## SQLite (Default)

Zero-dependency local storage:

```yaml
database:
  type: "sqlite"
  sqlite:
    path: "./data/m9m.db"
```

| Setting | Default | Description |
|---------|---------|-------------|
| `sqlite.path` | `~/.m9m/data/m9m.db` | Database file path |

### CLI Override

```bash
m9m serve --db /custom/path/m9m.db
```

### Benefits

- No external dependencies
- Simple backup (copy file)
- Good for development
- Sufficient for single-node production

### Limitations

- Single writer (no concurrent writes)
- Not suitable for clustered deployments

## PostgreSQL

Production-grade relational database:

```yaml
database:
  type: "postgres"
  postgres:
    host: "localhost"
    port: 5432
    database: "m9m"
    user: "m9m"
    password: "changeme"
    ssl_mode: "disable"
    max_connections: 25
    max_idle_connections: 5
    connection_lifetime: "5m"
```

### Connection URL

Alternatively, use a connection URL:

```yaml
database:
  type: "postgres"
  postgres:
    url: "postgres://user:password@localhost:5432/m9m?sslmode=require"
```

### CLI Override

```bash
m9m serve --postgres "postgres://user:pass@localhost:5432/m9m"
```

### Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `host` | `localhost` | Database host |
| `port` | `5432` | Database port |
| `database` | `m9m` | Database name |
| `user` | `m9m` | Username |
| `password` | - | Password |
| `ssl_mode` | `disable` | SSL mode |
| `max_connections` | `25` | Max open connections |
| `max_idle_connections` | `5` | Max idle connections |
| `connection_lifetime` | `5m` | Connection max lifetime |

### SSL Modes

| Mode | Description |
|------|-------------|
| `disable` | No SSL |
| `require` | SSL required, no verification |
| `verify-ca` | Verify server certificate |
| `verify-full` | Verify certificate and hostname |

### Create Database

```sql
CREATE DATABASE m9m;
CREATE USER m9m WITH PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE m9m TO m9m;
```

## Environment Variables

```bash
# SQLite
export M9M_DB_TYPE=sqlite
export M9M_DB_PATH=./data/m9m.db

# PostgreSQL
export M9M_DB_TYPE=postgres
export M9M_POSTGRES_HOST=localhost
export M9M_POSTGRES_PORT=5432
export M9M_POSTGRES_DATABASE=m9m
export M9M_POSTGRES_USER=m9m
export M9M_POSTGRES_PASSWORD=secret
```

## Schema Management

m9m automatically manages database schema:

- Creates tables on first run
- Runs migrations on upgrade
- No manual schema management needed

## Backup

### SQLite Backup

```bash
# Stop server first for consistent backup
cp ~/.m9m/data/m9m.db backup.db

# Or use SQLite backup
sqlite3 ~/.m9m/data/m9m.db ".backup backup.db"
```

### PostgreSQL Backup

```bash
pg_dump -U m9m -d m9m > backup.sql

# Restore
psql -U m9m -d m9m < backup.sql
```

## Data Stored

The database stores:

| Table | Contents |
|-------|----------|
| `workflows` | Workflow definitions |
| `executions` | Execution history and data |
| `credentials` | Encrypted credentials |
| `schedules` | Scheduled workflow configs |
| `settings` | Application settings |

## Performance Tuning

### PostgreSQL

```yaml
database:
  postgres:
    max_connections: 50
    max_idle_connections: 10
    connection_lifetime: "1h"
```

### Connection Pooling

For high-traffic deployments, use PgBouncer:

```yaml
database:
  postgres:
    url: "postgres://user:pass@pgbouncer:6432/m9m"
```

## Troubleshooting

### Connection Refused

```
Error: connection refused
```

Check PostgreSQL is running and accessible:

```bash
psql -h localhost -U m9m -d m9m
```

### Permission Denied

```
Error: permission denied
```

Grant necessary permissions:

```sql
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO m9m;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO m9m;
```

### Database Locked (SQLite)

```
Error: database is locked
```

Only one m9m instance can write to SQLite. Use PostgreSQL for multiple instances.

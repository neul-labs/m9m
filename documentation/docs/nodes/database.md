# Database Nodes

Database nodes allow workflows to interact with relational databases.

## PostgreSQL Node

Execute queries against PostgreSQL databases.

### Type

```
n8n-nodes-base.postgres
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `connectionUrl` | string | No* | PostgreSQL connection URL |
| `host` | string | No* | Database host |
| `port` | number | No | Port (default: 5432) |
| `database` | string | No* | Database name |
| `user` | string | No* | Username |
| `password` | string | No* | Password |
| `operation` | string | Yes | `executeQuery`, `insert`, `update`, `delete` |
| `query` | string | Depends | SQL query (for executeQuery) |

*Either `connectionUrl` OR individual connection parameters required.

### Connection Examples

#### Using Connection URL

```json
{
  "type": "n8n-nodes-base.postgres",
  "parameters": {
    "connectionUrl": "postgres://user:password@localhost:5432/mydb",
    "operation": "executeQuery",
    "query": "SELECT * FROM users"
  }
}
```

#### Using Individual Parameters

```json
{
  "type": "n8n-nodes-base.postgres",
  "parameters": {
    "host": "localhost",
    "port": 5432,
    "database": "mydb",
    "user": "dbuser",
    "password": "={{ $credentials.postgres.password }}",
    "operation": "executeQuery",
    "query": "SELECT * FROM users WHERE status = 'active'"
  }
}
```

### Operations

#### Execute Query

```json
{
  "type": "n8n-nodes-base.postgres",
  "parameters": {
    "connectionUrl": "postgres://...",
    "operation": "executeQuery",
    "query": "SELECT id, name, email FROM users WHERE created_at > '2024-01-01'"
  }
}
```

Output:
```json
[
  {"json": {"id": 1, "name": "John", "email": "john@example.com"}},
  {"json": {"id": 2, "name": "Jane", "email": "jane@example.com"}}
]
```

#### Insert

```json
{
  "type": "n8n-nodes-base.postgres",
  "parameters": {
    "connectionUrl": "postgres://...",
    "operation": "insert",
    "table": "users",
    "columns": ["name", "email"],
    "values": ["={{ $json.name }}", "={{ $json.email }}"]
  }
}
```

#### Update

```json
{
  "type": "n8n-nodes-base.postgres",
  "parameters": {
    "connectionUrl": "postgres://...",
    "operation": "update",
    "table": "users",
    "updateKey": "id",
    "updateValue": "={{ $json.id }}",
    "columns": ["status"],
    "values": ["inactive"]
  }
}
```

#### Delete

```json
{
  "type": "n8n-nodes-base.postgres",
  "parameters": {
    "connectionUrl": "postgres://...",
    "operation": "delete",
    "table": "users",
    "deleteKey": "id",
    "deleteValue": "={{ $json.id }}"
  }
}
```

---

## MySQL Node

Execute queries against MySQL databases.

### Type

```
n8n-nodes-base.mysql
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `connectionUrl` | string | No* | MySQL connection URL |
| `host` | string | No* | Database host |
| `port` | number | No | Port (default: 3306) |
| `database` | string | No* | Database name |
| `user` | string | No* | Username |
| `password` | string | No* | Password |
| `operation` | string | Yes | `executeQuery`, `insert`, `update`, `delete` |
| `query` | string | Depends | SQL query |

### Examples

#### Select Query

```json
{
  "type": "n8n-nodes-base.mysql",
  "parameters": {
    "host": "localhost",
    "port": 3306,
    "database": "myapp",
    "user": "root",
    "password": "={{ $credentials.mysql.password }}",
    "operation": "executeQuery",
    "query": "SELECT * FROM orders WHERE status = 'pending'"
  }
}
```

#### Insert with Expression

```json
{
  "type": "n8n-nodes-base.mysql",
  "parameters": {
    "connectionUrl": "mysql://user:pass@localhost:3306/myapp",
    "operation": "executeQuery",
    "query": "INSERT INTO logs (message, timestamp) VALUES ('{{ $json.message }}', NOW())"
  }
}
```

---

## SQLite Node

Execute queries against SQLite databases.

### Type

```
n8n-nodes-base.sqlite
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `filename` | string | Yes | Path to SQLite database file |
| `operation` | string | Yes | `executeQuery`, `insert`, `update`, `delete` |
| `query` | string | Depends | SQL query |

### Examples

#### Query SQLite File

```json
{
  "type": "n8n-nodes-base.sqlite",
  "parameters": {
    "filename": "/data/myapp.db",
    "operation": "executeQuery",
    "query": "SELECT * FROM settings"
  }
}
```

#### Insert Record

```json
{
  "type": "n8n-nodes-base.sqlite",
  "parameters": {
    "filename": "./local.db",
    "operation": "executeQuery",
    "query": "INSERT INTO events (name, data) VALUES ('{{ $json.event }}', '{{ JSON.stringify($json.data) }}')"
  }
}
```

---

## Common Patterns

### Parameterized Queries

Prevent SQL injection with expressions:

```json
{
  "query": "SELECT * FROM users WHERE id = {{ parseInt($json.userId) }}"
}
```

### Batch Insert

Loop through items:

```json
{
  "type": "n8n-nodes-base.postgres",
  "parameters": {
    "operation": "executeQuery",
    "query": "INSERT INTO items (name, value) VALUES ('{{ $json.name }}', {{ $json.value }})"
  }
}
```

### Check Query Results

Use Filter node after database query:

```json
{
  "type": "n8n-nodes-base.filter",
  "parameters": {
    "conditions": [
      {
        "leftValue": "={{ $json.length }}",
        "operator": "greaterThan",
        "rightValue": 0
      }
    ]
  }
}
```

---

## Quick Reference

| Node | Type | Default Port |
|------|------|--------------|
| PostgreSQL | `n8n-nodes-base.postgres` | 5432 |
| MySQL | `n8n-nodes-base.mysql` | 3306 |
| SQLite | `n8n-nodes-base.sqlite` | N/A (file) |

# Database Nodes

Database nodes enable reading from and writing to various database systems.

## PostgreSQL

### Query

```json
{
  "type": "n8n-nodes-base.postgres",
  "parameters": {
    "operation": "executeQuery",
    "query": "SELECT * FROM users WHERE active = true"
  },
  "credentials": {
    "postgres": {"id": "1", "name": "Production DB"}
  }
}
```

### Insert

```json
{
  "parameters": {
    "operation": "insert",
    "table": "users",
    "columns": "name, email, created_at",
    "returnFields": "id"
  }
}
```

Data from input:
```javascript
{{ $json.name }}, {{ $json.email }}, NOW()
```

### Update

```json
{
  "parameters": {
    "operation": "update",
    "table": "users",
    "updateKey": "id",
    "columns": "name, email, updated_at"
  }
}
```

### Parameterized Queries

Prevent SQL injection:

```json
{
  "parameters": {
    "operation": "executeQuery",
    "query": "SELECT * FROM users WHERE id = $1 AND status = $2",
    "queryParams": "={{ [$json.userId, $json.status] }}"
  }
}
```

### Credential Setup

```json
{
  "type": "postgres",
  "data": {
    "host": "db.example.com",
    "port": 5432,
    "database": "myapp",
    "user": "app_user",
    "password": "secure_password",
    "ssl": true
  }
}
```

## MySQL

### Query

```json
{
  "type": "n8n-nodes-base.mysql",
  "parameters": {
    "operation": "executeQuery",
    "query": "SELECT * FROM products WHERE price > ?"
  },
  "credentials": {
    "mysql": {"id": "1", "name": "MySQL DB"}
  }
}
```

### Insert

```json
{
  "parameters": {
    "operation": "insert",
    "table": "orders",
    "columns": "customer_id, product_id, quantity"
  }
}
```

### Bulk Insert

```json
{
  "parameters": {
    "operation": "insert",
    "table": "logs",
    "columns": "level, message, timestamp",
    "bulkOperation": true
  }
}
```

## SQLite

Local file-based database.

```json
{
  "type": "n8n-nodes-base.sqlite",
  "parameters": {
    "operation": "executeQuery",
    "query": "SELECT * FROM config"
  },
  "credentials": {
    "sqlite": {"id": "1", "name": "Local DB"}
  }
}
```

### Credential

```json
{
  "type": "sqlite",
  "data": {
    "database": "/path/to/database.db"
  }
}
```

## MongoDB

### Find Documents

```json
{
  "type": "n8n-nodes-base.mongodb",
  "parameters": {
    "operation": "find",
    "collection": "users",
    "query": "{\"active\": true}",
    "limit": 100
  },
  "credentials": {
    "mongodb": {"id": "1", "name": "MongoDB"}
  }
}
```

### Insert Document

```json
{
  "parameters": {
    "operation": "insert",
    "collection": "events",
    "fields": "={{ $json }}"
  }
}
```

### Update Document

```json
{
  "parameters": {
    "operation": "update",
    "collection": "users",
    "query": "{\"_id\": \"{{ $json.userId }}\"}",
    "update": "{\"$set\": {\"lastLogin\": \"{{ $now.toISO() }}\"}}"
  }
}
```

### Aggregate

```json
{
  "parameters": {
    "operation": "aggregate",
    "collection": "orders",
    "pipeline": "[{\"$group\": {\"_id\": \"$customerId\", \"total\": {\"$sum\": \"$amount\"}}}]"
  }
}
```

## Redis

### Get Value

```json
{
  "type": "n8n-nodes-base.redis",
  "parameters": {
    "operation": "get",
    "key": "user:{{ $json.userId }}"
  },
  "credentials": {
    "redis": {"id": "1", "name": "Redis Cache"}
  }
}
```

### Set Value

```json
{
  "parameters": {
    "operation": "set",
    "key": "session:{{ $json.sessionId }}",
    "value": "={{ JSON.stringify($json.data) }}",
    "expire": 3600
  }
}
```

### Hash Operations

```json
{
  "parameters": {
    "operation": "hset",
    "key": "user:{{ $json.id }}",
    "field": "lastSeen",
    "value": "={{ $now.toISO() }}"
  }
}
```

### List Operations

```json
{
  "parameters": {
    "operation": "lpush",
    "key": "queue:tasks",
    "value": "={{ JSON.stringify($json) }}"
  }
}
```

## Elasticsearch

### Search

```json
{
  "type": "n8n-nodes-base.elasticsearch",
  "parameters": {
    "operation": "search",
    "index": "logs",
    "query": {
      "bool": {
        "must": [
          {"match": {"level": "error"}},
          {"range": {"timestamp": {"gte": "now-1h"}}}
        ]
      }
    }
  }
}
```

### Index Document

```json
{
  "parameters": {
    "operation": "index",
    "index": "events",
    "body": "={{ $json }}"
  }
}
```

## Common Patterns

### Upsert Pattern

Insert or update based on existence:

```json
{
  "nodes": [
    {
      "id": "check-exists",
      "type": "n8n-nodes-base.postgres",
      "parameters": {
        "operation": "executeQuery",
        "query": "SELECT id FROM users WHERE email = $1",
        "queryParams": "={{ [$json.email] }}"
      }
    },
    {
      "id": "branch",
      "type": "n8n-nodes-base.if",
      "parameters": {
        "conditions": {
          "number": [{"value1": "={{ $json.length }}", "operation": "larger", "value2": 0}]
        }
      }
    },
    {
      "id": "update",
      "type": "n8n-nodes-base.postgres",
      "parameters": {"operation": "update"}
    },
    {
      "id": "insert",
      "type": "n8n-nodes-base.postgres",
      "parameters": {"operation": "insert"}
    }
  ]
}
```

### Pagination

Fetch all records:

```json
{
  "parameters": {
    "operation": "executeQuery",
    "query": "SELECT * FROM large_table LIMIT 1000 OFFSET {{ $json.offset || 0 }}"
  }
}
```

### Transaction Pattern

Use code node for transactions:

```javascript
const client = await pool.connect();
try {
  await client.query('BEGIN');
  await client.query('INSERT INTO orders ...');
  await client.query('UPDATE inventory ...');
  await client.query('COMMIT');
} catch (e) {
  await client.query('ROLLBACK');
  throw e;
} finally {
  client.release();
}
```

## Connection Pooling

m9m manages connection pools automatically. Configure in settings:

```yaml
database:
  maxConnections: 20
  idleTimeout: 30000
  connectionTimeout: 5000
```

## Error Handling

### Retry on Deadlock

```json
{
  "retryOnFail": true,
  "maxRetries": 3,
  "retryConditions": {
    "errorTypes": ["deadlock", "serialization_failure"]
  }
}
```

### Handle Not Found

```javascript
if ($json.length === 0) {
  throw new Error('Record not found');
}
```

## Security Best Practices

1. **Use parameterized queries** - Never concatenate user input
2. **Least privilege access** - Use read-only users when possible
3. **Encrypt connections** - Enable SSL/TLS
4. **Rotate credentials** - Change passwords regularly
5. **Audit queries** - Log database operations

## Performance Tips

1. **Index frequently queried columns**
2. **Use LIMIT for large result sets**
3. **Batch inserts** for multiple records
4. **Use connection pooling**
5. **Cache read-heavy queries** with Redis

## Next Steps

- [HTTP Nodes](http.md) - API integrations
- [Transform Nodes](transform.md) - Process query results
- [Error Handling](../user-guide/error-handling.md) - Handle database errors

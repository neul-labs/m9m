# API Endpoints Reference

Complete reference for all m9m REST API endpoints.

## Workflows

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/workflows` | List all workflows |
| POST | `/api/v1/workflows` | Create workflow |
| GET | `/api/v1/workflows/:id` | Get workflow |
| PUT | `/api/v1/workflows/:id` | Update workflow |
| DELETE | `/api/v1/workflows/:id` | Delete workflow |
| POST | `/api/v1/workflows/:id/activate` | Activate workflow |
| POST | `/api/v1/workflows/:id/deactivate` | Deactivate workflow |
| POST | `/api/v1/workflows/:id/execute` | Execute workflow |
| POST | `/api/v1/workflows/execute` | Execute inline workflow |
| GET | `/api/v1/workflows/:id/versions` | List workflow versions |
| POST | `/api/v1/workflows/:id/versions` | Create version |
| GET | `/api/v1/workflows/:id/versions/:version` | Get specific version |
| POST | `/api/v1/workflows/:id/restore/:version` | Restore version |

## Executions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/executions` | List executions |
| GET | `/api/v1/executions/:id` | Get execution |
| POST | `/api/v1/executions/:id/stop` | Stop execution |
| POST | `/api/v1/executions/:id/retry` | Retry execution |
| DELETE | `/api/v1/executions/:id` | Delete execution |
| GET | `/api/v1/executions/:id/logs` | Get execution logs |

## Credentials

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/credentials` | List credentials |
| POST | `/api/v1/credentials` | Create credential |
| GET | `/api/v1/credentials/:id` | Get credential metadata |
| PUT | `/api/v1/credentials/:id` | Update credential |
| DELETE | `/api/v1/credentials/:id` | Delete credential |
| POST | `/api/v1/credentials/:id/test` | Test credential |
| GET | `/api/v1/credential-types` | List credential types |

## Variables

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/variables` | List variables |
| GET | `/api/v1/variables/:key` | Get variable |
| PUT | `/api/v1/variables/:key` | Set variable |
| DELETE | `/api/v1/variables/:key` | Delete variable |
| POST | `/api/v1/variables/import` | Import variables |
| GET | `/api/v1/variables/export` | Export variables |

## Nodes

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/nodes` | List available nodes |
| GET | `/api/v1/nodes/:type` | Get node info |
| GET | `/api/v1/nodes/categories` | List node categories |

## Webhooks

| Method | Endpoint | Description |
|--------|----------|-------------|
| * | `/webhook/:path` | Webhook endpoint |
| * | `/webhook-test/:path` | Test webhook endpoint |
| GET | `/api/v1/webhooks` | List active webhooks |

## Tags

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/tags` | List tags |
| POST | `/api/v1/tags` | Create tag |
| PUT | `/api/v1/tags/:id` | Update tag |
| DELETE | `/api/v1/tags/:id` | Delete tag |

## Users (Enterprise)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/users` | List users |
| POST | `/api/v1/users` | Create user |
| GET | `/api/v1/users/:id` | Get user |
| PUT | `/api/v1/users/:id` | Update user |
| DELETE | `/api/v1/users/:id` | Delete user |
| GET | `/api/v1/users/me` | Get current user |

## Audit (Enterprise)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/audit` | List audit logs |
| GET | `/api/v1/audit/:id` | Get audit entry |

## System

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |
| GET | `/metrics` | Prometheus metrics |
| GET | `/api/v1/info` | System info |
| GET | `/api/v1/settings` | Get settings |
| PUT | `/api/v1/settings` | Update settings |

## Query Parameters

### Pagination

All list endpoints support:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 20 | Items per page (max: 100) |
| `offset` | int | 0 | Items to skip |

### Sorting

| Parameter | Type | Description |
|-----------|------|-------------|
| `sort` | string | Field to sort by |
| `order` | string | `asc` or `desc` |

Example:
```
GET /api/v1/workflows?sort=updatedAt&order=desc
```

### Filtering

Common filters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `search` | string | Text search |
| `active` | bool | Active status |
| `tags` | string | Comma-separated tag IDs |
| `createdAfter` | datetime | Created after date |
| `createdBefore` | datetime | Created before date |

## Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| `X-API-Key` | Yes* | API key authentication |
| `Authorization` | Yes* | Bearer token authentication |
| `Content-Type` | Yes | `application/json` for POST/PUT |
| `Accept` | No | `application/json` (default) |

*One authentication method required

## Response Headers

| Header | Description |
|--------|-------------|
| `X-Request-Id` | Unique request identifier |
| `X-RateLimit-Limit` | Rate limit maximum |
| `X-RateLimit-Remaining` | Remaining requests |
| `X-RateLimit-Reset` | Reset timestamp |

## Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 204 | No Content |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 422 | Validation Error |
| 429 | Rate Limited |
| 500 | Internal Error |

## Next Steps

- [REST API Overview](rest-api.md) - API usage guide
- [Authentication](authentication.md) - Auth setup
- [Webhooks](webhooks.md) - Webhook configuration

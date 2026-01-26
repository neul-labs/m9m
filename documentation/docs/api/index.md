# API Reference

m9m provides a comprehensive REST API for workflow management.

## Base URL

```
http://localhost:8080/api/v1
```

## API Endpoints Overview

| Resource | Endpoints |
|----------|-----------|
| [Workflows](workflows.md) | CRUD operations for workflows |
| [Executions](executions.md) | Manage workflow executions |
| [Jobs](jobs.md) | Job queue management |
| [Schedules](schedules.md) | Cron scheduling |
| [Credentials](credentials.md) | Credential management |
| [Webhooks](webhooks.md) | Webhook configuration |

## Quick Reference

### Workflows

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/workflows` | List workflows |
| POST | `/workflows` | Create workflow |
| GET | `/workflows/{id}` | Get workflow |
| PUT | `/workflows/{id}` | Update workflow |
| DELETE | `/workflows/{id}` | Delete workflow |
| POST | `/workflows/{id}/execute` | Execute workflow |
| POST | `/workflows/{id}/execute-async` | Execute asynchronously |

### Executions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/executions` | List executions |
| GET | `/executions/{id}` | Get execution |
| DELETE | `/executions/{id}` | Delete execution |
| POST | `/executions/{id}/retry` | Retry execution |
| POST | `/executions/{id}/cancel` | Cancel execution |

### Jobs

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/jobs` | List jobs |
| GET | `/jobs/{id}` | Get job details |

### Health & Monitoring

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |
| GET | `/metrics` | Prometheus metrics |

## Request Format

### Headers

```http
Content-Type: application/json
Authorization: Bearer <token>
```

### Request Body

```json
{
  "field": "value"
}
```

## Response Format

### Success Response

```json
{
  "id": "resource-id",
  "field": "value"
}
```

### Error Response

```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": {}
}
```

## HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 202 | Accepted (async operation queued) |
| 204 | No Content (successful deletion) |
| 400 | Bad Request (validation error) |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 422 | Unprocessable Entity |
| 429 | Too Many Requests |
| 500 | Internal Server Error |

## Pagination

List endpoints support pagination:

```
GET /api/v1/workflows?offset=0&limit=20
```

| Parameter | Default | Max | Description |
|-----------|---------|-----|-------------|
| `offset` | 0 | - | Starting position |
| `limit` | 20 | 100 | Items per page |

Response includes pagination info:

```json
{
  "data": [...],
  "total": 150,
  "offset": 0,
  "limit": 20
}
```

## Filtering

List endpoints support filtering:

```
GET /api/v1/workflows?active=true&search=daily
GET /api/v1/executions?status=failed&workflowId=abc123
```

## Example: Complete Workflow Lifecycle

```bash
# 1. Create workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @workflow.json

# Response: {"id": "wf-123", ...}

# 2. Execute workflow
curl -X POST http://localhost:8080/api/v1/workflows/wf-123/execute \
  -H "Content-Type: application/json" \
  -d '{"inputData": [{"json": {"key": "value"}}]}'

# Response: {"id": "exec-456", "status": "success", ...}

# 3. Check execution
curl http://localhost:8080/api/v1/executions/exec-456

# 4. Delete workflow
curl -X DELETE http://localhost:8080/api/v1/workflows/wf-123
```

## See Also

- [Authentication](authentication.md) - API authentication methods
- [CLI Reference](../cli/index.md) - Command-line interface

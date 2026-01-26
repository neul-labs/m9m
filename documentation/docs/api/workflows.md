# Workflows API

API endpoints for managing workflows.

## List Workflows

Retrieve all workflows with optional filtering.

```http
GET /api/v1/workflows
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `offset` | integer | 0 | Pagination offset |
| `limit` | integer | 20 | Items per page (max 100) |
| `active` | boolean | - | Filter by active status |
| `search` | string | - | Search by name |

### Example Request

```bash
curl "http://localhost:8080/api/v1/workflows?active=true&limit=10" \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Daily Report",
      "active": true,
      "nodes": [...],
      "connections": {...},
      "createdAt": "2024-01-20T10:00:00Z",
      "updatedAt": "2024-01-26T15:30:00Z"
    }
  ],
  "total": 25,
  "offset": 0,
  "limit": 10
}
```

---

## Get Workflow

Retrieve a single workflow by ID.

```http
GET /api/v1/workflows/{id}
```

### Example Request

```bash
curl http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Daily Report",
  "description": "Sends daily summary reports",
  "active": true,
  "nodes": [
    {
      "id": "start-1",
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    },
    {
      "id": "http-1",
      "name": "Fetch Data",
      "type": "n8n-nodes-base.httpRequest",
      "position": [450, 300],
      "parameters": {
        "url": "https://api.example.com/data",
        "method": "GET"
      }
    }
  ],
  "connections": {
    "Start": {
      "main": [[{"node": "Fetch Data", "type": "main", "index": 0}]]
    }
  },
  "settings": {},
  "tags": ["reports", "daily"],
  "createdAt": "2024-01-20T10:00:00Z",
  "updatedAt": "2024-01-26T15:30:00Z"
}
```

---

## Create Workflow

Create a new workflow.

```http
POST /api/v1/workflows
```

### Request Body

```json
{
  "name": "My Workflow",
  "description": "Optional description",
  "nodes": [
    {
      "id": "start-1",
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    }
  ],
  "connections": {},
  "settings": {},
  "tags": ["tag1", "tag2"]
}
```

### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Workflow",
    "nodes": [
      {
        "id": "start",
        "name": "Start",
        "type": "n8n-nodes-base.start",
        "position": [250, 300],
        "parameters": {}
      }
    ],
    "connections": {}
  }'
```

### Response

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "name": "My Workflow",
  "active": false,
  "nodes": [...],
  "connections": {},
  "createdAt": "2024-01-26T16:00:00Z",
  "updatedAt": "2024-01-26T16:00:00Z"
}
```

---

## Update Workflow

Update an existing workflow.

```http
PUT /api/v1/workflows/{id}
```

### Request Body

Full workflow object (replaces existing).

### Example Request

```bash
curl -X PUT http://localhost:8080/api/v1/workflows/660e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Workflow",
    "nodes": [...],
    "connections": {...}
  }'
```

---

## Partial Update

Update specific workflow fields.

```http
PATCH /api/v1/workflows/{id}
```

### Request Body

Only fields to update.

### Example Request

```bash
curl -X PATCH http://localhost:8080/api/v1/workflows/660e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Renamed Workflow",
    "description": "New description"
  }'
```

---

## Delete Workflow

Delete a workflow.

```http
DELETE /api/v1/workflows/{id}
```

### Example Request

```bash
curl -X DELETE http://localhost:8080/api/v1/workflows/660e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer <token>"
```

### Response

```
204 No Content
```

---

## Activate Workflow

Enable workflow triggers.

```http
POST /api/v1/workflows/{id}/activate
```

### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/activate \
  -H "Authorization: Bearer <token>"
```

### Response

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "active": true
}
```

---

## Deactivate Workflow

Disable workflow triggers.

```http
POST /api/v1/workflows/{id}/deactivate
```

---

## Execute Workflow (Sync)

Execute a workflow synchronously and wait for results.

```http
POST /api/v1/workflows/{id}/execute
```

### Request Body

```json
{
  "inputData": [
    {
      "json": {
        "key": "value"
      }
    }
  ]
}
```

### Example Request

```bash
curl -X POST http://localhost:8080/api/v1/workflows/550e8400-e29b-41d4-a716-446655440000/execute \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "inputData": [{"json": {"name": "John"}}]
  }'
```

### Response

```json
{
  "id": "exec-123456",
  "workflowId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "success",
  "mode": "manual",
  "startedAt": "2024-01-26T16:00:00Z",
  "finishedAt": "2024-01-26T16:00:01Z",
  "data": [
    {
      "json": {
        "result": "processed",
        "name": "John"
      }
    }
  ]
}
```

---

## Execute Workflow (Async)

Queue workflow for execution and return immediately.

```http
POST /api/v1/workflows/{id}/execute-async
```

### Response

```json
{
  "jobId": "job-789012",
  "status": "pending",
  "message": "Workflow execution queued"
}
```

Use the [Jobs API](jobs.md) to check execution status.

---

## Duplicate Workflow

Create a copy of a workflow.

```http
POST /api/v1/workflows/{id}/duplicate
```

### Request Body (Optional)

```json
{
  "name": "Copy of My Workflow"
}
```

### Response

```json
{
  "id": "new-workflow-id",
  "name": "Copy of My Workflow",
  ...
}
```

---

## Error Responses

### 400 Bad Request

```json
{
  "error": "Validation failed",
  "code": "VALIDATION_ERROR",
  "details": {
    "nodes[0].type": "invalid node type"
  }
}
```

### 404 Not Found

```json
{
  "error": "Workflow not found",
  "code": "NOT_FOUND"
}
```

### 409 Conflict

```json
{
  "error": "Workflow with this name already exists",
  "code": "CONFLICT"
}
```

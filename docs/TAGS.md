# n8n-go Tags System

## Overview

The tags system in n8n-go provides workflow organization and categorization capabilities. Tags allow users to group, filter, and manage workflows more effectively, especially in environments with many workflows.

## Table of Contents

1. [Features](#features)
2. [Architecture](#architecture)
3. [API Reference](#api-reference)
4. [Usage Examples](#usage-examples)
5. [Storage](#storage)
6. [Best Practices](#best-practices)

---

## Features

### Core Capabilities

- **Tag Management**: Create, read, update, and delete tags
- **Workflow Association**: Associate multiple tags with workflows
- **Color Coding**: Visual organization with hex color codes
- **Search & Filter**: Find tags by name or search term
- **Bulk Operations**: Set all tags for a workflow at once
- **Validation**: Automatic validation of tag names and colors
- **Duplicate Prevention**: Ensure unique tag names

### Technical Features

- **Multiple Storage Backends**: Memory and persistent storage
- **Atomic Operations**: Thread-safe tag operations
- **RESTful API**: Standard HTTP methods for all operations
- **n8n Compatibility**: API compatible with n8n frontend
- **Pagination**: Efficient listing with limit/offset

---

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────┐
│                     Tag Handler                          │
│              (HTTP API Endpoints)                        │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                    Tag Manager                           │
│              (Business Logic)                            │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                   Tag Storage                            │
│  ┌──────────────────┐    ┌──────────────────────────┐  │
│  │ Memory Storage   │    │ Persistent Storage       │  │
│  │ (In-memory map)  │    │ (BadgerDB/Postgres/etc)  │  │
│  └──────────────────┘    └──────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Data Models

#### Tag
```go
type Tag struct {
    ID        string    `json:"id"`         // Unique identifier
    Name      string    `json:"name"`       // Tag name
    Color     string    `json:"color"`      // Hex color code (optional)
    CreatedAt time.Time `json:"createdAt"`  // Creation timestamp
    UpdatedAt time.Time `json:"updatedAt"`  // Last update timestamp
}
```

#### WorkflowTag (Association)
```go
type WorkflowTag struct {
    WorkflowID string    `json:"workflowId"` // Workflow identifier
    TagID      string    `json:"tagId"`      // Tag identifier
    CreatedAt  time.Time `json:"createdAt"`  // Association timestamp
}
```

---

## API Reference

### Tag Endpoints

#### 1. List Tags

**Endpoint**: `GET /api/v1/tags`

**Query Parameters**:
- `search` (string, optional): Search term for tag names
- `limit` (integer, optional): Maximum number of results (default: 50)
- `offset` (integer, optional): Number of results to skip (default: 0)

**Request**:
```bash
curl -X GET "http://localhost:8080/api/v1/tags?search=prod&limit=20&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "data": [
    {
      "id": "tag_abc123",
      "name": "production",
      "color": "#FF5733",
      "createdAt": "2025-01-10T00:00:00Z",
      "updatedAt": "2025-01-10T00:00:00Z"
    },
    {
      "id": "tag_def456",
      "name": "prod-urgent",
      "color": "#FF0000",
      "createdAt": "2025-01-10T00:00:00Z",
      "updatedAt": "2025-01-10T00:00:00Z"
    }
  ],
  "total": 2,
  "count": 2,
  "limit": 20,
  "offset": 0
}
```

**Status Codes**:
- `200 OK`: Success
- `500 Internal Server Error`: Server error

---

#### 2. Create Tag

**Endpoint**: `POST /api/v1/tags`

**Request Body**:
```json
{
  "name": "production",
  "color": "#FF5733"
}
```

**Request**:
```bash
curl -X POST http://localhost:8080/api/v1/tags \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "production",
    "color": "#FF5733"
  }'
```

**Response**:
```json
{
  "id": "tag_abc123",
  "name": "production",
  "color": "#FF5733",
  "createdAt": "2025-01-10T00:00:00Z",
  "updatedAt": "2025-01-10T00:00:00Z"
}
```

**Validation Rules**:
- `name` is required
- `name` must be unique
- `name` maximum length: 100 characters
- `color` must be valid hex color code (e.g., "#FF5733")
- `color` is optional

**Status Codes**:
- `201 Created`: Tag created successfully
- `400 Bad Request`: Validation error
- `500 Internal Server Error`: Server error

---

#### 3. Get Tag

**Endpoint**: `GET /api/v1/tags/:id`

**Request**:
```bash
curl -X GET http://localhost:8080/api/v1/tags/tag_abc123 \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "id": "tag_abc123",
  "name": "production",
  "color": "#FF5733",
  "createdAt": "2025-01-10T00:00:00Z",
  "updatedAt": "2025-01-10T00:00:00Z"
}
```

**Status Codes**:
- `200 OK`: Success
- `404 Not Found`: Tag not found
- `500 Internal Server Error`: Server error

---

#### 4. Update Tag

**Endpoint**: `PATCH /api/v1/tags/:id` or `PUT /api/v1/tags/:id`

**Request Body** (all fields optional):
```json
{
  "name": "production-updated",
  "color": "#00FF00"
}
```

**Request**:
```bash
curl -X PATCH http://localhost:8080/api/v1/tags/tag_abc123 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "production-updated",
    "color": "#00FF00"
  }'
```

**Response**:
```json
{
  "id": "tag_abc123",
  "name": "production-updated",
  "color": "#00FF00",
  "createdAt": "2025-01-10T00:00:00Z",
  "updatedAt": "2025-01-10T01:00:00Z"
}
```

**Status Codes**:
- `200 OK`: Success
- `400 Bad Request`: Validation error
- `404 Not Found`: Tag not found
- `500 Internal Server Error`: Server error

---

#### 5. Delete Tag

**Endpoint**: `DELETE /api/v1/tags/:id`

**Request**:
```bash
curl -X DELETE http://localhost:8080/api/v1/tags/tag_abc123 \
  -H "Authorization: Bearer $TOKEN"
```

**Response**: `204 No Content` (empty body)

**Status Codes**:
- `204 No Content`: Tag deleted successfully
- `404 Not Found`: Tag not found
- `409 Conflict`: Tag is in use by workflows
- `500 Internal Server Error`: Server error

**Note**: Tags cannot be deleted if they are currently assigned to any workflows. Remove the tag from all workflows first.

---

### Workflow Tag Endpoints

#### 6. Get Workflow Tags

**Endpoint**: `GET /api/v1/workflows/:id/tags`

**Request**:
```bash
curl -X GET http://localhost:8080/api/v1/workflows/wf_xyz789/tags \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "data": [
    {
      "id": "tag_abc123",
      "name": "production",
      "color": "#FF5733",
      "createdAt": "2025-01-10T00:00:00Z",
      "updatedAt": "2025-01-10T00:00:00Z"
    },
    {
      "id": "tag_def456",
      "name": "critical",
      "color": "#FF0000",
      "createdAt": "2025-01-10T00:00:00Z",
      "updatedAt": "2025-01-10T00:00:00Z"
    }
  ],
  "count": 2
}
```

**Status Codes**:
- `200 OK`: Success
- `500 Internal Server Error`: Server error

---

#### 7. Set Workflow Tags

**Endpoint**: `POST /api/v1/workflows/:id/tags` or `PUT /api/v1/workflows/:id/tags`

**Description**: Replaces all tags for a workflow with the specified list.

**Request Body**:
```json
{
  "tagIds": ["tag_abc123", "tag_def456"]
}
```

**Request**:
```bash
curl -X POST http://localhost:8080/api/v1/workflows/wf_xyz789/tags \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "tagIds": ["tag_abc123", "tag_def456"]
  }'
```

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": "tag_abc123",
      "name": "production",
      "color": "#FF5733",
      "createdAt": "2025-01-10T00:00:00Z",
      "updatedAt": "2025-01-10T00:00:00Z"
    },
    {
      "id": "tag_def456",
      "name": "critical",
      "color": "#FF0000",
      "createdAt": "2025-01-10T00:00:00Z",
      "updatedAt": "2025-01-10T00:00:00Z"
    }
  ],
  "count": 2
}
```

**Status Codes**:
- `200 OK`: Success
- `400 Bad Request`: Invalid tag ID(s)
- `500 Internal Server Error`: Server error

**Note**: Passing an empty array `[]` removes all tags from the workflow.

---

#### 8. Add Workflow Tag

**Endpoint**: `POST /api/v1/workflows/:id/tags/:tagId`

**Description**: Adds a single tag to a workflow (idempotent).

**Request**:
```bash
curl -X POST http://localhost:8080/api/v1/workflows/wf_xyz789/tags/tag_abc123 \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "success": true,
  "message": "Tag added to workflow"
}
```

**Status Codes**:
- `200 OK`: Success (including if tag already added)
- `404 Not Found`: Tag not found
- `400 Bad Request`: Invalid request
- `500 Internal Server Error`: Server error

---

#### 9. Remove Workflow Tag

**Endpoint**: `DELETE /api/v1/workflows/:id/tags/:tagId`

**Description**: Removes a single tag from a workflow (idempotent).

**Request**:
```bash
curl -X DELETE http://localhost:8080/api/v1/workflows/wf_xyz789/tags/tag_abc123 \
  -H "Authorization: Bearer $TOKEN"
```

**Response**: `204 No Content` (empty body)

**Status Codes**:
- `204 No Content`: Success (including if tag wasn't assigned)
- `400 Bad Request`: Invalid request
- `500 Internal Server Error`: Server error

---

## Usage Examples

### Example 1: Create Tags for Environment Classification

```bash
# Create production tag
curl -X POST http://localhost:8080/api/v1/tags \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "production", "color": "#FF5733"}'

# Create staging tag
curl -X POST http://localhost:8080/api/v1/tags \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "staging", "color": "#FFA500"}'

# Create development tag
curl -X POST http://localhost:8080/api/v1/tags \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "development", "color": "#00FF00"}'
```

### Example 2: Tag a Workflow

```bash
# Get tag IDs
PROD_TAG_ID=$(curl -s http://localhost:8080/api/v1/tags?search=production | jq -r '.data[0].id')
CRITICAL_TAG_ID=$(curl -s http://localhost:8080/api/v1/tags?search=critical | jq -r '.data[0].id')

# Set workflow tags
curl -X POST http://localhost:8080/api/v1/workflows/wf_xyz789/tags \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"tagIds\": [\"$PROD_TAG_ID\", \"$CRITICAL_TAG_ID\"]}"
```

### Example 3: Filter Workflows by Tag (Client-Side)

```bash
# Get all workflows with their tags
curl -X GET http://localhost:8080/api/v1/workflows \
  -H "Authorization: Bearer $TOKEN" | jq '.'

# For each workflow, get its tags and filter
# This example shows the concept - implement in your application
```

### Example 4: Update Tag Color

```bash
# Update production tag to red
curl -X PATCH http://localhost:8080/api/v1/tags/tag_abc123 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"color": "#FF0000"}'
```

### Example 5: Clean Up Unused Tags

```bash
# List all tags
TAGS=$(curl -s http://localhost:8080/api/v1/tags -H "Authorization: Bearer $TOKEN")

# For each tag, try to delete it
# Tags in use will return 409 Conflict and won't be deleted
echo "$TAGS" | jq -r '.data[].id' | while read TAG_ID; do
  echo "Attempting to delete $TAG_ID..."
  curl -X DELETE http://localhost:8080/api/v1/tags/$TAG_ID \
    -H "Authorization: Bearer $TOKEN" \
    -w "\nStatus: %{http_code}\n"
done
```

---

## Storage

### Storage Backends

The tags system supports multiple storage backends:

#### Memory Storage
- **Use Case**: Development, testing, or temporary deployments
- **Persistence**: Data lost on restart
- **Performance**: Fastest
- **Configuration**: Default, no setup required

```go
tagStorage := tags.NewMemoryTagStorage()
```

#### Persistent Storage
- **Use Case**: Production deployments
- **Persistence**: Data persists across restarts
- **Performance**: Depends on backend (BadgerDB, PostgreSQL, SQLite)
- **Configuration**: Uses existing workflow storage backend

```go
tagStorage := tags.NewPersistentTagStorage(workflowStorage)
```

### Storage Keys

Persistent storage uses the following key patterns:

| Key Pattern | Purpose | Example |
|-------------|---------|---------|
| `tag:{id}` | Tag data | `tag:tag_abc123` |
| `tag_name:{name}` | Name index | `tag_name:production` |
| `workflow_tag:{workflowId}:{tagId}` | Workflow-tag association | `workflow_tag:wf_xyz:tag_abc` |

---

## Best Practices

### Naming Conventions

**Do**:
- Use descriptive, meaningful names
- Use lowercase with hyphens: `production`, `data-pipeline`
- Keep names concise (under 30 characters when possible)
- Use consistent naming across your organization

**Don't**:
- Use special characters or spaces
- Use overly long names (>100 characters limit)
- Create duplicate or similar names

### Color Coding

**Suggested Color Scheme**:
```
Environment:
- Production:   #FF5733 (Red)
- Staging:      #FFA500 (Orange)
- Development:  #00FF00 (Green)

Priority:
- Critical:     #FF0000 (Bright Red)
- High:         #FF6600 (Dark Orange)
- Medium:       #FFFF00 (Yellow)
- Low:          #0000FF (Blue)

Category:
- Data Pipeline: #9B59B6 (Purple)
- API:          #3498DB (Light Blue)
- Scheduled:    #1ABC9C (Teal)
- Manual:       #95A5A6 (Gray)
```

### Organization Strategies

#### By Environment
```
tags: [production, staging, development]
```

#### By Team
```
tags: [team-data, team-api, team-frontend]
```

#### By Function
```
tags: [etl, monitoring, notifications, reports]
```

#### By Priority
```
tags: [critical, high-priority, low-priority]
```

#### Combined Approach
```
tags: [production, team-data, critical, etl]
```

### Performance Considerations

1. **Limit Tags per Workflow**: Recommended maximum of 10 tags per workflow
2. **Use Pagination**: When listing tags, use appropriate limit/offset
3. **Cache Tag Data**: Cache frequently accessed tag lists in your application
4. **Batch Operations**: Use `SetWorkflowTags` instead of multiple `AddWorkflowTag` calls
5. **Index Cleanup**: Periodically review and delete unused tags

### Security Considerations

1. **Authentication Required**: All tag endpoints require authentication
2. **Authorization**: Implement workflow-level permissions if needed
3. **Input Validation**: Tag names are automatically validated
4. **Audit Trail**: Log tag creation/modification in production

---

## Integration Examples

### JavaScript/TypeScript

```typescript
import axios from 'axios';

const API_URL = 'http://localhost:8080/api/v1';
const TOKEN = 'your-jwt-token';

// Create tag
async function createTag(name: string, color: string) {
  const response = await axios.post(
    `${API_URL}/tags`,
    { name, color },
    { headers: { Authorization: `Bearer ${TOKEN}` } }
  );
  return response.data;
}

// Get workflow tags
async function getWorkflowTags(workflowId: string) {
  const response = await axios.get(
    `${API_URL}/workflows/${workflowId}/tags`,
    { headers: { Authorization: `Bearer ${TOKEN}` } }
  );
  return response.data.data;
}

// Set workflow tags
async function setWorkflowTags(workflowId: string, tagIds: string[]) {
  const response = await axios.post(
    `${API_URL}/workflows/${workflowId}/tags`,
    { tagIds },
    { headers: { Authorization: `Bearer ${TOKEN}` } }
  );
  return response.data;
}
```

### Python

```python
import requests

API_URL = "http://localhost:8080/api/v1"
TOKEN = "your-jwt-token"
HEADERS = {"Authorization": f"Bearer {TOKEN}"}

# Create tag
def create_tag(name: str, color: str):
    response = requests.post(
        f"{API_URL}/tags",
        json={"name": name, "color": color},
        headers=HEADERS
    )
    return response.json()

# Get workflow tags
def get_workflow_tags(workflow_id: str):
    response = requests.get(
        f"{API_URL}/workflows/{workflow_id}/tags",
        headers=HEADERS
    )
    return response.json()["data"]

# Set workflow tags
def set_workflow_tags(workflow_id: str, tag_ids: list):
    response = requests.post(
        f"{API_URL}/workflows/{workflow_id}/tags",
        json={"tagIds": tag_ids},
        headers=HEADERS
    )
    return response.json()
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

const (
    APIURL = "http://localhost:8080/api/v1"
    Token  = "your-jwt-token"
)

type Tag struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    Color     string `json:"color"`
    CreatedAt string `json:"createdAt"`
    UpdatedAt string `json:"updatedAt"`
}

func createTag(name, color string) (*Tag, error) {
    payload := map[string]string{
        "name":  name,
        "color": color,
    }
    body, _ := json.Marshal(payload)

    req, _ := http.NewRequest("POST", APIURL+"/tags", bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer "+Token)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var tag Tag
    json.NewDecoder(resp.Body).Decode(&tag)
    return &tag, nil
}
```

---

## Troubleshooting

### Common Issues

#### 1. Tag Name Already Exists

**Error**: `400 Bad Request: tag with this name already exists`

**Solution**: Tag names must be unique. Either:
- Use a different name
- Update the existing tag instead of creating a new one

#### 2. Cannot Delete Tag

**Error**: `409 Conflict: tag is in use by workflows`

**Solution**: Remove the tag from all workflows before deleting:

```bash
# First, remove tag from all workflows
# Then delete the tag
curl -X DELETE http://localhost:8080/api/v1/tags/tag_abc123 \
  -H "Authorization: Bearer $TOKEN"
```

#### 3. Tag Not Found

**Error**: `404 Not Found: Tag not found`

**Solution**: Verify the tag ID is correct by listing all tags:

```bash
curl -X GET http://localhost:8080/api/v1/tags \
  -H "Authorization: Bearer $TOKEN"
```

---

## Compatibility

### n8n Compatibility

The n8n-go tags API is designed to be compatible with the n8n frontend. Key compatibility points:

- ✅ Same endpoint paths
- ✅ Same request/response formats
- ✅ Same validation rules
- ✅ Same HTTP status codes
- ✅ Same error messages

### Version Information

- **Introduced**: n8n-go v0.4.0
- **API Version**: v1
- **Status**: Stable

---

## Future Enhancements

Planned features for future releases:

1. **Tag Hierarchies**: Parent-child tag relationships
2. **Tag Icons**: Custom icons for visual identification
3. **Tag Permissions**: Role-based access control for tag management
4. **Tag Analytics**: Usage statistics and reports
5. **Tag Import/Export**: Bulk import/export of tag configurations
6. **Smart Tags**: Auto-tagging based on workflow properties
7. **Tag Suggestions**: AI-powered tag recommendations

---

**Last Updated**: November 10, 2025
**Version**: 1.0
**Status**: Production Ready

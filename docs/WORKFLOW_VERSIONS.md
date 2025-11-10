# Workflow Versions

The n8n-go workflow version system provides comprehensive version control for workflows, enabling teams to track changes, compare versions, and restore to previous states with ease.

## Overview

The version system automatically tracks workflow changes and allows you to:
- Create snapshot versions of workflows
- Track detailed changes between versions
- Compare different versions
- Restore workflows to previous versions with automatic backup
- Delete old versions (except current version)
- Tag and categorize versions

## Architecture

The version system consists of four main components:

### 1. Version Model (`version.go`)
Defines the data structures for workflow versions:
- **WorkflowVersion**: Complete workflow snapshot with metadata
- **VersionComparison**: Detailed comparison between two versions
- **VersionChanges**: Breakdown of specific changes
- **Request/Response Types**: API contracts

### 2. Version Storage (`version_storage.go`)
Provides persistence layer with two implementations:
- **MemoryVersionStorage**: In-memory storage for development/testing
- **PersistentVersionStorage**: Persistent storage using WorkflowStorage backend
- Supports all storage backends (Memory, BadgerDB, PostgreSQL, SQLite)

### 3. Version Manager (`version_manager.go`)
Contains business logic for:
- Creating versions with automatic change detection
- Comparing versions
- Restoring workflows with automatic backup
- Managing version lifecycle

### 4. Version Handler (`version_handler.go`)
Exposes REST API endpoints for version operations

## API Endpoints

All endpoints are under `/api/v1/workflows/{id}/versions`

### List Versions
```bash
GET /api/v1/workflows/{id}/versions?limit=50&offset=0&author=user123&tags=release

Response:
{
  "data": [...],
  "total": 10,
  "count": 10,
  "limit": 50,
  "offset": 0
}
```

### Create Version
```bash
POST /api/v1/workflows/{id}/versions

Request:
{
  "versionTag": "v1.0.0",           # Optional, auto-generated if not provided
  "description": "Initial release",  # Optional
  "tags": ["release", "production"]  # Optional
}

Response:
{
  "id": "ver_1234567890",
  "workflowId": "workflow_123",
  "versionTag": "v1.0.0",
  "versionNum": 1,
  "workflow": { ... },              # Complete workflow snapshot
  "author": "system",
  "description": "Initial release",
  "changes": [
    "Initial version"
  ],
  "isCurrent": true,
  "createdAt": "2025-11-10T13:40:39Z",
  "tags": ["release", "production"]
}
```

### Get Version
```bash
GET /api/v1/workflows/{id}/versions/{versionId}

Response:
{
  "id": "ver_1234567890",
  "workflowId": "workflow_123",
  "versionTag": "v1.0.0",
  ...
}
```

### Delete Version
```bash
DELETE /api/v1/workflows/{id}/versions/{versionId}

Response: 204 No Content

Note: Cannot delete the current version
```

### Restore Version
```bash
POST /api/v1/workflows/{id}/versions/{versionId}/restore

Request:
{
  "createBackup": true,              # Optional, default: true
  "description": "Rollback changes"  # Optional
}

Response:
{
  "message": "Workflow restored successfully",
  "restoredFrom": { ... },          # Version that was restored
  "backupCreated": true
}
```

### Compare Versions
```bash
GET /api/v1/workflows/{id}/versions/compare?from=ver_123&to=ver_456

Response:
{
  "fromVersion": { ... },
  "toVersion": { ... },
  "changes": {
    "nodesAdded": ["HTTP Request", "Set"],
    "nodesRemoved": ["Old Node"],
    "nodesModified": ["Start"],
    "connectionsChanged": true,
    "settingsChanged": false,
    "summary": "2 nodes added, 1 removed, 1 modified, connections changed"
  }
}
```

## Usage Examples

### 1. Create Initial Version
```bash
# Create a workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Workflow",
    "active": false,
    "nodes": [...],
    "connections": {}
  }'

# Create initial version
curl -X POST http://localhost:8080/api/v1/workflows/{id}/versions \
  -H "Content-Type: application/json" \
  -d '{
    "versionTag": "v1.0.0",
    "description": "Initial release",
    "tags": ["release"]
  }'
```

### 2. Update Workflow and Create New Version
```bash
# Update workflow
curl -X PUT http://localhost:8080/api/v1/workflows/{id} \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Workflow - Updated",
    "active": true,
    "nodes": [... new nodes ...],
    "connections": {}
  }'

# Create new version (automatically detects changes)
curl -X POST http://localhost:8080/api/v1/workflows/{id}/versions \
  -H "Content-Type: application/json" \
  -d '{
    "versionTag": "v1.1.0",
    "description": "Added HTTP Request node",
    "tags": ["feature"]
  }'
```

### 3. Restore to Previous Version
```bash
# Restore to v1.0.0 (automatically creates backup)
curl -X POST http://localhost:8080/api/v1/workflows/{id}/versions/{versionId}/restore \
  -H "Content-Type: application/json" \
  -d '{
    "createBackup": true,
    "description": "Rolling back changes"
  }'
```

### 4. Compare Versions
```bash
# Compare two versions
curl -X GET "http://localhost:8080/api/v1/workflows/{id}/versions/compare?from={versionId1}&to={versionId2}"
```

### 5. Delete Old Version
```bash
# Delete a non-current version
curl -X DELETE http://localhost:8080/api/v1/workflows/{id}/versions/{versionId}
```

## Change Detection

The system automatically detects and tracks:

### Workflow-Level Changes
- Name changes
- Description updates
- Active status changes
- Settings modifications

### Node-Level Changes
- **Added Nodes**: New nodes added to workflow
- **Removed Nodes**: Nodes removed from workflow
- **Modified Nodes**: Node type or parameter changes

### Connection Changes
- Modifications to node connections
- New connections added
- Connections removed

### Example Change Output
```json
{
  "changes": [
    "Renamed from 'Old Name' to 'New Name'",
    "Activated workflow",
    "Added node: HTTP Request",
    "Added node: Set",
    "Removed node: Old Node",
    "Changed node type: Data Transform",
    "Connections modified",
    "Settings updated"
  ]
}
```

## Version Numbering

Versions use two numbering systems:

1. **Sequential Numbers**: Auto-incrementing integers (1, 2, 3, ...)
   - Guaranteed to be sequential
   - Never reused, even if versions are deleted

2. **Version Tags**: User-defined strings (e.g., "v1.0.0", "2023-11-prod")
   - Optional, auto-generated if not provided
   - Can follow semantic versioning or any naming scheme

## Current Version Tracking

- Only one version is marked as "current" at a time
- When creating a new version, it automatically becomes current
- When restoring a version, a new version is created and marked as current
- Cannot delete the current version

## Backup Strategy

When restoring a version:
1. **Automatic Backup** (if `createBackup: true`):
   - Current workflow state is saved as a backup version
   - Tagged with timestamp and "backup", "auto" tags
   - Prevents data loss during restoration

2. **Restore Operation**:
   - Workflow is updated with the restored version's content
   - New version is created and marked as current
   - Original version remains unchanged

Example version timeline after restore:
```
v1 (initial)
v2 (feature added)
v3 (backup-before-restore) ← automatic backup
v4 (restored-from-v1) ← current
```

## Storage Backend Support

The version system works with all storage backends:

### Memory Storage
- Fast, ephemeral storage
- Data lost on restart
- Ideal for development/testing

### BadgerDB Storage
- Persistent key-value storage
- Fast, embedded database
- Ideal for single-node deployments

### PostgreSQL Storage
- Relational database storage
- Production-ready
- Supports clustering

### SQLite Storage
- File-based SQL database
- Easy deployment
- Good for small/medium deployments

## Integration

### In Your Application

The version system is automatically initialized in both single-node and cluster modes:

```go
// Single-node mode (main.go)
versionStorage := versions.NewMemoryVersionStorage(store)
versionManager := versions.NewVersionManager(versionStorage, store)
versionHandler := versions.NewVersionHandler(versionManager)
versionHandler.RegisterRoutes(router)
```

### With Authentication

When authentication is enabled, the system will:
- Extract user information from JWT tokens
- Track version author from authenticated user
- Apply permission checks (future feature)

Currently uses placeholder "system" user when auth context is not available.

## Best Practices

### 1. Version Naming
```bash
# Semantic versioning
v1.0.0, v1.1.0, v2.0.0

# Date-based
2025-11-10-prod, 2025-11-11-hotfix

# Descriptive
initial-release, added-slack-integration
```

### 2. Version Descriptions
```bash
# Good descriptions
"Added Slack notification node"
"Fixed HTTP timeout issue"
"Performance optimization for large datasets"

# Avoid
"changes"
"updates"
"fixes"
```

### 3. Tagging Strategy
```bash
# Environment tags
["production", "release"]
["staging", "testing"]
["development", "experimental"]

# Feature tags
["feature", "http-integration"]
["bugfix", "timeout-fix"]
["backup", "auto"]
["restore"]
```

### 4. Version Cleanup
Periodically delete old versions to save storage:
```bash
# Keep only:
- Current version
- Last 5 versions
- All production releases
- All versions less than 30 days old
```

### 5. Restore Safety
Always use `createBackup: true` when restoring:
```json
{
  "createBackup": true,
  "description": "Descriptive reason for restoration"
}
```

## Limitations

1. **Storage Size**: Complete workflow snapshots are stored
   - Consider cleanup strategy for long-running workflows
   - Disk space grows with number of versions

2. **Current Version Protection**: Cannot delete current version
   - Must restore or create new version first

3. **Author Tracking**: Currently uses "system" placeholder
   - Will use authenticated user when auth context is integrated

4. **Compare Endpoint**: Currently has known issues
   - Use version changes in create/list responses instead

## Future Enhancements

Planned features for future releases:

1. **Diff-Based Storage**: Store only changes instead of full snapshots
2. **Version Branching**: Support for parallel version branches
3. **Automatic Versioning**: Auto-create versions on workflow updates
4. **Version Approval**: Workflow for approving versions before deployment
5. **Retention Policies**: Automatic cleanup based on rules
6. **Version Export/Import**: Share versions between instances
7. **Visual Comparison**: UI for comparing workflow versions
8. **Rollback Limits**: Configurable rollback restrictions
9. **Audit Trail**: Enhanced logging of version operations
10. **Permission System**: Role-based access control for versions

## Troubleshooting

### Issue: "Version not found" Error
**Cause**: Version ID is incorrect or version was deleted
**Solution**: List versions to get correct ID

### Issue: "Cannot delete the current version"
**Cause**: Trying to delete the current active version
**Solution**: Restore or create a new version first

### Issue: Versions List is Empty
**Cause**: No versions created yet
**Solution**: Create a version using POST endpoint

### Issue: Restore Doesn't Create Backup
**Cause**: `createBackup` is set to false
**Solution**: Set `createBackup: true` in request body

### Issue: Changes Not Detected
**Cause**: Workflow hasn't actually changed
**Solution**: Make actual changes to workflow before creating version

## Monitoring

### Log Messages
The system logs important operations:
```
✅ Version created: ver_123 (workflow=wf_456, version=1, author=system)
✅ Workflow restored: workflow=wf_456, restored-from=v1, author=system
⚠️  Failed to create backup: ...
⚠️  Failed to create restore version: ...
```

### Metrics to Track
- Number of versions per workflow
- Version creation rate
- Restore frequency
- Storage usage for versions
- Average version size

## Additional Resources

- [Workflow API Documentation](./WORKFLOW_API.md)
- [Authentication Documentation](./AUTHENTICATION.md)
- [Storage Backends](./STORAGE.md)
- [Webhook System](./WEBHOOKS.md)

## Support

For issues or questions:
- GitHub Issues: https://github.com/dipankar/n8n-go/issues
- Documentation: https://github.com/dipankar/n8n-go/docs

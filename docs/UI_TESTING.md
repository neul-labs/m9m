# n8n Frontend Compatibility Testing Guide

## Overview

This guide provides comprehensive procedures for testing n8n-go backend compatibility with the n8n frontend UI. The goal is to ensure that workflows created and managed through the n8n UI work seamlessly with the n8n-go backend.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Setup Instructions](#setup-instructions)
3. [Testing Procedures](#testing-procedures)
4. [Expected Behaviors](#expected-behaviors)
5. [Known Differences](#known-differences)
6. [Troubleshooting](#troubleshooting)
7. [Reporting Issues](#reporting-issues)

---

## Prerequisites

### Required Software

- **n8n-go backend**: Built and ready to run
- **n8n frontend**: Latest stable version or specific version for testing
- **Node.js**: v18+ (for running n8n frontend)
- **Docker**: Optional, for running n8n UI in container
- **curl or Postman**: For API testing
- **Web browser**: Chrome, Firefox, or Safari

### Test Environment

```bash
# Backend configuration
N8N_GO_PORT=8080
N8N_GO_HOST=0.0.0.0
N8N_GO_LOG_LEVEL=debug
N8N_GO_AUTH_ENABLED=true
N8N_GO_JWT_SECRET=test-secret-key-for-development

# Frontend configuration
VUE_APP_URL_BASE_API=http://localhost:8080
N8N_HOST=localhost
N8N_PORT=5678
N8N_PROTOCOL=http
```

---

## Setup Instructions

### Method 1: Run n8n Frontend from Source

1. **Clone n8n repository**:
```bash
git clone https://github.com/n8n-io/n8n.git
cd n8n
git checkout [version-tag]  # e.g., n8n@1.0.0
```

2. **Install dependencies**:
```bash
npm install -g pnpm
pnpm install
```

3. **Configure backend URL**:
```bash
# Edit packages/editor-ui/.env or create .env.local
echo "VUE_APP_URL_BASE_API=http://localhost:8080" > packages/editor-ui/.env.local
```

4. **Build and run frontend only**:
```bash
cd packages/editor-ui
pnpm run dev
```

5. **Start n8n-go backend**:
```bash
cd /path/to/n8n-go
./n8n-go --port 8080 --auth-enabled=true --jwt-secret=test-secret
```

6. **Access UI**:
```
http://localhost:5678
```

### Method 2: Use Docker with Custom Backend

1. **Run n8n container with custom backend URL**:
```bash
docker run -it --rm \
  --name n8n \
  -p 5678:5678 \
  -e N8N_HOST=localhost \
  -e N8N_PORT=5678 \
  -e N8N_PROTOCOL=http \
  -e WEBHOOK_URL=http://localhost:8080 \
  -e VUE_APP_URL_BASE_API=http://localhost:8080 \
  n8nio/n8n
```

2. **Start n8n-go backend**:
```bash
./n8n-go --port 8080 --auth-enabled=true
```

### Method 3: Proxy Configuration

Use nginx or another proxy to route API calls to n8n-go while serving n8n UI:

```nginx
# nginx.conf
server {
    listen 5678;

    # Serve n8n UI
    location / {
        proxy_pass http://localhost:5679;  # n8n UI
    }

    # Route API to n8n-go
    location /api/ {
        proxy_pass http://localhost:8080;
    }

    # Route webhooks to n8n-go
    location /webhook/ {
        proxy_pass http://localhost:8080;
    }
}
```

---

## Testing Procedures

### 1. Authentication Testing

#### Test 1.1: Initial Login
**Objective**: Verify JWT authentication works with UI

**Steps**:
1. Navigate to n8n UI login page
2. Enter credentials:
   - Email: `admin@example.com`
   - Password: `admin123`
3. Click "Sign in"

**Expected Result**:
- User is redirected to workflow dashboard
- JWT token is stored in browser
- User info is displayed in UI

**API Calls to Monitor**:
```
POST /api/v1/auth/login
GET /api/v1/auth/me
```

#### Test 1.2: Session Persistence
**Objective**: Verify session persists across page reloads

**Steps**:
1. Login successfully
2. Refresh browser
3. Check if still logged in

**Expected Result**:
- User remains authenticated
- No login prompt shown

#### Test 1.3: Logout
**Objective**: Verify logout functionality

**Steps**:
1. Click user menu
2. Select "Log out"

**Expected Result**:
- User redirected to login page
- Token cleared from browser
- Subsequent API calls return 401

---

### 2. Workflow CRUD Operations

#### Test 2.1: Create New Workflow
**Objective**: Create workflow from UI

**Steps**:
1. Click "New Workflow"
2. Add workflow name: "Test Workflow 1"
3. Click "Save"

**Expected Result**:
- Workflow appears in workflow list
- Workflow ID generated
- Success notification shown

**API Calls**:
```
POST /api/v1/workflows
{
  "name": "Test Workflow 1",
  "nodes": [],
  "connections": {},
  "active": false
}
```

**Validation**:
```bash
curl http://localhost:8080/api/v1/workflows
```

#### Test 2.2: List Workflows
**Objective**: View all workflows in list

**Steps**:
1. Navigate to workflows page
2. Verify all workflows appear

**Expected Result**:
- All workflows listed
- Pagination works (if > 20 workflows)
- Active/inactive status shown

**API Calls**:
```
GET /api/v1/workflows
```

#### Test 2.3: Open Existing Workflow
**Objective**: Load workflow for editing

**Steps**:
1. Click on existing workflow from list
2. Verify workflow loads in editor

**Expected Result**:
- All nodes displayed correctly
- Connections shown properly
- Node parameters populated

**API Calls**:
```
GET /api/v1/workflows/:id
```

#### Test 2.4: Update Workflow
**Objective**: Modify and save workflow

**Steps**:
1. Open workflow
2. Change workflow name
3. Add a node (e.g., HTTP Request)
4. Click "Save"

**Expected Result**:
- Changes saved successfully
- Updated workflow persists
- Version number increments (if versioning enabled)

**API Calls**:
```
PATCH /api/v1/workflows/:id
```

#### Test 2.5: Delete Workflow
**Objective**: Remove workflow

**Steps**:
1. Right-click workflow in list
2. Select "Delete"
3. Confirm deletion

**Expected Result**:
- Workflow removed from list
- Data deleted from backend

**API Calls**:
```
DELETE /api/v1/workflows/:id
```

---

### 3. Node Configuration Testing

#### Test 3.1: Add HTTP Request Node
**Objective**: Configure basic HTTP node

**Steps**:
1. Open workflow
2. Click "Add Node"
3. Search for "HTTP Request"
4. Add node to canvas
5. Configure parameters:
   - URL: `https://jsonplaceholder.typicode.com/posts/1`
   - Method: GET
6. Click "Execute Node"

**Expected Result**:
- Node executes successfully
- Response data displayed
- Node status shows success

#### Test 3.2: Add Webhook Node
**Objective**: Configure webhook trigger

**Steps**:
1. Add "Webhook" node
2. Set HTTP Method: POST
3. Set Path: `test-webhook`
4. Note webhook URL
5. Save workflow
6. Activate workflow
7. Send test request to webhook URL

**Expected Result**:
- Webhook registered
- URL displayed in node
- Test request triggers workflow

**API Calls**:
```
POST /api/v1/webhooks/test/test-webhook
GET /api/v1/webhooks
```

#### Test 3.3: Configure Node Credentials
**Objective**: Add and use credentials

**Steps**:
1. Add node requiring credentials (e.g., Slack, OpenAI)
2. Click "Create New Credential"
3. Enter credential details
4. Save credential
5. Select credential in node

**Expected Result**:
- Credential saved securely
- Node can use credential
- Credential appears in credentials list

**API Calls**:
```
POST /api/v1/credentials
GET /api/v1/credentials
```

---

### 4. Workflow Execution Testing

#### Test 4.1: Manual Execution
**Objective**: Execute workflow manually

**Steps**:
1. Open workflow with nodes
2. Click "Execute Workflow" button
3. Monitor execution progress

**Expected Result**:
- Workflow executes
- Node results displayed
- Execution history recorded

**API Calls**:
```
POST /api/v1/workflows/:id/execute
GET /api/v1/executions
```

#### Test 4.2: Webhook Trigger Execution
**Objective**: Execute via webhook

**Steps**:
1. Create workflow with webhook trigger
2. Activate workflow
3. Send POST request to webhook URL
4. Check execution results in UI

**Expected Result**:
- Workflow triggered by webhook
- Execution appears in history
- Webhook data available in nodes

#### Test 4.3: View Execution History
**Objective**: Review past executions

**Steps**:
1. Navigate to "Executions" tab
2. View execution list
3. Click on specific execution
4. Review execution details

**Expected Result**:
- All executions listed
- Execution details show node data
- Filter/search works

**API Calls**:
```
GET /api/v1/executions
GET /api/v1/executions/:id
```

---

### 5. Variables and Environments Testing

#### Test 5.1: Create Global Variable
**Objective**: Add global variable via UI

**Steps**:
1. Navigate to Variables section
2. Click "Add Variable"
3. Enter:
   - Key: `API_BASE_URL`
   - Value: `https://api.example.com`
   - Type: Global
4. Save

**Expected Result**:
- Variable appears in list
- Variable usable in workflows

**API Calls**:
```
POST /api/v1/variables
GET /api/v1/variables
```

#### Test 5.2: Use Variable in Workflow
**Objective**: Reference variable in node

**Steps**:
1. Open workflow
2. Add HTTP Request node
3. Set URL to: `{{ $vars.API_BASE_URL }}/endpoint`
4. Execute node

**Expected Result**:
- Variable resolved correctly
- Request sent to correct URL

#### Test 5.3: Create Environment
**Objective**: Set up environment

**Steps**:
1. Navigate to Environments
2. Click "Add Environment"
3. Enter:
   - Name: `production`
   - Key: `prod`
4. Add environment variables
5. Save

**Expected Result**:
- Environment created
- Variables scoped to environment

**API Calls**:
```
POST /api/v1/environments
GET /api/v1/environments
```

---

### 6. Workflow Versions Testing

#### Test 6.1: Create Version
**Objective**: Save workflow version

**Steps**:
1. Open workflow
2. Make changes
3. Click "Create Version" or save
4. Enter version description
5. Confirm

**Expected Result**:
- Version saved
- Version appears in history

**API Calls**:
```
POST /api/v1/workflows/:id/versions
GET /api/v1/workflows/:id/versions
```

#### Test 6.2: Compare Versions
**Objective**: View version differences

**Steps**:
1. Open version history
2. Select two versions
3. Click "Compare"

**Expected Result**:
- Differences highlighted
- Changes clearly shown

**API Calls**:
```
GET /api/v1/workflows/:id/versions/compare?from=1&to=2
```

#### Test 6.3: Restore Version
**Objective**: Rollback to previous version

**Steps**:
1. Open version history
2. Select older version
3. Click "Restore"
4. Confirm restoration

**Expected Result**:
- Workflow reverted
- Backup created
- Current version updated

**API Calls**:
```
POST /api/v1/workflows/:id/versions/:versionId/restore
```

---

### 7. User Management Testing

#### Test 7.1: Create User
**Objective**: Add new user via UI

**Steps**:
1. Navigate to Users section (admin only)
2. Click "Add User"
3. Enter details:
   - Email: `test@example.com`
   - Password: `Test123!`
   - Role: Member
4. Save

**Expected Result**:
- User created
- User can login
- Permissions applied correctly

**API Calls**:
```
POST /api/v1/auth/users
GET /api/v1/auth/users
```

#### Test 7.2: Update User Role
**Objective**: Change user permissions

**Steps**:
1. Select user
2. Change role to "Viewer"
3. Save

**Expected Result**:
- Role updated
- Permissions reflect new role

#### Test 7.3: Delete User
**Objective**: Remove user

**Steps**:
1. Select user
2. Click "Delete"
3. Confirm

**Expected Result**:
- User removed
- Login no longer works

---

## Expected Behaviors

### API Response Formats

#### Workflow Response
```json
{
  "id": "wf_abc123",
  "name": "My Workflow",
  "nodes": [...],
  "connections": {...},
  "active": false,
  "createdAt": "2025-01-10T00:00:00Z",
  "updatedAt": "2025-01-10T00:00:00Z"
}
```

#### Execution Response
```json
{
  "id": "exec_xyz789",
  "workflowId": "wf_abc123",
  "status": "success",
  "startedAt": "2025-01-10T00:00:00Z",
  "finishedAt": "2025-01-10T00:00:01Z",
  "data": {...}
}
```

### HTTP Status Codes

| Code | Meaning | When Used |
|------|---------|-----------|
| 200 | OK | Successful GET, PUT, PATCH |
| 201 | Created | Successful POST |
| 204 | No Content | Successful DELETE |
| 400 | Bad Request | Invalid input |
| 401 | Unauthorized | Missing/invalid auth |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 500 | Server Error | Backend error |

---

## Known Differences

### Feature Differences

| Feature | n8n | n8n-go | Notes |
|---------|-----|--------|-------|
| Node Count | 400+ | 15 | Core nodes implemented |
| Queue Systems | Bull | Memory/Redis/RabbitMQ | Different implementation |
| Database | PostgreSQL | PostgreSQL/SQLite/BadgerDB | Multiple backends |
| Execution Mode | Process-based | Goroutine-based | Different concurrency |
| Memory Usage | ~512MB | ~150MB | 70% reduction |
| Startup Time | ~3s | ~500ms | 6x faster |

### API Endpoint Differences

Some endpoints may have slightly different parameter names or response formats:

1. **Pagination**: n8n-go uses `limit`/`offset`, n8n may use `take`/`skip`
2. **Timestamps**: n8n-go uses RFC3339, n8n may use different format
3. **Error format**: May differ slightly in structure

---

## Troubleshooting

### Issue 1: CORS Errors

**Symptom**: Browser console shows CORS errors

**Solution**:
```bash
# Enable CORS in n8n-go
./n8n-go --cors-enabled=true --cors-origins="http://localhost:5678"
```

### Issue 2: Authentication Fails

**Symptom**: Login returns 401 or 403

**Possible Causes**:
1. JWT secret mismatch
2. User doesn't exist
3. Password incorrect

**Solutions**:
```bash
# Verify JWT secret is set
./n8n-go --jwt-secret=your-secret-key

# Create initial admin user
curl -X POST http://localhost:8080/api/v1/auth/users \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "admin123",
    "role": "admin"
  }'
```

### Issue 3: Workflow Won't Save

**Symptom**: Save button doesn't work or returns error

**Possible Causes**:
1. Invalid workflow format
2. Missing required fields
3. Network error

**Solutions**:
1. Check browser console for errors
2. Verify backend logs: `tail -f /var/log/n8n-go.log`
3. Test API directly with curl

### Issue 4: Nodes Don't Execute

**Symptom**: Node execution fails or hangs

**Possible Causes**:
1. Node not implemented
2. Missing credentials
3. Parameter validation error

**Solutions**:
1. Check which nodes are implemented: `GET /api/v1/node-types`
2. Verify credentials configured correctly
3. Check backend logs for errors

### Issue 5: Webhooks Not Working

**Symptom**: Webhook doesn't trigger workflow

**Possible Causes**:
1. Workflow not activated
2. Webhook path incorrect
3. Wrong HTTP method

**Solutions**:
```bash
# List registered webhooks
curl http://localhost:8080/api/v1/webhooks

# Test webhook directly
curl -X POST http://localhost:8080/webhook/test-path \
  -H "Content-Type: application/json" \
  -d '{"test": "data"}'
```

---

## Reporting Issues

### Information to Include

When reporting compatibility issues, include:

1. **n8n version**: e.g., `1.0.0`
2. **n8n-go version**: e.g., `0.4.0`
3. **Browser**: Chrome 120, Firefox 121, etc.
4. **Workflow JSON**: Export problematic workflow
5. **API requests**: Copy from Network tab
6. **Backend logs**: Relevant log entries
7. **Steps to reproduce**: Detailed steps
8. **Expected vs actual behavior**

### Issue Template

```markdown
### Environment
- n8n version:
- n8n-go version:
- Browser:
- OS:

### Description
Brief description of the issue

### Steps to Reproduce
1. Step one
2. Step two
3. ...

### Expected Behavior
What should happen

### Actual Behavior
What actually happens

### Logs
```
Paste relevant logs here
```

### Additional Context
Any other relevant information
```

---

## Testing Checklist

Use this checklist to track testing progress:

### Authentication
- [ ] Login with valid credentials
- [ ] Login with invalid credentials
- [ ] Session persistence
- [ ] Logout
- [ ] Token refresh

### Workflows
- [ ] Create workflow
- [ ] List workflows
- [ ] Open workflow
- [ ] Update workflow
- [ ] Delete workflow
- [ ] Activate/deactivate workflow

### Nodes
- [ ] Add node to canvas
- [ ] Configure node parameters
- [ ] Add credentials
- [ ] Execute node
- [ ] View node output

### Execution
- [ ] Manual execution
- [ ] Webhook trigger execution
- [ ] View execution history
- [ ] View execution details
- [ ] Error handling

### Variables
- [ ] Create global variable
- [ ] Create environment variable
- [ ] Create workflow variable
- [ ] Use variable in node
- [ ] Update variable
- [ ] Delete variable

### Versions
- [ ] Create version
- [ ] List versions
- [ ] Compare versions
- [ ] Restore version
- [ ] Delete version

### Users
- [ ] Create user
- [ ] Update user
- [ ] Delete user
- [ ] Role-based access control

---

## Next Steps

After completing these tests:

1. **Document Results**: Create FRONTEND_COMPATIBILITY.md with findings
2. **Fix Critical Issues**: Address any blocking compatibility problems
3. **Optimize API**: Improve endpoint compatibility where needed
4. **Update Documentation**: Add any discovered quirks or workarounds
5. **Performance Testing**: Measure UI responsiveness with n8n-go backend

---

**Last Updated**: November 10, 2025
**Version**: 1.0
**Status**: Ready for testing

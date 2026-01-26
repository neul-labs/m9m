# Version Control Nodes

Version control nodes interact with Git hosting platforms.

## GitHub Node

Interact with GitHub repositories, issues, and pull requests.

### Type

```
n8n-nodes-base.github
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `accessToken` | string | Yes | GitHub personal access token |
| `resource` | string | Yes | Resource type |
| `operation` | string | Yes | Operation to perform |
| `owner` | string | Depends | Repository owner |
| `repository` | string | Depends | Repository name |

### Resources & Operations

#### Repository

| Operation | Description |
|-----------|-------------|
| `get` | Get repository details |
| `list` | List repositories |

#### Issue

| Operation | Description |
|-----------|-------------|
| `get` | Get issue details |
| `list` | List issues |

#### Pull Request

| Operation | Description |
|-----------|-------------|
| `get` | Get PR details |
| `list` | List pull requests |

#### User

| Operation | Description |
|-----------|-------------|
| `getAuthenticated` | Get authenticated user |
| `get` | Get user by username |

### Examples

#### Get Repository

```json
{
  "type": "n8n-nodes-base.github",
  "parameters": {
    "accessToken": "={{ $credentials.github.accessToken }}",
    "resource": "repository",
    "operation": "get",
    "owner": "neul-labs",
    "repository": "m9m"
  }
}
```

#### List Issues

```json
{
  "type": "n8n-nodes-base.github",
  "parameters": {
    "accessToken": "={{ $credentials.github.accessToken }}",
    "resource": "issue",
    "operation": "list",
    "owner": "neul-labs",
    "repository": "m9m"
  }
}
```

#### Get Pull Request

```json
{
  "type": "n8n-nodes-base.github",
  "parameters": {
    "accessToken": "={{ $credentials.github.accessToken }}",
    "resource": "pullRequest",
    "operation": "get",
    "owner": "neul-labs",
    "repository": "m9m",
    "pullNumber": "={{ $json.prNumber }}"
  }
}
```

#### List User Repositories

```json
{
  "type": "n8n-nodes-base.github",
  "parameters": {
    "accessToken": "={{ $credentials.github.accessToken }}",
    "resource": "repository",
    "operation": "list",
    "username": "octocat"
  }
}
```

### Output Example

Repository response:

```json
{
  "json": {
    "id": 123456,
    "name": "m9m",
    "full_name": "neul-labs/m9m",
    "description": "High-performance workflow automation",
    "html_url": "https://github.com/neul-labs/m9m",
    "stargazers_count": 100,
    "forks_count": 25
  }
}
```

---

## GitLab Node

Interact with GitLab projects, issues, merge requests, and pipelines.

### Type

```
n8n-nodes-base.gitlab
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `accessToken` | string | Yes | - | GitLab access token |
| `baseUrl` | string | No | `https://gitlab.com` | GitLab instance URL |
| `resource` | string | Yes | - | Resource type |
| `operation` | string | Yes | - | Operation to perform |
| `projectId` | string | Depends | - | Project ID or path |

### Resources & Operations

#### Project

| Operation | Description |
|-----------|-------------|
| `get` | Get project details |
| `list` | List projects |

#### Issue

| Operation | Description |
|-----------|-------------|
| `create` | Create issue |
| `get` | Get issue |
| `getAll` | List issues |
| `update` | Update issue |
| `delete` | Delete issue |

#### Merge Request

| Operation | Description |
|-----------|-------------|
| `create` | Create MR |
| `get` | Get MR details |
| `getAll` | List MRs |
| `merge` | Merge MR |

#### Pipeline

| Operation | Description |
|-----------|-------------|
| `trigger` | Trigger pipeline |
| `get` | Get pipeline |
| `getAll` | List pipelines |
| `cancel` | Cancel pipeline |
| `retry` | Retry pipeline |

#### Branch

| Operation | Description |
|-----------|-------------|
| `list` | List branches |
| `get` | Get branch |

### Examples

#### Get Project

```json
{
  "type": "n8n-nodes-base.gitlab",
  "parameters": {
    "accessToken": "={{ $credentials.gitlab.accessToken }}",
    "resource": "project",
    "operation": "get",
    "projectId": "mygroup/myproject"
  }
}
```

#### Self-Hosted GitLab

```json
{
  "type": "n8n-nodes-base.gitlab",
  "parameters": {
    "accessToken": "={{ $credentials.gitlab.accessToken }}",
    "baseUrl": "https://gitlab.company.com",
    "resource": "project",
    "operation": "list"
  }
}
```

#### Create Issue

```json
{
  "type": "n8n-nodes-base.gitlab",
  "parameters": {
    "accessToken": "={{ $credentials.gitlab.accessToken }}",
    "resource": "issue",
    "operation": "create",
    "projectId": "12345",
    "title": "{{ $json.issueTitle }}",
    "description": "{{ $json.issueBody }}"
  }
}
```

#### List Merge Requests

```json
{
  "type": "n8n-nodes-base.gitlab",
  "parameters": {
    "accessToken": "={{ $credentials.gitlab.accessToken }}",
    "resource": "mergeRequest",
    "operation": "getAll",
    "projectId": "mygroup/myproject",
    "state": "opened"
  }
}
```

#### Trigger Pipeline

```json
{
  "type": "n8n-nodes-base.gitlab",
  "parameters": {
    "accessToken": "={{ $credentials.gitlab.accessToken }}",
    "resource": "pipeline",
    "operation": "trigger",
    "projectId": "12345",
    "ref": "main",
    "variables": {
      "DEPLOY_ENV": "production"
    }
  }
}
```

#### Get Pipeline Status

```json
{
  "type": "n8n-nodes-base.gitlab",
  "parameters": {
    "accessToken": "={{ $credentials.gitlab.accessToken }}",
    "resource": "pipeline",
    "operation": "get",
    "projectId": "12345",
    "pipelineId": "{{ $json.pipelineId }}"
  }
}
```

### Output Example

Merge request response:

```json
{
  "json": {
    "id": 1,
    "iid": 42,
    "title": "Feature: Add new endpoint",
    "state": "opened",
    "web_url": "https://gitlab.com/group/project/-/merge_requests/42",
    "author": {
      "username": "developer"
    }
  }
}
```

---

## Quick Reference

| Node | Type | Platform |
|------|------|----------|
| GitHub | `n8n-nodes-base.github` | GitHub.com |
| GitLab | `n8n-nodes-base.gitlab` | GitLab.com or self-hosted |

### Token Permissions

| Platform | Required Scopes |
|----------|-----------------|
| GitHub | `repo`, `read:user` |
| GitLab | `api` or `read_api` |

### Common Use Cases

| Use Case | Node | Resource | Operation |
|----------|------|----------|-----------|
| Monitor PRs | GitHub | pullRequest | list |
| Auto-create issues | GitLab | issue | create |
| Trigger deploys | GitLab | pipeline | trigger |
| Get repo stats | GitHub | repository | get |

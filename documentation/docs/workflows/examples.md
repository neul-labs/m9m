# Workflow Examples

Practical workflow examples for common use cases.

## HTTP API Integration

Fetch data from an API and process it:

```json
{
  "name": "API Data Fetch",
  "nodes": [
    {
      "id": "start",
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    },
    {
      "id": "fetch",
      "name": "Fetch Users",
      "type": "n8n-nodes-base.httpRequest",
      "position": [450, 300],
      "parameters": {
        "url": "https://jsonplaceholder.typicode.com/users",
        "method": "GET"
      }
    },
    {
      "id": "filter",
      "name": "Filter Active",
      "type": "n8n-nodes-base.filter",
      "position": [650, 300],
      "parameters": {
        "conditions": [
          {
            "leftValue": "={{ $json.company.name }}",
            "operator": "exists"
          }
        ]
      }
    }
  ],
  "connections": {
    "Start": {"main": [[{"node": "Fetch Users", "type": "main", "index": 0}]]},
    "Fetch Users": {"main": [[{"node": "Filter Active", "type": "main", "index": 0}]]}
  }
}
```

## Webhook to Slack

Receive webhooks and notify Slack:

```json
{
  "name": "Webhook to Slack",
  "nodes": [
    {
      "id": "webhook",
      "name": "Webhook",
      "type": "n8n-nodes-base.webhook",
      "position": [250, 300],
      "parameters": {
        "path": "/alert",
        "httpMethod": "POST"
      }
    },
    {
      "id": "format",
      "name": "Format Message",
      "type": "n8n-nodes-base.set",
      "position": [450, 300],
      "parameters": {
        "assignments": [
          {
            "name": "slackMessage",
            "value": "Alert: {{ $json.body.message }}\nSeverity: {{ $json.body.severity }}"
          }
        ]
      }
    },
    {
      "id": "slack",
      "name": "Send to Slack",
      "type": "n8n-nodes-base.slack",
      "position": [650, 300],
      "parameters": {
        "webhookUrl": "https://hooks.slack.com/services/...",
        "text": "={{ $json.slackMessage }}"
      }
    }
  ],
  "connections": {
    "Webhook": {"main": [[{"node": "Format Message", "type": "main", "index": 0}]]},
    "Format Message": {"main": [[{"node": "Send to Slack", "type": "main", "index": 0}]]}
  }
}
```

## Scheduled Database Backup

Daily database query and notification:

```json
{
  "name": "Daily Stats Report",
  "nodes": [
    {
      "id": "cron",
      "name": "Daily at 9 AM",
      "type": "n8n-nodes-base.cron",
      "position": [250, 300],
      "parameters": {
        "cronExpression": "0 9 * * *"
      }
    },
    {
      "id": "query",
      "name": "Query Stats",
      "type": "n8n-nodes-base.postgres",
      "position": [450, 300],
      "parameters": {
        "connectionUrl": "postgres://user:pass@localhost/db",
        "operation": "executeQuery",
        "query": "SELECT COUNT(*) as users, DATE(created_at) as date FROM users WHERE created_at > NOW() - INTERVAL '1 day' GROUP BY date"
      }
    },
    {
      "id": "format",
      "name": "Format Report",
      "type": "n8n-nodes-base.set",
      "position": [650, 300],
      "parameters": {
        "assignments": [
          {
            "name": "report",
            "value": "Daily Stats:\n- New users: {{ $json.users }}\n- Date: {{ $json.date }}"
          }
        ]
      }
    },
    {
      "id": "email",
      "name": "Send Email",
      "type": "n8n-nodes-base.emailSend",
      "position": [850, 300],
      "parameters": {
        "smtpHost": "smtp.gmail.com",
        "smtpPort": 587,
        "fromEmail": "reports@company.com",
        "toEmail": "team@company.com",
        "subject": "Daily Stats Report",
        "body": "={{ $json.report }}"
      }
    }
  ],
  "connections": {
    "Daily at 9 AM": {"main": [[{"node": "Query Stats", "type": "main", "index": 0}]]},
    "Query Stats": {"main": [[{"node": "Format Report", "type": "main", "index": 0}]]},
    "Format Report": {"main": [[{"node": "Send Email", "type": "main", "index": 0}]]}
  }
}
```

## AI Content Generation

Use OpenAI to generate content:

```json
{
  "name": "AI Content Generator",
  "nodes": [
    {
      "id": "webhook",
      "name": "Webhook",
      "type": "n8n-nodes-base.webhook",
      "position": [250, 300],
      "parameters": {
        "path": "/generate",
        "httpMethod": "POST"
      }
    },
    {
      "id": "openai",
      "name": "Generate Content",
      "type": "n8n-nodes-base.openAi",
      "position": [450, 300],
      "parameters": {
        "apiKey": "={{ $credentials.openai.apiKey }}",
        "model": "gpt-4",
        "prompt": "Write a blog post about: {{ $json.body.topic }}",
        "maxTokens": 1000
      }
    },
    {
      "id": "response",
      "name": "Format Response",
      "type": "n8n-nodes-base.set",
      "position": [650, 300],
      "parameters": {
        "assignments": [
          {"name": "content", "value": "={{ $json.response }}"},
          {"name": "topic", "value": "={{ $json.body.topic }}"}
        ]
      }
    }
  ],
  "connections": {
    "Webhook": {"main": [[{"node": "Generate Content", "type": "main", "index": 0}]]},
    "Generate Content": {"main": [[{"node": "Format Response", "type": "main", "index": 0}]]}
  }
}
```

## GitHub Issue Monitor

Watch for new issues and notify:

```json
{
  "name": "GitHub Issue Monitor",
  "nodes": [
    {
      "id": "cron",
      "name": "Every 5 Minutes",
      "type": "n8n-nodes-base.cron",
      "position": [250, 300],
      "parameters": {
        "cronExpression": "*/5 * * * *"
      }
    },
    {
      "id": "github",
      "name": "Get Issues",
      "type": "n8n-nodes-base.github",
      "position": [450, 300],
      "parameters": {
        "accessToken": "={{ $credentials.github.accessToken }}",
        "resource": "issue",
        "operation": "list",
        "owner": "my-org",
        "repository": "my-repo"
      }
    },
    {
      "id": "filter",
      "name": "New Issues",
      "type": "n8n-nodes-base.filter",
      "position": [650, 300],
      "parameters": {
        "conditions": [
          {
            "leftValue": "={{ new Date($json.created_at) > new Date(Date.now() - 300000) }}",
            "operator": "equals",
            "rightValue": true
          }
        ]
      }
    },
    {
      "id": "slack",
      "name": "Notify",
      "type": "n8n-nodes-base.slack",
      "position": [850, 300],
      "parameters": {
        "webhookUrl": "https://hooks.slack.com/...",
        "text": "New issue: {{ $json.title }}\n{{ $json.html_url }}"
      }
    }
  ],
  "connections": {
    "Every 5 Minutes": {"main": [[{"node": "Get Issues", "type": "main", "index": 0}]]},
    "Get Issues": {"main": [[{"node": "New Issues", "type": "main", "index": 0}]]},
    "New Issues": {"main": [[{"node": "Notify", "type": "main", "index": 0}]]}
  }
}
```

## Data Transformation Pipeline

Complex data processing:

```json
{
  "name": "Data Pipeline",
  "nodes": [
    {
      "id": "start",
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "position": [250, 300],
      "parameters": {}
    },
    {
      "id": "fetch",
      "name": "Fetch Data",
      "type": "n8n-nodes-base.httpRequest",
      "position": [450, 300],
      "parameters": {
        "url": "https://api.example.com/data",
        "method": "GET"
      }
    },
    {
      "id": "code",
      "name": "Transform",
      "type": "n8n-nodes-base.code",
      "position": [650, 300],
      "parameters": {
        "language": "javascript",
        "code": "return items.map(item => ({\n  json: {\n    id: item.json.id,\n    name: item.json.name.toUpperCase(),\n    processed: true,\n    timestamp: new Date().toISOString()\n  }\n}));"
      }
    },
    {
      "id": "save",
      "name": "Save to DB",
      "type": "n8n-nodes-base.postgres",
      "position": [850, 300],
      "parameters": {
        "connectionUrl": "postgres://...",
        "operation": "executeQuery",
        "query": "INSERT INTO processed (id, name, timestamp) VALUES ('{{ $json.id }}', '{{ $json.name }}', '{{ $json.timestamp }}')"
      }
    }
  ],
  "connections": {
    "Start": {"main": [[{"node": "Fetch Data", "type": "main", "index": 0}]]},
    "Fetch Data": {"main": [[{"node": "Transform", "type": "main", "index": 0}]]},
    "Transform": {"main": [[{"node": "Save to DB", "type": "main", "index": 0}]]}
  }
}
```

## Running Examples

```bash
# Save example to file
cat > example.json << 'EOF'
{...workflow json...}
EOF

# Run the workflow
m9m run example.json
```

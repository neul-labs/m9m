# Your First Workflow

Build a complete workflow that monitors a website and sends notifications.

## What We're Building

A workflow that:

1. Runs on a schedule (every 5 minutes)
2. Checks if a website is responding
3. Sends a Slack notification if the site is down

## Prerequisites

- m9m installed and running
- Slack webhook URL (for notifications)

## Step 1: Create the Workflow Structure

Create a file `site-monitor.json`:

```json
{
  "name": "Site Monitor",
  "active": true,
  "nodes": [],
  "connections": {}
}
```

## Step 2: Add the Trigger Node

Add a cron trigger that runs every 5 minutes:

```json
{
  "id": "cron",
  "type": "n8n-nodes-base.cron",
  "position": [250, 300],
  "parameters": {
    "triggerTimes": {
      "item": [
        {
          "mode": "everyMinute",
          "minute": 5
        }
      ]
    }
  }
}
```

## Step 3: Add the HTTP Request Node

Check the target website:

```json
{
  "id": "check-site",
  "type": "n8n-nodes-base.httpRequest",
  "position": [450, 300],
  "parameters": {
    "url": "https://example.com",
    "method": "GET",
    "timeout": 10000,
    "options": {
      "response": {
        "response": {
          "neverError": true
        }
      }
    }
  }
}
```

!!! note "neverError Option"
    Setting `neverError: true` prevents the workflow from stopping on HTTP errors. We want to capture the error and handle it ourselves.

## Step 4: Add the Filter Node

Filter for failed responses:

```json
{
  "id": "filter",
  "type": "n8n-nodes-base.filter",
  "position": [650, 300],
  "parameters": {
    "conditions": {
      "number": [
        {
          "value1": "={{ $json.statusCode }}",
          "operation": "notEqual",
          "value2": 200
        }
      ]
    }
  }
}
```

## Step 5: Add the Notification Node

Send a Slack alert when the site is down:

```json
{
  "id": "notify",
  "type": "n8n-nodes-base.slack",
  "position": [850, 300],
  "parameters": {
    "resource": "message",
    "operation": "post",
    "channel": "#alerts",
    "text": "Site is DOWN! Status: {{ $json.statusCode }}",
    "attachments": [
      {
        "color": "danger",
        "fields": [
          {
            "title": "URL",
            "value": "https://example.com"
          },
          {
            "title": "Status Code",
            "value": "={{ $json.statusCode }}"
          },
          {
            "title": "Time",
            "value": "={{ $now.format('YYYY-MM-DD HH:mm:ss') }}"
          }
        ]
      }
    ]
  },
  "credentials": {
    "slackApi": {
      "id": "1",
      "name": "Slack Account"
    }
  }
}
```

## Step 6: Connect the Nodes

Add connections between nodes:

```json
{
  "connections": {
    "cron": {
      "main": [[{"node": "check-site", "type": "main", "index": 0}]]
    },
    "check-site": {
      "main": [[{"node": "filter", "type": "main", "index": 0}]]
    },
    "filter": {
      "main": [[{"node": "notify", "type": "main", "index": 0}]]
    }
  }
}
```

## Complete Workflow

Here's the complete workflow:

```json
{
  "name": "Site Monitor",
  "active": true,
  "nodes": [
    {
      "id": "cron",
      "type": "n8n-nodes-base.cron",
      "position": [250, 300],
      "parameters": {
        "triggerTimes": {
          "item": [
            {
              "mode": "everyMinute",
              "minute": 5
            }
          ]
        }
      }
    },
    {
      "id": "check-site",
      "type": "n8n-nodes-base.httpRequest",
      "position": [450, 300],
      "parameters": {
        "url": "https://example.com",
        "method": "GET",
        "timeout": 10000,
        "options": {
          "response": {
            "response": {
              "neverError": true
            }
          }
        }
      }
    },
    {
      "id": "filter",
      "type": "n8n-nodes-base.filter",
      "position": [650, 300],
      "parameters": {
        "conditions": {
          "number": [
            {
              "value1": "={{ $json.statusCode }}",
              "operation": "notEqual",
              "value2": 200
            }
          ]
        }
      }
    },
    {
      "id": "notify",
      "type": "n8n-nodes-base.slack",
      "position": [850, 300],
      "parameters": {
        "resource": "message",
        "operation": "post",
        "channel": "#alerts",
        "text": "Site is DOWN! Status: {{ $json.statusCode }}"
      },
      "credentials": {
        "slackApi": {
          "id": "1",
          "name": "Slack Account"
        }
      }
    }
  ],
  "connections": {
    "cron": {
      "main": [[{"node": "check-site", "type": "main", "index": 0}]]
    },
    "check-site": {
      "main": [[{"node": "filter", "type": "main", "index": 0}]]
    },
    "filter": {
      "main": [[{"node": "notify", "type": "main", "index": 0}]]
    }
  }
}
```

## Set Up Credentials

Before running, configure Slack credentials:

```bash
# Via CLI
m9m credentials create slack \
  --type slackApi \
  --data '{"accessToken": "xoxb-your-token"}'
```

Or via the Web UI:

1. Go to **Settings** → **Credentials**
2. Click **Add Credential**
3. Select **Slack API**
4. Enter your access token
5. Click **Save**

## Deploy the Workflow

### Option 1: Register with Server

```bash
m9m workflow register site-monitor.json
```

This registers the workflow with the server and starts the cron trigger.

### Option 2: Import via API

```bash
curl -X POST http://localhost:8080/api/v1/workflows \
  -H "Content-Type: application/json" \
  -d @site-monitor.json
```

### Option 3: Test Manually

Test the workflow without the cron trigger:

```bash
m9m execute site-monitor.json --skip-trigger
```

## Monitor Execution

View workflow executions:

```bash
# List recent executions
m9m executions list --workflow "Site Monitor"

# View execution details
m9m executions get <execution-id>

# View logs
m9m logs --workflow "Site Monitor" --tail
```

## Enhance the Workflow

### Add Success Notification

Add a parallel branch for successful checks:

```json
{
  "id": "filter-success",
  "type": "n8n-nodes-base.filter",
  "parameters": {
    "conditions": {
      "number": [
        {
          "value1": "={{ $json.statusCode }}",
          "operation": "equal",
          "value2": 200
        }
      ]
    }
  }
}
```

### Add Response Time Tracking

Include response time in notifications:

```json
{
  "text": "Site DOWN! Status: {{ $json.statusCode }}, Response Time: {{ $json.responseTime }}ms"
}
```

### Add Multiple Sites

Monitor multiple sites with a loop:

```json
{
  "id": "sites",
  "type": "n8n-nodes-base.set",
  "parameters": {
    "values": {
      "array": [
        {
          "name": "sites",
          "value": ["https://example.com", "https://api.example.com"]
        }
      ]
    }
  }
}
```

## Next Steps

Congratulations! You've built your first m9m workflow. Continue learning:

- [Expressions Guide](../user-guide/expressions.md) - Master dynamic data
- [Error Handling](../user-guide/error-handling.md) - Handle failures gracefully
- [Node Reference](../nodes/overview.md) - Explore all available nodes
- [Production Deployment](../deployment/production.md) - Deploy to production

# MCP Examples

Real-world examples of using Claude Code with m9m for workflow automation.

## Quick Actions

### Making API Requests

```
You: "Check if our API is up at https://api.example.com/health"

Claude: I'll check the API health endpoint for you.
[Uses http_request tool]

The API is responding with status 200. Response:
{
  "status": "healthy",
  "version": "2.1.0",
  "uptime": "15d 4h 23m"
}
```

### Sending Notifications

```
You: "Send a message to #deployments on Slack saying 'v2.1.0 deployed to production'"

Claude: [Uses send_slack tool]

Message sent successfully to #deployments!
```

### AI Completions

```
You: "Use GPT-4 to summarize this error log: [paste log]"

Claude: [Uses ai_openai tool]

Summary: The error indicates a database connection timeout after 30 seconds.
The PostgreSQL server at db.example.com:5432 is not responding. This is
likely due to network issues or the database server being down.
```

---

## Workflow Creation

### API Health Monitor

```
You: "Create a workflow that checks https://api.example.com/health every 5 minutes
      and sends a Slack alert to #ops if it's not returning 200"

Claude: I'll create a health monitoring workflow with these components:

1. **Cron Trigger** - Runs every 5 minutes
2. **HTTP Request** - Checks the health endpoint
3. **IF Node** - Checks if status code is not 200
4. **Slack Node** - Sends alert to #ops

[Uses workflow_create tool]

Workflow created with ID: wf_health_monitor_abc123

[Uses workflow_activate tool]

The workflow is now active and will check your API every 5 minutes.
```

### Data Sync Pipeline

```
You: "Build a workflow that:
      1. Fetches users from our API at /api/users
      2. Filters for users created in the last 24 hours
      3. Sends a daily summary to #new-users Slack channel"

Claude: I'll create a data sync workflow:

**Nodes:**
- Start (Manual/Scheduled trigger)
- HTTP Request (GET /api/users)
- Filter (createdAt > 24 hours ago)
- Set (Format summary message)
- Slack (Post to #new-users)

[Uses workflow_create tool]

Created workflow: "Daily New Users Summary"

Would you like me to:
1. Activate it now for manual runs?
2. Add a schedule to run daily at 9 AM?
```

### Webhook Handler

```
You: "Create a workflow that receives GitHub webhook events and posts
      PR notifications to Discord"

Claude: [Uses workflow_create tool]

Created webhook workflow with:
- Webhook trigger at: https://your-m9m.com/webhook/github-prs
- Filter for pull_request events
- Discord notification with PR details

Configure this webhook URL in your GitHub repository settings.
```

---

## Debugging Workflows

### Investigating Failures

```
You: "My data-sync workflow failed last night. What happened?"

Claude: Let me check the execution logs.

[Uses execution_list to find recent failed executions]
[Uses debug_execution_logs with level="detailed"]

I found the issue. The workflow failed at the "HTTP Request" node:

**Error:** Connection timeout after 30 seconds
**Node:** Fetch Users
**Time:** 2024-01-15 03:15:42 UTC

The API at api.example.com didn't respond within the timeout period.
This could be due to:
1. API server was down during the scheduled run
2. Network connectivity issues
3. The request is taking too long (consider pagination)

Would you like me to:
1. Retry the execution now?
2. Increase the timeout setting?
3. Add retry logic to the workflow?
```

### Checking Node Outputs

```
You: "Show me what data the Filter node output in execution exec_abc123"

Claude: [Uses debug_node_output tool]

The Filter node output 15 items (filtered from 247 total):

| id | name | createdAt | status |
|----|------|-----------|--------|
| 142 | Alice | 2024-01-14 | active |
| 156 | Bob | 2024-01-14 | active |
... (13 more rows)

All 15 users were created within the last 24 hours.
```

### Performance Analysis

```
You: "How is my order-processing workflow performing?"

Claude: [Uses debug_performance tool]

**Performance Report: order-processing**

| Metric | Value |
|--------|-------|
| Total Executions | 1,247 |
| Success Rate | 98.7% |
| Avg Duration | 2.3s |
| Min Duration | 0.8s |
| Max Duration | 15.2s |

The workflow is performing well. The 1.3% failure rate (16 failures)
is mostly due to external API timeouts.

Would you like me to investigate the slowest executions?
```

---

## Custom Plugins

### Creating a Data Transformer

```
You: "Create a plugin that converts all text fields to uppercase"

Claude: [Uses plugin_create_js tool]

Created plugin: uppercase-transformer

You can now use it in workflows. It will:
- Find all string fields in input data
- Convert them to UPPERCASE
- Pass through non-string fields unchanged

Example usage in a workflow:
Input:  { "name": "John", "age": 30 }
Output: { "name": "JOHN", "age": 30 }
```

### Wrapping an External API

```
You: "We use JSONPlaceholder for testing. Create a node that fetches posts"

Claude: [Uses plugin_create_rest tool]

Created REST plugin: jsonplaceholder-posts

Configuration:
- Endpoint: https://jsonplaceholder.typicode.com/posts
- Method: GET
- No authentication required

You can now use this node in workflows to fetch mock posts data.
```

### Building a Validation Node

```
You: "Create a plugin that validates email addresses and phone numbers"

Claude: [Uses plugin_create_js tool]

Created plugin: contact-validator

The plugin adds these fields to each item:
- `isEmailValid`: boolean
- `isPhoneValid`: boolean
- `validationErrors`: array of error messages

Parameters:
- `emailField`: which field contains the email (default: "email")
- `phoneField`: which field contains the phone (default: "phone")
```

---

## Complex Workflows

### Multi-Step Data Pipeline

```
You: "Create an ETL workflow that:
      1. Fetches data from 3 different APIs
      2. Merges the results
      3. Transforms the data
      4. Loads into our database
      5. Sends a completion notification"

Claude: This is a complex workflow. Let me break it down:

**Data Sources (Parallel):**
- HTTP Request 1: Users API
- HTTP Request 2: Orders API
- HTTP Request 3: Products API

**Processing:**
- Merge: Combine all three datasets
- Code: Transform and enrich data
- Postgres: Insert into analytics table

**Notification:**
- Slack: Send completion summary

[Uses workflow_create tool]

Created workflow with 8 nodes. The three API calls run in parallel
for efficiency, then merge before processing.

Total estimated execution time: 5-10 seconds
```

### Error Handling Workflow

```
You: "Add error handling to my payment-processor workflow"

Claude: [Uses workflow_get to examine current workflow]

I see your workflow has 5 nodes but no error handling. I'll add:

1. **Try/Catch wrapper** around the payment API call
2. **Error notification** to #payments-errors Slack
3. **Retry logic** (3 attempts with exponential backoff)
4. **Dead letter queue** for failed payments

[Uses workflow_update tool]

Updated workflow with error handling. Failed payments will now:
1. Retry up to 3 times
2. Alert the team on final failure
3. Log to the dead letter queue for manual review
```

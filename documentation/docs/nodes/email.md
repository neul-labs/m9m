# Email Nodes

Email nodes send emails via SMTP servers.

## Send Email Node

Send emails through SMTP servers.

### Type

```
n8n-nodes-base.emailSend
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `smtpHost` | string | Yes | SMTP server hostname |
| `smtpPort` | number | Yes | SMTP port |
| `username` | string | No | SMTP username |
| `password` | string | No | SMTP password |
| `fromEmail` | string | Yes | Sender email address |
| `toEmail` | string | Yes | Recipient email address |
| `subject` | string | Yes | Email subject |
| `body` | string | No | Email body content |
| `html` | boolean | No | Send as HTML |

### Common SMTP Ports

| Port | Security | Use Case |
|------|----------|----------|
| 25 | None | Legacy, often blocked |
| 465 | SSL/TLS | Secure (implicit TLS) |
| 587 | STARTTLS | Secure (explicit TLS), recommended |

### Examples

#### Basic Email

```json
{
  "id": "email-1",
  "name": "Send Email",
  "type": "n8n-nodes-base.emailSend",
  "position": [450, 300],
  "parameters": {
    "smtpHost": "smtp.gmail.com",
    "smtpPort": 587,
    "username": "your-email@gmail.com",
    "password": "={{ $credentials.gmail.appPassword }}",
    "fromEmail": "your-email@gmail.com",
    "toEmail": "recipient@example.com",
    "subject": "Hello from m9m",
    "body": "This is a test email sent from m9m workflow."
  }
}
```

#### Dynamic Content

```json
{
  "type": "n8n-nodes-base.emailSend",
  "parameters": {
    "smtpHost": "smtp.office365.com",
    "smtpPort": 587,
    "username": "={{ $credentials.smtp.username }}",
    "password": "={{ $credentials.smtp.password }}",
    "fromEmail": "notifications@company.com",
    "toEmail": "={{ $json.userEmail }}",
    "subject": "Order Confirmation #{{ $json.orderId }}",
    "body": "Thank you for your order!\n\nOrder ID: {{ $json.orderId }}\nTotal: ${{ $json.total }}\n\nItems:\n{{ $json.items.map(i => '- ' + i.name).join('\\n') }}"
  }
}
```

#### HTML Email

```json
{
  "type": "n8n-nodes-base.emailSend",
  "parameters": {
    "smtpHost": "smtp.sendgrid.net",
    "smtpPort": 587,
    "username": "apikey",
    "password": "={{ $credentials.sendgrid.apiKey }}",
    "fromEmail": "noreply@company.com",
    "toEmail": "{{ $json.email }}",
    "subject": "Welcome to Our Service",
    "body": "<html><body><h1>Welcome, {{ $json.name }}!</h1><p>Thank you for signing up.</p><a href='https://example.com/verify?token={{ $json.token }}'>Verify your email</a></body></html>",
    "html": true
  }
}
```

#### Alert Email

```json
{
  "type": "n8n-nodes-base.emailSend",
  "parameters": {
    "smtpHost": "smtp.gmail.com",
    "smtpPort": 587,
    "username": "alerts@company.com",
    "password": "={{ $credentials.gmail.appPassword }}",
    "fromEmail": "alerts@company.com",
    "toEmail": "team@company.com",
    "subject": "[ALERT] {{ $json.severity }}: {{ $json.message }}",
    "body": "Alert Details:\n\nSeverity: {{ $json.severity }}\nMessage: {{ $json.message }}\nTimestamp: {{ $json.timestamp }}\nSource: {{ $json.source }}\n\n---\nThis is an automated alert from m9m."
  }
}
```

### Output

```json
{
  "json": {
    "success": true,
    "message": "Email sent successfully"
  }
}
```

### SMTP Provider Configuration

#### Gmail

```json
{
  "smtpHost": "smtp.gmail.com",
  "smtpPort": 587,
  "username": "your-email@gmail.com",
  "password": "your-app-password"
}
```

!!! note "Gmail App Password"
    Use an [App Password](https://support.google.com/accounts/answer/185833) instead of your regular password.

#### Microsoft 365 / Outlook

```json
{
  "smtpHost": "smtp.office365.com",
  "smtpPort": 587,
  "username": "your-email@company.com",
  "password": "your-password"
}
```

#### SendGrid

```json
{
  "smtpHost": "smtp.sendgrid.net",
  "smtpPort": 587,
  "username": "apikey",
  "password": "your-sendgrid-api-key"
}
```

#### Amazon SES

```json
{
  "smtpHost": "email-smtp.us-east-1.amazonaws.com",
  "smtpPort": 587,
  "username": "your-ses-smtp-username",
  "password": "your-ses-smtp-password"
}
```

#### Mailgun

```json
{
  "smtpHost": "smtp.mailgun.org",
  "smtpPort": 587,
  "username": "postmaster@your-domain.mailgun.org",
  "password": "your-mailgun-smtp-password"
}
```

### Common Patterns

#### Multiple Recipients

Send to multiple recipients by using a loop or comma-separated addresses:

```json
{
  "toEmail": "user1@example.com, user2@example.com"
}
```

#### Conditional Send

Use a Filter node before the email node:

```json
{
  "type": "n8n-nodes-base.filter",
  "parameters": {
    "conditions": [
      {
        "leftValue": "={{ $json.shouldNotify }}",
        "operator": "equals",
        "rightValue": true
      }
    ]
  }
}
```

#### Error Notifications

Send email on workflow error:

```json
{
  "type": "n8n-nodes-base.emailSend",
  "parameters": {
    "smtpHost": "smtp.gmail.com",
    "smtpPort": 587,
    "username": "alerts@company.com",
    "password": "={{ $credentials.gmail.appPassword }}",
    "fromEmail": "alerts@company.com",
    "toEmail": "devops@company.com",
    "subject": "Workflow Error: {{ $json.workflowName }}",
    "body": "Error occurred:\n\n{{ $json.error }}\n\nNode: {{ $json.nodeName }}\nTime: {{ $now }}"
  }
}
```

### Best Practices

1. **Use App Passwords** - Don't use main account passwords
2. **Store credentials securely** - Use credential manager
3. **Handle bounces** - Consider email validation
4. **Rate limiting** - Respect provider limits
5. **Test thoroughly** - Use test email addresses

### Troubleshooting

| Issue | Solution |
|-------|----------|
| Connection timeout | Check firewall, try different port |
| Authentication failed | Verify credentials, check app password |
| TLS error | Ensure correct port (587 for STARTTLS) |
| Email not delivered | Check spam folder, verify sender |

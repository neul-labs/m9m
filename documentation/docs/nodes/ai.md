# AI & LLM Nodes

AI nodes integrate with large language models for text generation and analysis.

## OpenAI Node

Interact with OpenAI's GPT models.

### Type

```
n8n-nodes-base.openAi
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `apiKey` | string | Yes | - | OpenAI API key |
| `model` | string | No | `gpt-3.5-turbo` | Model name |
| `prompt` | string | Yes | - | Input prompt |
| `maxTokens` | number | No | 1000 | Max response tokens |
| `temperature` | number | No | 0.7 | Randomness (0-2) |

### Models

| Model | Description |
|-------|-------------|
| `gpt-4` | Most capable, slower |
| `gpt-4-turbo` | Fast GPT-4 |
| `gpt-3.5-turbo` | Fast, cost-effective |

### Examples

#### Basic Completion

```json
{
  "id": "openai-1",
  "name": "Generate Text",
  "type": "n8n-nodes-base.openAi",
  "position": [450, 300],
  "parameters": {
    "apiKey": "={{ $credentials.openai.apiKey }}",
    "model": "gpt-3.5-turbo",
    "prompt": "Summarize this article: {{ $json.content }}",
    "maxTokens": 500
  }
}
```

#### Dynamic Prompt

```json
{
  "type": "n8n-nodes-base.openAi",
  "parameters": {
    "apiKey": "={{ $credentials.openai.apiKey }}",
    "model": "gpt-4",
    "prompt": "Analyze the following customer feedback and extract:\n1. Sentiment (positive/negative/neutral)\n2. Key topics\n3. Action items\n\nFeedback: {{ $json.feedback }}",
    "maxTokens": 1000,
    "temperature": 0.3
  }
}
```

#### Code Generation

```json
{
  "type": "n8n-nodes-base.openAi",
  "parameters": {
    "apiKey": "={{ $credentials.openai.apiKey }}",
    "model": "gpt-4",
    "prompt": "Write a {{ $json.language }} function that {{ $json.description }}",
    "maxTokens": 2000,
    "temperature": 0.2
  }
}
```

### Output

```json
{
  "json": {
    "response": "Generated text content...",
    "usage": {
      "prompt_tokens": 50,
      "completion_tokens": 150,
      "total_tokens": 200
    },
    "finish_reason": "stop"
  }
}
```

| Field | Description |
|-------|-------------|
| `response` | Generated text |
| `usage` | Token usage statistics |
| `finish_reason` | Why generation stopped |

### Temperature Guide

| Temperature | Use Case |
|-------------|----------|
| 0.0 - 0.3 | Factual, deterministic |
| 0.4 - 0.7 | Balanced creativity |
| 0.8 - 1.2 | Creative writing |
| 1.3 - 2.0 | Highly random |

---

## Anthropic Node

Interact with Anthropic's Claude models.

### Type

```
n8n-nodes-base.anthropic
```

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `apiKey` | string | Yes | - | Anthropic API key |
| `model` | string | No | `claude-3-5-sonnet-20241022` | Model name |
| `prompt` | string | Yes | - | Input prompt |
| `maxTokens` | number | No | 1024 | Max response tokens |
| `temperature` | number | No | 1.0 | Randomness (0-1) |

### Models

| Model | Description |
|-------|-------------|
| `claude-3-5-sonnet-20241022` | Latest, balanced |
| `claude-3-opus-20240229` | Most capable |
| `claude-3-sonnet-20240229` | Fast, capable |
| `claude-3-haiku-20240307` | Fastest, economical |

### Examples

#### Basic Completion

```json
{
  "id": "anthropic-1",
  "name": "Claude Response",
  "type": "n8n-nodes-base.anthropic",
  "position": [450, 300],
  "parameters": {
    "apiKey": "={{ $credentials.anthropic.apiKey }}",
    "model": "claude-3-5-sonnet-20241022",
    "prompt": "Explain {{ $json.topic }} in simple terms.",
    "maxTokens": 500
  }
}
```

#### Analysis Task

```json
{
  "type": "n8n-nodes-base.anthropic",
  "parameters": {
    "apiKey": "={{ $credentials.anthropic.apiKey }}",
    "model": "claude-3-opus-20240229",
    "prompt": "Analyze this document and provide:\n1. Summary (2-3 sentences)\n2. Key points\n3. Recommendations\n\nDocument:\n{{ $json.document }}",
    "maxTokens": 2000,
    "temperature": 0.5
  }
}
```

#### Data Extraction

```json
{
  "type": "n8n-nodes-base.anthropic",
  "parameters": {
    "apiKey": "={{ $credentials.anthropic.apiKey }}",
    "model": "claude-3-sonnet-20240229",
    "prompt": "Extract the following information from this email as JSON:\n- sender_name\n- subject\n- action_items (array)\n- urgency (low/medium/high)\n\nEmail:\n{{ $json.emailBody }}\n\nRespond only with valid JSON.",
    "maxTokens": 500,
    "temperature": 0.1
  }
}
```

### Output

```json
{
  "json": {
    "response": "Generated response text...",
    "usage": {
      "input_tokens": 100,
      "output_tokens": 200
    },
    "stop_reason": "end_turn"
  }
}
```

---

## Common Patterns

### Prompt Templates

Store prompts in variables:

```json
{
  "type": "n8n-nodes-base.set",
  "parameters": {
    "assignments": [
      {
        "name": "prompt",
        "value": "You are a helpful assistant. User query: {{ $json.userInput }}"
      }
    ]
  }
}
```

### Error Handling

Check for API errors:

```json
{
  "type": "n8n-nodes-base.filter",
  "parameters": {
    "conditions": [
      {
        "leftValue": "={{ $json.error }}",
        "operator": "notExists"
      }
    ]
  }
}
```

### Response Parsing

Parse JSON from AI response:

```json
{
  "type": "n8n-nodes-base.code",
  "parameters": {
    "language": "javascript",
    "code": "const response = items[0].json.response;\ntry {\n  return [{json: JSON.parse(response)}];\n} catch(e) {\n  return [{json: {raw: response, parseError: true}}];\n}"
  }
}
```

---

## Quick Reference

| Node | Type | Models |
|------|------|--------|
| OpenAI | `n8n-nodes-base.openAi` | GPT-4, GPT-3.5-turbo |
| Anthropic | `n8n-nodes-base.anthropic` | Claude 3.5, Claude 3 |

### Use Cases

| Use Case | Recommended | Model |
|----------|-------------|-------|
| Summarization | Either | Sonnet/GPT-4 |
| Code generation | OpenAI | GPT-4 |
| Analysis | Anthropic | Opus |
| Quick responses | Either | Haiku/GPT-3.5 |
| Data extraction | Either | Sonnet/GPT-4 |

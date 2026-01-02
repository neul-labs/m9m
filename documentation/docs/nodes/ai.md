# AI Nodes

AI nodes enable integration with Large Language Models (LLMs) and AI services.

## OpenAI

### Chat Completion

```json
{
  "type": "n8n-nodes-base.openai",
  "parameters": {
    "resource": "chat",
    "operation": "complete",
    "model": "gpt-4",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful assistant."
      },
      {
        "role": "user",
        "content": "={{ $json.userMessage }}"
      }
    ]
  },
  "credentials": {
    "openAiApi": {"id": "1", "name": "OpenAI"}
  }
}
```

### With Temperature

```json
{
  "parameters": {
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "={{ $json.prompt }}"}
    ],
    "options": {
      "temperature": 0.7,
      "maxTokens": 1000
    }
  }
}
```

### Embeddings

```json
{
  "parameters": {
    "resource": "embedding",
    "operation": "create",
    "model": "text-embedding-3-small",
    "input": "={{ $json.text }}"
  }
}
```

### Image Generation

```json
{
  "parameters": {
    "resource": "image",
    "operation": "generate",
    "prompt": "={{ $json.imageDescription }}",
    "model": "dall-e-3",
    "size": "1024x1024"
  }
}
```

### Credential Setup

```json
{
  "type": "openAiApi",
  "data": {
    "apiKey": "sk-your-openai-api-key"
  }
}
```

## Anthropic (Claude)

### Message

```json
{
  "type": "n8n-nodes-base.anthropic",
  "parameters": {
    "model": "claude-3-opus-20240229",
    "messages": [
      {
        "role": "user",
        "content": "={{ $json.prompt }}"
      }
    ],
    "maxTokens": 1024
  },
  "credentials": {
    "anthropicApi": {"id": "1", "name": "Anthropic"}
  }
}
```

### With System Prompt

```json
{
  "parameters": {
    "model": "claude-3-sonnet-20240229",
    "system": "You are an expert data analyst. Provide concise, accurate analysis.",
    "messages": [
      {"role": "user", "content": "Analyze this data: {{ JSON.stringify($json.data) }}"}
    ]
  }
}
```

### Credential Setup

```json
{
  "type": "anthropicApi",
  "data": {
    "apiKey": "sk-ant-your-anthropic-key"
  }
}
```

## Common AI Patterns

### Text Classification

```json
{
  "parameters": {
    "model": "gpt-4",
    "messages": [
      {
        "role": "system",
        "content": "Classify the following text into one of these categories: support, sales, billing, other. Respond with only the category name."
      },
      {
        "role": "user",
        "content": "={{ $json.emailBody }}"
      }
    ],
    "options": {
      "temperature": 0
    }
  }
}
```

### Summarization

```json
{
  "parameters": {
    "model": "gpt-4",
    "messages": [
      {
        "role": "system",
        "content": "Summarize the following text in 2-3 sentences."
      },
      {
        "role": "user",
        "content": "={{ $json.article }}"
      }
    ]
  }
}
```

### Data Extraction

```json
{
  "parameters": {
    "model": "gpt-4",
    "messages": [
      {
        "role": "system",
        "content": "Extract the following information from the text and return as JSON: name, email, phone, company. If not found, use null."
      },
      {
        "role": "user",
        "content": "={{ $json.rawText }}"
      }
    ],
    "options": {
      "responseFormat": {"type": "json_object"}
    }
  }
}
```

### Sentiment Analysis

```json
{
  "parameters": {
    "model": "gpt-4",
    "messages": [
      {
        "role": "system",
        "content": "Analyze the sentiment of the following text. Return JSON with: sentiment (positive/negative/neutral), confidence (0-1), and key_phrases (array of important phrases)."
      },
      {
        "role": "user",
        "content": "={{ $json.reviewText }}"
      }
    ]
  }
}
```

### Translation

```json
{
  "parameters": {
    "model": "gpt-4",
    "messages": [
      {
        "role": "system",
        "content": "Translate the following text to {{ $json.targetLanguage }}. Preserve formatting."
      },
      {
        "role": "user",
        "content": "={{ $json.text }}"
      }
    ]
  }
}
```

## Advanced Usage

### Function Calling (OpenAI)

```json
{
  "parameters": {
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "What's the weather in San Francisco?"}
    ],
    "functions": [
      {
        "name": "get_weather",
        "description": "Get current weather for a location",
        "parameters": {
          "type": "object",
          "properties": {
            "location": {"type": "string"},
            "unit": {"type": "string", "enum": ["celsius", "fahrenheit"]}
          },
          "required": ["location"]
        }
      }
    ]
  }
}
```

### Streaming (Code Node)

```javascript
const response = await fetch('https://api.openai.com/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${$env.OPENAI_API_KEY}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    model: 'gpt-4',
    messages: $json.messages,
    stream: true
  })
});

let fullResponse = '';
for await (const chunk of response.body) {
  const text = new TextDecoder().decode(chunk);
  // Process streaming chunks
  fullResponse += text;
}

return { json: { response: fullResponse } };
```

### Conversation History

Maintain context across messages:

```json
{
  "nodes": [
    {
      "id": "get-history",
      "type": "n8n-nodes-base.redis",
      "parameters": {
        "operation": "get",
        "key": "conversation:{{ $json.userId }}"
      }
    },
    {
      "id": "ai-chat",
      "type": "n8n-nodes-base.openai",
      "parameters": {
        "messages": "={{ JSON.parse($json.history || '[]').concat([{role: 'user', content: $json.message}]) }}"
      }
    },
    {
      "id": "save-history",
      "type": "n8n-nodes-base.redis",
      "parameters": {
        "operation": "set",
        "key": "conversation:{{ $json.userId }}",
        "value": "={{ JSON.stringify($json.messages) }}",
        "expire": 3600
      }
    }
  ]
}
```

## Rate Limiting & Costs

### Token Counting

```javascript
// Approximate token count
const tokenCount = Math.ceil($json.text.length / 4);

if (tokenCount > 4000) {
  // Split into chunks
  const chunks = splitIntoChunks($json.text, 4000);
  return chunks.map(chunk => ({ json: { text: chunk } }));
}
```

### Rate Limit Handling

```json
{
  "retryOnFail": true,
  "maxRetries": 5,
  "retryConditions": {
    "statusCodes": [429]
  },
  "retryStrategy": "exponential",
  "retryBaseInterval": 5000
}
```

### Cost Tracking

```javascript
const inputTokens = $json.usage.prompt_tokens;
const outputTokens = $json.usage.completion_tokens;

// GPT-4 pricing (example)
const inputCost = (inputTokens / 1000) * 0.03;
const outputCost = (outputTokens / 1000) * 0.06;
const totalCost = inputCost + outputCost;

return {
  json: {
    ...$json,
    cost: {
      input: inputCost,
      output: outputCost,
      total: totalCost
    }
  }
};
```

## Best Practices

1. **Use appropriate temperature** - 0 for deterministic, 0.7-1 for creative
2. **Limit max tokens** - Prevent runaway costs
3. **Cache responses** - For repeated queries
4. **Handle rate limits** - Implement exponential backoff
5. **Validate outputs** - Parse and verify AI responses
6. **Use system prompts** - Guide model behavior consistently
7. **Monitor costs** - Track token usage

## Security

1. **Never expose API keys** in workflows
2. **Sanitize user inputs** before sending to AI
3. **Filter AI outputs** for sensitive content
4. **Log AI interactions** for audit

## Next Steps

- [Cloud Nodes](cloud.md) - Cloud service integrations
- [Transform Nodes](transform.md) - Process AI responses
- [Custom Nodes](custom-nodes.md) - Build custom AI integrations

package copilot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	require.NotNil(t, config)
	assert.Equal(t, ProviderOpenAI, config.Provider)
	assert.Equal(t, "gpt-4", config.Model)
	assert.Equal(t, 4096, config.MaxTokens)
	assert.Equal(t, 0.7, config.Temperature)
	assert.Equal(t, 60*1e9, float64(config.Timeout)) // 60s in ns
}

func TestNewCopilot_NilConfig(t *testing.T) {
	c := NewCopilot(nil)
	require.NotNil(t, c)
	assert.Equal(t, ProviderOpenAI, c.config.Provider)
	assert.NotNil(t, c.httpClient)
	assert.NotEmpty(t, c.nodeTypes)
}

func TestNewCopilot_CustomConfig(t *testing.T) {
	config := &Config{
		Provider:    ProviderAnthropic,
		APIKey:      "test-key",
		Model:       "claude-3",
		MaxTokens:   8192,
		Temperature: 0.5,
		Timeout:     30 * 1e9,
	}

	c := NewCopilot(config)
	require.NotNil(t, c)
	assert.Equal(t, ProviderAnthropic, c.config.Provider)
	assert.Equal(t, "test-key", c.config.APIKey)
	assert.Equal(t, "claude-3", c.config.Model)
	assert.Equal(t, 8192, c.config.MaxTokens)
}

func TestProviderConstants(t *testing.T) {
	assert.Equal(t, Provider("openai"), ProviderOpenAI)
	assert.Equal(t, Provider("anthropic"), ProviderAnthropic)
	assert.Equal(t, Provider("ollama"), ProviderOllama)
}

func TestGetAvailableNodeTypes(t *testing.T) {
	types := getAvailableNodeTypes()
	assert.NotEmpty(t, types)
	for _, nt := range types {
		assert.NotEmpty(t, nt.Type)
		assert.NotEmpty(t, nt.Name)
	}
}

func TestGenerateWorkflowRequest(t *testing.T) {
	req := &GenerateWorkflowRequest{
		Description: "Send a Slack message when a new GitHub issue is created",
		Context: map[string]interface{}{
			"team": "engineering",
		},
	}
	assert.Equal(t, "Send a Slack message when a new GitHub issue is created", req.Description)
	assert.Equal(t, "engineering", req.Context["team"])
}

func TestSuggestNodesRequest(t *testing.T) {
	req := &SuggestNodesRequest{
		UserQuery: "I need to filter data",
	}
	assert.Equal(t, "I need to filter data", req.UserQuery)
	assert.Nil(t, req.CurrentWorkflow)
}

func TestChatMessage(t *testing.T) {
	msg := ChatMessage{
		Role:    "user",
		Content: "How do I add error handling?",
	}
	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "How do I add error handling?", msg.Content)
}

func TestCopilot_BuildPrompts(t *testing.T) {
	c := NewCopilot(nil)

	t.Run("build generate workflow prompt", func(t *testing.T) {
		req := &GenerateWorkflowRequest{
			Description: "Send email on schedule",
		}
		prompt := c.buildGenerateWorkflowPrompt(req)
		assert.Contains(t, prompt, "Send email on schedule")
	})

	t.Run("build suggest nodes prompt", func(t *testing.T) {
		req := &SuggestNodesRequest{
			UserQuery: "filter data by date",
		}
		prompt := c.buildSuggestNodesPrompt(req)
		assert.Contains(t, prompt, "filter data by date")
	})
}

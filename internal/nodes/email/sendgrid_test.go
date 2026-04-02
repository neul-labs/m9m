package email

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendGridNode_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/mail/send")
		assert.Equal(t, "Bearer sg-api-key", r.Header.Get("Authorization"))

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		from := body["from"].(map[string]interface{})
		assert.Equal(t, "sender@example.com", from["email"])

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	node := NewSendGridNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "send",
		"apiKey":    "sg-api-key",
		"from":      "sender@example.com",
		"to":        "recipient@example.com",
		"subject":   "Test",
		"content":   "Hello!",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, true, result[0].JSON["success"])
}

func TestSendGridNode_SendTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "d-template123", body["template_id"])
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	node := NewSendGridNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":    server.URL,
		"operation":  "sendTemplate",
		"apiKey":     "sg-key",
		"from":       "sender@example.com",
		"to":         "recipient@example.com",
		"templateId": "d-template123",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, true, result[0].JSON["success"])
}

func TestSendGridNode_SendTemplateMissingId(t *testing.T) {
	node := NewSendGridNode()
	_, err := node.Execute(nil, map[string]interface{}{
		"operation": "sendTemplate",
		"apiKey":    "key",
		"from":      "a@b.com",
		"to":        "c@d.com",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "templateId is required")
}

func TestSendGridNode_MissingApiKey(t *testing.T) {
	node := NewSendGridNode()
	_, err := node.Execute(nil, map[string]interface{}{"operation": "send"})
	assert.Error(t, err)
}

func TestSendGridNode_MissingFromTo(t *testing.T) {
	node := NewSendGridNode()
	_, err := node.Execute(nil, map[string]interface{}{
		"operation": "send",
		"apiKey":    "key",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from and to are required")
}

func TestSendGridNode_Validate(t *testing.T) {
	node := NewSendGridNode()
	assert.Error(t, node.ValidateParameters(nil))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{
		"apiKey":    "key",
		"operation": "send",
	}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{
		"apiKey":    "key",
		"operation": "invalid",
	}))
}

func TestSendGridNode_Description(t *testing.T) {
	node := NewSendGridNode()
	desc := node.Description()
	assert.Equal(t, "SendGrid", desc.Name)
	assert.Equal(t, "Email", desc.Category)
}

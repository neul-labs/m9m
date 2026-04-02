package messaging

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTwilioNode_SendSms(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/Messages.json")
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		user, pass, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "AC123", user)
		assert.Equal(t, "token123", pass)

		r.ParseForm()
		assert.Equal(t, "+15551234567", r.FormValue("From"))
		assert.Equal(t, "+15559876543", r.FormValue("To"))
		assert.Equal(t, "Hello!", r.FormValue("Body"))

		json.NewEncoder(w).Encode(map[string]interface{}{
			"sid":    "SM123",
			"status": "queued",
		})
	}))
	defer server.Close()

	node := NewTwilioNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":    server.URL,
		"operation":  "sendSms",
		"accountSid": "AC123",
		"authToken":  "token123",
		"from":       "+15551234567",
		"to":         "+15559876543",
		"body":       "Hello!",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "SM123", result[0].JSON["sid"])
	assert.Equal(t, "queued", result[0].JSON["status"])
}

func TestTwilioNode_GetMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"messages": []interface{}{
				map[string]interface{}{"sid": "SM1", "body": "msg1"},
			},
		})
	}))
	defer server.Close()

	node := NewTwilioNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":    server.URL,
		"operation":  "getMessages",
		"accountSid": "AC123",
		"authToken":  "token123",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.NotNil(t, result[0].JSON["messages"])
}

func TestTwilioNode_MissingAuth(t *testing.T) {
	node := NewTwilioNode()
	_, err := node.Execute(nil, map[string]interface{}{"operation": "sendSms"})
	assert.Error(t, err)
}

func TestTwilioNode_SendSmsMissingFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := NewTwilioNodeWithClient(server.Client())
	_, err := node.Execute(nil, map[string]interface{}{
		"baseUrl":    server.URL,
		"operation":  "sendSms",
		"accountSid": "AC123",
		"authToken":  "token",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from, to, and body are required")
}

func TestTwilioNode_Validate(t *testing.T) {
	node := NewTwilioNode()
	assert.Error(t, node.ValidateParameters(nil))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{"accountSid": "x"}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{
		"accountSid": "AC123",
		"authToken":  "token",
	}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{
		"accountSid": "AC123",
		"authToken":  "token",
		"operation":  "invalid",
	}))
}

func TestTwilioNode_Description(t *testing.T) {
	node := NewTwilioNode()
	desc := node.Description()
	assert.Equal(t, "Twilio", desc.Name)
	assert.Equal(t, "Messaging", desc.Category)
}

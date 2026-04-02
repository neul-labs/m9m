package messaging

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamsNode_SendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/teams/team1/channels/chan1/messages")
		assert.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		bodyContent := body["body"].(map[string]interface{})
		assert.Equal(t, "Hello Teams!", bodyContent["content"])

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "msg123",
			"createdAt": "2024-01-01T00:00:00Z",
		})
	}))
	defer server.Close()

	node := NewTeamsNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":     server.URL,
		"operation":   "sendMessage",
		"accessToken": "mytoken",
		"teamId":      "team1",
		"channelId":   "chan1",
		"text":        "Hello Teams!",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "msg123", result[0].JSON["id"])
}

func TestTeamsNode_ListChannels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/teams/team1/channels")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"value": []interface{}{
				map[string]interface{}{"id": "chan1", "displayName": "General"},
				map[string]interface{}{"id": "chan2", "displayName": "Random"},
			},
		})
	}))
	defer server.Close()

	node := NewTeamsNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":     server.URL,
		"operation":   "listChannels",
		"accessToken": "mytoken",
		"teamId":      "team1",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.NotNil(t, result[0].JSON["value"])
}

func TestTeamsNode_MissingAuth(t *testing.T) {
	node := NewTeamsNode()
	_, err := node.Execute(nil, map[string]interface{}{"operation": "sendMessage"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accessToken is required")
}

func TestTeamsNode_SendMessageMissingFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := NewTeamsNodeWithClient(server.Client())

	// Missing teamId/channelId
	_, err := node.Execute(nil, map[string]interface{}{
		"baseUrl":     server.URL,
		"operation":   "sendMessage",
		"accessToken": "tok",
	})
	assert.Error(t, err)

	// Missing text
	_, err = node.Execute(nil, map[string]interface{}{
		"baseUrl":     server.URL,
		"operation":   "sendMessage",
		"accessToken": "tok",
		"teamId":      "t1",
		"channelId":   "c1",
	})
	assert.Error(t, err)
}

func TestTeamsNode_ListChannelsMissingTeamId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	node := NewTeamsNodeWithClient(server.Client())
	_, err := node.Execute(nil, map[string]interface{}{
		"baseUrl":     server.URL,
		"operation":   "listChannels",
		"accessToken": "tok",
	})
	assert.Error(t, err)
}

func TestTeamsNode_Validate(t *testing.T) {
	node := NewTeamsNode()
	assert.Error(t, node.ValidateParameters(nil))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{
		"accessToken": "tok",
		"operation":   "sendMessage",
	}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{
		"accessToken": "tok",
		"operation":   "invalid",
	}))
}

func TestTeamsNode_Description(t *testing.T) {
	node := NewTeamsNode()
	desc := node.Description()
	assert.Equal(t, "Microsoft Teams", desc.Name)
	assert.Equal(t, "Messaging", desc.Category)
}

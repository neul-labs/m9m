package messaging

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport rewrites request URLs to point at the test server.
type rewriteTransport struct {
	base    http.RoundTripper
	baseURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	u, _ := url.Parse(t.baseURL)
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
	return t.base.RoundTrip(req)
}

// ===========================================================================
// SlackNode additional tests (supplements slack_test.go)
// ===========================================================================

// --- Interface compliance ---

func TestSlackNode_ImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*SlackNode)(nil)
}

// --- Webhook path: verify that webhook takes priority over token ---

func TestSlackNode_Execute_WebhookPriorityOverToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When webhook is used, there should be no Authorization header
		assert.Empty(t, r.Header.Get("Authorization"),
			"webhook path should not set an Authorization header")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	node := NewSlackNode()
	node.httpClient = server.Client()

	inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
	params := map[string]interface{}{
		"webhookUrl": server.URL,
		"token":      "xoxb-should-be-ignored",
		"channel":    "#general",
		"text":       "hello",
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "webhook", results[0].JSON["method"],
		"webhook should be preferred when both webhookUrl and token are provided")
}

// --- Webhook path: verify Content-Type header ---

func TestSlackNode_Execute_Webhook_ContentTypeHeader(t *testing.T) {
	var capturedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	node := NewSlackNode()
	node.httpClient = server.Client()

	inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
	params := map[string]interface{}{
		"webhookUrl": server.URL,
		"text":       "hello",
	}

	_, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.Equal(t, "application/json", capturedContentType)
}

// --- Webhook path: the node sends one HTTP request per input item ---

func TestSlackNode_Execute_Webhook_RequestCountMatchesInputCount(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	node := NewSlackNode()
	node.httpClient = server.Client()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"a": 1}},
		{JSON: map[string]interface{}{"b": 2}},
		{JSON: map[string]interface{}{"c": 3}},
		{JSON: map[string]interface{}{"d": 4}},
		{JSON: map[string]interface{}{"e": 5}},
	}
	params := map[string]interface{}{
		"webhookUrl": server.URL,
		"text":       "bulk",
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.Len(t, results, 5)
	assert.Equal(t, int32(5), atomic.LoadInt32(&requestCount))
}

// --- Webhook path: first failure stops iteration ---

func TestSlackNode_Execute_Webhook_StopsOnFirstError(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count == 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	node := NewSlackNode()
	node.httpClient = server.Client()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"a": 1}},
		{JSON: map[string]interface{}{"b": 2}},
		{JSON: map[string]interface{}{"c": 3}},
	}
	params := map[string]interface{}{
		"webhookUrl": server.URL,
		"text":       "hello",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	// The second request fails, so the third should never be sent
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount),
		"processing should stop after the first HTTP error")
}

// --- API path: verify Authorization header format ---

func TestSlackNode_Execute_API_AuthorizationHeaderFormat(t *testing.T) {
	var capturedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	node := NewSlackNode()
	node.httpClient = &http.Client{
		Transport: &rewriteTransport{
			base:    server.Client().Transport,
			baseURL: server.URL,
		},
	}

	inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
	params := map[string]interface{}{
		"token":   "xoxb-my-secret-token",
		"channel": "#dev",
		"text":    "auth test",
	}

	_, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.Equal(t, "Bearer xoxb-my-secret-token", capturedAuth)
}

// --- API path: result shape ---

func TestSlackNode_Execute_API_ResultShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
	}))
	defer server.Close()

	node := NewSlackNode()
	node.httpClient = &http.Client{
		Transport: &rewriteTransport{
			base:    server.Client().Transport,
			baseURL: server.URL,
		},
	}

	inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
	params := map[string]interface{}{
		"token":   "xoxb-token",
		"channel": "#general",
		"text":    "shape test",
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)

	item := results[0].JSON
	assert.Equal(t, true, item["success"])
	assert.Equal(t, "shape test", item["text"])
	assert.Equal(t, "#general", item["channel"])
	assert.Equal(t, "api", item["method"])
}

// --- Nil params should be handled gracefully ---

func TestSlackNode_Execute_NilParams(t *testing.T) {
	node := NewSlackNode()
	_, err := node.Execute(
		[]model.DataItem{{JSON: map[string]interface{}{}}},
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "either webhookUrl or token is required")
}

// --- Webhook path: various non-200 status codes ---

func TestSlackNode_Execute_Webhook_VariousStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusTooManyRequests,
		http.StatusServiceUnavailable,
	}

	for _, code := range statusCodes {
		t.Run(http.StatusText(code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer server.Close()

			node := NewSlackNode()
			node.httpClient = server.Client()

			inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
			params := map[string]interface{}{
				"webhookUrl": server.URL,
				"text":       "hello",
			}

			_, err := node.Execute(inputData, params)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to send message")
		})
	}
}

// ===========================================================================
// DiscordNode additional tests (supplements discord_test.go)
// ===========================================================================

// --- Interface compliance ---

func TestDiscordNode_ImplementsNodeExecutor(t *testing.T) {
	var _ base.NodeExecutor = (*DiscordNode)(nil)
}

// --- Successful 2xx range: both 200 and 204 should succeed ---

func TestDiscordNode_Execute_Various2xxStatusCodes(t *testing.T) {
	successCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusNoContent,
	}

	for _, code := range successCodes {
		t.Run(http.StatusText(code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer server.Close()

			node := NewDiscordNode()
			node.httpClient = server.Client()

			inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
			params := map[string]interface{}{
				"webhookUrl": server.URL,
				"content":    "hello",
			}

			results, err := node.Execute(inputData, params)
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, true, results[0].JSON["success"])
			assert.Equal(t, code, results[0].JSON["status"])
		})
	}
}

// --- Error on non-2xx status (boundary check) ---

func TestDiscordNode_Execute_BoundaryStatusCodes(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		expectErr bool
	}{
		{"200 is success", 200, false},
		{"204 is success", 204, false},
		{"299 is success", 299, false},
		{"300 is an error", 300, true},
		{"400 is an error", 400, true},
		{"500 is an error", 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			node := NewDiscordNode()
			node.httpClient = server.Client()

			inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
			params := map[string]interface{}{
				"webhookUrl": server.URL,
				"content":    "hello",
			}

			results, err := node.Execute(inputData, params)
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "discord returned status")
			} else {
				require.NoError(t, err)
				require.Len(t, results, 1)
			}
		})
	}
}

// --- Request body: avatar_url is only included when non-empty ---

func TestDiscordNode_Execute_AvatarURLConditionalInclusion(t *testing.T) {
	tests := []struct {
		name          string
		avatarURL     string
		expectInBody  bool
	}{
		{"avatar provided", "https://example.com/img.png", true},
		{"avatar empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPayload map[string]interface{}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				defer r.Body.Close()
				json.Unmarshal(body, &capturedPayload)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			node := NewDiscordNode()
			node.httpClient = server.Client()

			inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
			params := map[string]interface{}{
				"webhookUrl": server.URL,
				"content":    "hello",
			}
			if tt.avatarURL != "" {
				params["avatarUrl"] = tt.avatarURL
			}

			_, err := node.Execute(inputData, params)
			require.NoError(t, err)

			_, hasAvatar := capturedPayload["avatar_url"]
			assert.Equal(t, tt.expectInBody, hasAvatar)
		})
	}
}

// --- First failure should stop iteration ---

func TestDiscordNode_Execute_StopsOnFirstError(t *testing.T) {
	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count == 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	node := NewDiscordNode()
	node.httpClient = server.Client()

	inputData := []model.DataItem{
		{JSON: map[string]interface{}{"a": 1}},
		{JSON: map[string]interface{}{"b": 2}},
		{JSON: map[string]interface{}{"c": 3}},
	}
	params := map[string]interface{}{
		"webhookUrl": server.URL,
		"content":    "hello",
	}

	_, err := node.Execute(inputData, params)
	require.Error(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount),
		"processing should stop after the first HTTP error")
}

// --- Nil params ---

func TestDiscordNode_Execute_NilParams(t *testing.T) {
	node := NewDiscordNode()
	_, err := node.Execute(
		[]model.DataItem{{JSON: map[string]interface{}{}}},
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhookUrl is required")
}

// --- Result username matches what was sent ---

func TestDiscordNode_Execute_ResultUsernameMatchesParam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	node := NewDiscordNode()
	node.httpClient = server.Client()

	inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
	params := map[string]interface{}{
		"webhookUrl": server.URL,
		"content":    "hello",
		"username":   "SpecialBot",
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "SpecialBot", results[0].JSON["username"])
}

// --- Result username defaults to "m9m Bot" ---

func TestDiscordNode_Execute_ResultUsernameDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	node := NewDiscordNode()
	node.httpClient = server.Client()

	inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
	params := map[string]interface{}{
		"webhookUrl": server.URL,
		"content":    "hello",
	}

	results, err := node.Execute(inputData, params)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "m9m Bot", results[0].JSON["username"])
}

// --- Content-Type header ---

func TestDiscordNode_Execute_ContentTypeHeader(t *testing.T) {
	var capturedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	node := NewDiscordNode()
	node.httpClient = server.Client()

	inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
	params := map[string]interface{}{
		"webhookUrl": server.URL,
		"content":    "hello",
	}

	_, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.Equal(t, "application/json", capturedContentType)
}

// --- HTTP method is POST ---

func TestDiscordNode_Execute_UsesPostMethod(t *testing.T) {
	var capturedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	node := NewDiscordNode()
	node.httpClient = server.Client()

	inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}
	params := map[string]interface{}{
		"webhookUrl": server.URL,
		"content":    "hello",
	}

	_, err := node.Execute(inputData, params)
	require.NoError(t, err)
	assert.Equal(t, "POST", capturedMethod)
}

// ===========================================================================
// Cross-node comparison tests
// ===========================================================================

func TestBothNodes_DefaultUsernameMatches(t *testing.T) {
	// Both Slack and Discord nodes should use "m9m Bot" as the default username.
	// This is a cross-cutting concern so we verify them side by side.

	var slackUsername, discordUsername string

	slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		var payload map[string]interface{}
		json.Unmarshal(body, &payload)
		slackUsername = payload["username"].(string)
		w.WriteHeader(http.StatusOK)
	}))
	defer slackServer.Close()

	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		var payload map[string]interface{}
		json.Unmarshal(body, &payload)
		discordUsername = payload["username"].(string)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer discordServer.Close()

	inputData := []model.DataItem{{JSON: map[string]interface{}{"k": "v"}}}

	slackNode := NewSlackNode()
	slackNode.httpClient = slackServer.Client()
	_, err := slackNode.Execute(inputData, map[string]interface{}{
		"webhookUrl": slackServer.URL,
		"text":       "hello",
	})
	require.NoError(t, err)

	discordNode := NewDiscordNode()
	discordNode.httpClient = discordServer.Client()
	_, err = discordNode.Execute(inputData, map[string]interface{}{
		"webhookUrl": discordServer.URL,
		"content":    "hello",
	})
	require.NoError(t, err)

	assert.Equal(t, "m9m Bot", slackUsername)
	assert.Equal(t, "m9m Bot", discordUsername)
	assert.Equal(t, slackUsername, discordUsername,
		"both nodes should have the same default username")
}

func TestBothNodes_CategoryIsCommunication(t *testing.T) {
	slackNode := NewSlackNode()
	discordNode := NewDiscordNode()

	assert.Equal(t, "communication", slackNode.Description().Category)
	assert.Equal(t, "communication", discordNode.Description().Category)
}

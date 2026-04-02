package productivity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripeNode_CreateCustomer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/customers")
		assert.Equal(t, "Bearer sk_test_123", r.Header.Get("Authorization"))
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		r.ParseForm()
		assert.Equal(t, "alice@example.com", r.FormValue("email"))
		assert.Equal(t, "Alice", r.FormValue("name"))

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "cus_abc123",
			"object": "customer",
			"email":  "alice@example.com",
		})
	}))
	defer server.Close()

	node := NewStripeNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "createCustomer",
		"apiKey":    "sk_test_123",
		"email":     "alice@example.com",
		"name":      "Alice",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "cus_abc123", result[0].JSON["id"])
	assert.Equal(t, "customer", result[0].JSON["object"])
}

func TestStripeNode_CreatePaymentIntent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/payment_intents")

		r.ParseForm()
		assert.Equal(t, "2000", r.FormValue("amount"))
		assert.Equal(t, "usd", r.FormValue("currency"))

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       "pi_xyz",
			"object":   "payment_intent",
			"amount":   float64(2000),
			"currency": "usd",
		})
	}))
	defer server.Close()

	node := NewStripeNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "createPaymentIntent",
		"apiKey":    "sk_test_123",
		"amount":    2000,
		"currency":  "usd",
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.Equal(t, "pi_xyz", result[0].JSON["id"])
	assert.Equal(t, float64(2000), result[0].JSON["amount"])
}

func TestStripeNode_CreatePaymentIntentZeroAmount(t *testing.T) {
	node := NewStripeNode()
	_, err := node.Execute(nil, map[string]interface{}{
		"operation": "createPaymentIntent",
		"apiKey":    "key",
		"amount":    0,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be greater than 0")
}

func TestStripeNode_ListPayments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/payment_intents")
		assert.Equal(t, "10", r.URL.Query().Get("limit"))

		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{"id": "pi_1", "amount": float64(1000)},
			},
		})
	}))
	defer server.Close()

	node := NewStripeNodeWithClient(server.Client())
	params := map[string]interface{}{
		"baseUrl":   server.URL,
		"operation": "listPayments",
		"apiKey":    "sk_test_123",
		"limit":     10,
	}

	result, err := node.Execute(nil, params)
	require.NoError(t, err)
	assert.NotNil(t, result[0].JSON["data"])
}

func TestStripeNode_MissingApiKey(t *testing.T) {
	node := NewStripeNode()
	_, err := node.Execute(nil, map[string]interface{}{"operation": "createCustomer"})
	assert.Error(t, err)
}

func TestStripeNode_Validate(t *testing.T) {
	node := NewStripeNode()
	assert.Error(t, node.ValidateParameters(nil))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{
		"apiKey":    "key",
		"operation": "createCustomer",
	}))
	assert.NoError(t, node.ValidateParameters(map[string]interface{}{
		"apiKey":    "key",
		"operation": "listPayments",
	}))
	assert.Error(t, node.ValidateParameters(map[string]interface{}{
		"apiKey":    "key",
		"operation": "invalid",
	}))
}

func TestStripeNode_Description(t *testing.T) {
	node := NewStripeNode()
	desc := node.Description()
	assert.Equal(t, "Stripe", desc.Name)
	assert.Equal(t, "Productivity", desc.Category)
}

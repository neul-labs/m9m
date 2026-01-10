package credentials

import (
	"os"
	"testing"
)

func TestCredentialStoreCreation(t *testing.T) {
	store, err := NewCredentialStore()
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	if store == nil {
		t.Fatal("Expected credential store to be created, got nil")
	}
}

func TestCredentialStorageAndRetrieval(t *testing.T) {
	store, err := NewCredentialStore()
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Create a test credential
	cred := &Credential{
		ID:   "test-cred-1",
		Name: "Test Credential",
		Type: "apiKey",
		Data: map[string]interface{}{
			"apiKey": "secret-api-key-123",
			"url":    "https://api.example.com",
		},
	}

	// Store the credential
	err = store.StoreCredential(cred)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Retrieve the credential
	retrieved, err := store.GetCredential("test-cred-1")
	if err != nil {
		t.Fatalf("Failed to retrieve credential: %v", err)
	}

	if retrieved.ID != cred.ID {
		t.Errorf("Expected ID %s, got %s", cred.ID, retrieved.ID)
	}

	if retrieved.Name != cred.Name {
		t.Errorf("Expected Name %s, got %s", cred.Name, retrieved.Name)
	}

	// Check that sensitive data is decrypted
	apiKey, ok := retrieved.Data["apiKey"].(string)
	if !ok {
		t.Fatal("Expected apiKey to be a string")
	}

	if apiKey != "secret-api-key-123" {
		t.Errorf("Expected apiKey 'secret-api-key-123', got '%s'", apiKey)
	}
}

func TestCredentialDeletion(t *testing.T) {
	store, err := NewCredentialStore()
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Create and store a test credential
	cred := &Credential{
		ID:   "test-cred-2",
		Name: "Test Credential 2",
		Type: "basicAuth",
		Data: map[string]interface{}{
			"username": "testuser",
			"password": "testpass",
		},
	}

	err = store.StoreCredential(cred)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Verify it exists
	_, err = store.GetCredential("test-cred-2")
	if err != nil {
		t.Fatalf("Failed to retrieve credential before deletion: %v", err)
	}

	// Delete the credential
	err = store.DeleteCredential("test-cred-2")
	if err != nil {
		t.Fatalf("Failed to delete credential: %v", err)
	}

	// Verify it no longer exists
	_, err = store.GetCredential("test-cred-2")
	if err == nil {
		t.Error("Expected error when retrieving deleted credential, got nil")
	}
}

func TestCredentialListing(t *testing.T) {
	store, err := NewCredentialStore()
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Store multiple credentials
	creds := []*Credential{
		{ID: "cred-1", Name: "Credential 1", Type: "apiKey", Data: map[string]interface{}{"key": "value1"}},
		{ID: "cred-2", Name: "Credential 2", Type: "oauth2", Data: map[string]interface{}{"key": "value2"}},
		{ID: "cred-3", Name: "Credential 3", Type: "basicAuth", Data: map[string]interface{}{"key": "value3"}},
	}

	for _, cred := range creds {
		err = store.StoreCredential(cred)
		if err != nil {
			t.Fatalf("Failed to store credential %s: %v", cred.ID, err)
		}
	}

	// List credentials
	ids := store.ListCredentials()
	if len(ids) != 3 {
		t.Errorf("Expected 3 credentials, got %d", len(ids))
	}

	// Verify all IDs are present
	expectedIDs := map[string]bool{
		"cred-1": true,
		"cred-2": true,
		"cred-3": true,
	}

	for _, id := range ids {
		if !expectedIDs[id] {
			t.Errorf("Unexpected credential ID: %s", id)
		}
		delete(expectedIDs, id)
	}

	if len(expectedIDs) != 0 {
		t.Error("Not all expected IDs were found in listing")
	}
}

func TestEnvironmentVariableResolution(t *testing.T) {
	store, err := NewCredentialStore()
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Set an environment variable for testing
	os.Setenv("TEST_API_KEY", "env-api-key-456")
	defer os.Unsetenv("TEST_API_KEY")

	// Test resolving environment variable
	value, err := store.ResolveCredentialValue("${TEST_API_KEY}")
	if err != nil {
		t.Fatalf("Failed to resolve environment variable: %v", err)
	}

	if value != "env-api-key-456" {
		t.Errorf("Expected 'env-api-key-456', got '%s'", value)
	}

	// Test resolving literal value
	literalValue, err := store.ResolveCredentialValue("literal-value")
	if err != nil {
		t.Fatalf("Failed to resolve literal value: %v", err)
	}

	if literalValue != "literal-value" {
		t.Errorf("Expected 'literal-value', got '%s'", literalValue)
	}

	// Test resolving non-string value
	numValue, err := store.ResolveCredentialValue(42)
	if err != nil {
		t.Fatalf("Failed to resolve numeric value: %v", err)
	}

	if numValue != "42" {
		t.Errorf("Expected '42', got '%s'", numValue)
	}
}

func TestCredentialEncryption(t *testing.T) {
	store, err := NewCredentialStore()
	if err != nil {
		t.Fatalf("Failed to create credential store: %v", err)
	}

	// Create a credential with sensitive data
	cred := &Credential{
		ID:   "encrypted-cred",
		Name: "Encrypted Credential",
		Type: "oauth2",
		Data: map[string]interface{}{
			"clientId":     "client-123",
			"clientSecret": "super-secret-client-secret",
			"accessToken":  "access-token-789",
			"refreshToken": "refresh-token-012",
		},
	}

	// Store the credential (should encrypt automatically)
	err = store.StoreCredential(cred)
	if err != nil {
		t.Fatalf("Failed to store credential: %v", err)
	}

	// Retrieve the credential (should decrypt automatically)
	retrieved, err := store.GetCredential("encrypted-cred")
	if err != nil {
		t.Fatalf("Failed to retrieve credential: %v", err)
	}

	// Verify all sensitive data is decrypted correctly
	expectedData := map[string]string{
		"clientId":     "client-123",
		"clientSecret": "super-secret-client-secret",
		"accessToken":  "access-token-789",
		"refreshToken": "refresh-token-012",
	}

	for key, expectedValue := range expectedData {
		actualValue, ok := retrieved.Data[key].(string)
		if !ok {
			t.Errorf("Expected %s to be a string", key)
			continue
		}

		if actualValue != expectedValue {
			t.Errorf("Expected %s '%s', got '%s'", key, expectedValue, actualValue)
		}
	}
}

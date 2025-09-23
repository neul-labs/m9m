/*
Package credentials provides secure credential management for n8n-go.
*/
package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	
	"golang.org/x/crypto/scrypt"
)

// Credential represents a stored credential
type Credential struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
	Encrypted   bool                   `json:"encrypted"`
	CreatedAt   string                 `json:"createdAt"`
	UpdatedAt   string                 `json:"updatedAt"`
}

// CredentialStore manages credential storage and retrieval
type CredentialStore struct {
	credentials map[string]*Credential
	mu          sync.RWMutex
	encryptionKey []byte
}

// NewCredentialStore creates a new credential store
func NewCredentialStore() (*CredentialStore, error) {
	store := &CredentialStore{
		credentials: make(map[string]*Credential),
	}
	
	// Generate encryption key from environment or use default
	key, err := store.generateEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %v", err)
	}
	
	store.encryptionKey = key
	return store, nil
}

// generateEncryptionKey generates an encryption key from environment or defaults
func (cs *CredentialStore) generateEncryptionKey() ([]byte, error) {
	// Try to get key from environment variable
	envKey := os.Getenv("N8N_ENCRYPTION_KEY")
	if envKey != "" {
		// Use scrypt to derive a proper key from the environment key
		salt := []byte("n8n-go-salt")
		key, err := scrypt.Key([]byte(envKey), salt, 32768, 8, 1, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to derive key from environment: %v", err)
		}
		return key, nil
	}
	
	// Generate a random key as fallback
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %v", err)
	}
	
	return key, nil
}

// StoreCredential stores a credential securely
func (cs *CredentialStore) StoreCredential(cred *Credential) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	// Encrypt sensitive data if not already encrypted
	if !cred.Encrypted {
		encryptedData, err := cs.encryptCredentialData(cred.Data)
		if err != nil {
			return fmt.Errorf("failed to encrypt credential data: %v", err)
		}
		cred.Data = encryptedData
		cred.Encrypted = true
	}
	
	// Store the credential
	cs.credentials[cred.ID] = cred
	return nil
}

// GetCredential retrieves a credential by ID
func (cs *CredentialStore) GetCredential(id string) (*Credential, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	cred, exists := cs.credentials[id]
	if !exists {
		return nil, fmt.Errorf("credential not found: %s", id)
	}
	
	// Decrypt the credential data if it's encrypted
	if cred.Encrypted {
		decryptedData, err := cs.decryptCredentialData(cred.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt credential data: %v", err)
		}
		
		// Create a copy with decrypted data
		decryptedCred := &Credential{
			ID:        cred.ID,
			Name:      cred.Name,
			Type:      cred.Type,
			Data:      decryptedData,
			Encrypted: false,
			CreatedAt: cred.CreatedAt,
			UpdatedAt: cred.UpdatedAt,
		}
		
		return decryptedCred, nil
	}
	
	return cred, nil
}

// DeleteCredential deletes a credential by ID
func (cs *CredentialStore) DeleteCredential(id string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	delete(cs.credentials, id)
	return nil
}

// ListCredentials lists all credential IDs
func (cs *CredentialStore) ListCredentials() []string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	ids := make([]string, 0, len(cs.credentials))
	for id := range cs.credentials {
		ids = append(ids, id)
	}
	
	return ids
}

// encryptCredentialData encrypts credential data
func (cs *CredentialStore) encryptCredentialData(data map[string]interface{}) (map[string]interface{}, error) {
	encryptedData := make(map[string]interface{})
	
	for key, value := range data {
		// Only encrypt string values that might be sensitive
		if strValue, ok := value.(string); ok {
			encryptedValue, err := cs.encryptString(strValue)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt field %s: %v", key, err)
			}
			encryptedData[key] = encryptedValue
		} else {
			// Keep non-string values as-is
			encryptedData[key] = value
		}
	}
	
	return encryptedData, nil
}

// decryptCredentialData decrypts credential data
func (cs *CredentialStore) decryptCredentialData(data map[string]interface{}) (map[string]interface{}, error) {
	decryptedData := make(map[string]interface{})
	
	for key, value := range data {
		// Try to decrypt string values
		if strValue, ok := value.(string); ok {
			decryptedValue, err := cs.decryptString(strValue)
			if err != nil {
				// If decryption fails, keep the original value
				decryptedData[key] = strValue
			} else {
				decryptedData[key] = decryptedValue
			}
		} else {
			// Keep non-string values as-is
			decryptedData[key] = value
		}
	}
	
	return decryptedData, nil
}

// encryptString encrypts a string using AES-GCM
func (cs *CredentialStore) encryptString(plaintext string) (string, error) {
	block, err := aes.NewCipher(cs.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %v", err)
	}
	
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptString decrypts a string using AES-GCM
func (cs *CredentialStore) decryptString(encrypted string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %v", err)
	}
	
	block, err := aes.NewCipher(cs.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %v", err)
	}
	
	return string(plaintext), nil
}

// ResolveCredentialValue resolves a credential value, handling environment variables
func (cs *CredentialStore) ResolveCredentialValue(value interface{}) (string, error) {
	if strValue, ok := value.(string); ok {
		// Check if it's an environment variable reference
		if strings.HasPrefix(strValue, "${") && strings.HasSuffix(strValue, "}") {
			// Extract environment variable name
			envVar := strValue[2 : len(strValue)-1]
			
			// Get value from environment
			envValue := os.Getenv(envVar)
			if envValue == "" {
				return "", fmt.Errorf("environment variable %s is not set", envVar)
			}
			
			return envValue, nil
		}
		
		// Return literal value
		return strValue, nil
	}
	
	// Convert non-string values to string
	return fmt.Sprintf("%v", value), nil
}
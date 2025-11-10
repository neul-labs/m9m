package variables

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"time"
)

// VariableManager manages variables and environments
type VariableManager struct {
	storage       VariableStorage
	encryptionKey []byte // 32 bytes for AES-256
}

// NewVariableManager creates a new variable manager
func NewVariableManager(storage VariableStorage, encryptionKey string) *VariableManager {
	// Use provided key or generate a default (in production, always provide a key!)
	key := []byte(encryptionKey)
	if len(key) == 0 {
		key = []byte("default-encryption-key-change-me") // 32 bytes
	}
	// Ensure key is 32 bytes for AES-256
	if len(key) < 32 {
		// Pad the key
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	} else if len(key) > 32 {
		// Truncate the key
		key = key[:32]
	}

	return &VariableManager{
		storage:       storage,
		encryptionKey: key,
	}
}

// CreateVariable creates a new variable
func (m *VariableManager) CreateVariable(request *VariableCreateRequest) (*Variable, error) {
	if request.Key == "" {
		return nil, fmt.Errorf("variable key is required")
	}

	if request.Value == "" {
		return nil, fmt.Errorf("variable value is required")
	}

	// Check if variable with same key already exists
	_, err := m.storage.GetVariableByKey(request.Key, request.Type)
	if err == nil {
		return nil, fmt.Errorf("variable with key '%s' already exists", request.Key)
	}

	// Encrypt value if requested
	value := request.Value
	if request.Encrypted {
		encryptedValue, err := m.encrypt(request.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt variable value: %w", err)
		}
		value = encryptedValue
	}

	variable := &Variable{
		ID:          generateID("var"),
		Key:         request.Key,
		Value:       value,
		Type:        request.Type,
		Description: request.Description,
		Encrypted:   request.Encrypted,
		Tags:        request.Tags,
		Metadata:    request.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := m.storage.SaveVariable(variable); err != nil {
		return nil, fmt.Errorf("failed to save variable: %w", err)
	}

	log.Printf("✅ Variable created: %s (key=%s, type=%s, encrypted=%v)",
		variable.ID, variable.Key, variable.Type, variable.Encrypted)

	return variable, nil
}

// GetVariable retrieves a variable by ID
func (m *VariableManager) GetVariable(id string, decrypt bool) (*Variable, error) {
	variable, err := m.storage.GetVariable(id)
	if err != nil {
		return nil, err
	}

	// Decrypt if requested and variable is encrypted
	if decrypt && variable.Encrypted {
		decryptedValue, err := m.decrypt(variable.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt variable value: %w", err)
		}
		// Create a copy to avoid modifying the stored variable
		decryptedVar := *variable
		decryptedVar.Value = decryptedValue
		return &decryptedVar, nil
	}

	return variable, nil
}

// GetVariableByKey retrieves a variable by key
func (m *VariableManager) GetVariableByKey(key string, varType VariableType, decrypt bool) (*Variable, error) {
	variable, err := m.storage.GetVariableByKey(key, varType)
	if err != nil {
		return nil, err
	}

	// Decrypt if requested
	if decrypt && variable.Encrypted {
		decryptedValue, err := m.decrypt(variable.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt variable value: %w", err)
		}
		decryptedVar := *variable
		decryptedVar.Value = decryptedValue
		return &decryptedVar, nil
	}

	return variable, nil
}

// ListVariables lists all variables
func (m *VariableManager) ListVariables(filters VariableListFilters) ([]*Variable, int, error) {
	return m.storage.ListVariables(filters)
}

// UpdateVariable updates a variable
func (m *VariableManager) UpdateVariable(id string, request *VariableUpdateRequest) (*Variable, error) {
	variable, err := m.storage.GetVariable(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if request.Value != nil {
		value := *request.Value
		if variable.Encrypted {
			encryptedValue, err := m.encrypt(value)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt variable value: %w", err)
			}
			variable.Value = encryptedValue
		} else {
			variable.Value = value
		}
	}

	if request.Description != nil {
		variable.Description = *request.Description
	}

	if request.Tags != nil {
		variable.Tags = request.Tags
	}

	if request.Metadata != nil {
		variable.Metadata = request.Metadata
	}

	variable.UpdatedAt = time.Now()

	if err := m.storage.SaveVariable(variable); err != nil {
		return nil, fmt.Errorf("failed to update variable: %w", err)
	}

	log.Printf("✅ Variable updated: %s (key=%s)", variable.ID, variable.Key)

	return variable, nil
}

// DeleteVariable deletes a variable
func (m *VariableManager) DeleteVariable(id string) error {
	return m.storage.DeleteVariable(id)
}

// CreateEnvironment creates a new environment
func (m *VariableManager) CreateEnvironment(request *EnvironmentCreateRequest) (*Environment, error) {
	if request.Key == "" {
		return nil, fmt.Errorf("environment key is required")
	}

	if request.Name == "" {
		return nil, fmt.Errorf("environment name is required")
	}

	// Check if environment with same key already exists
	_, err := m.storage.GetEnvironmentByKey(request.Key)
	if err == nil {
		return nil, fmt.Errorf("environment with key '%s' already exists", request.Key)
	}

	environment := &Environment{
		ID:          generateID("env"),
		Name:        request.Name,
		Key:         request.Key,
		Description: request.Description,
		Variables:   request.Variables,
		Active:      request.Active,
		Metadata:    request.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if environment.Variables == nil {
		environment.Variables = make(map[string]string)
	}

	if err := m.storage.SaveEnvironment(environment); err != nil {
		return nil, fmt.Errorf("failed to save environment: %w", err)
	}

	log.Printf("✅ Environment created: %s (key=%s, active=%v)",
		environment.ID, environment.Key, environment.Active)

	return environment, nil
}

// GetEnvironment retrieves an environment by ID
func (m *VariableManager) GetEnvironment(id string) (*Environment, error) {
	return m.storage.GetEnvironment(id)
}

// ListEnvironments lists all environments
func (m *VariableManager) ListEnvironments() ([]*Environment, error) {
	return m.storage.ListEnvironments()
}

// UpdateEnvironment updates an environment
func (m *VariableManager) UpdateEnvironment(id string, request *EnvironmentUpdateRequest) (*Environment, error) {
	environment, err := m.storage.GetEnvironment(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if request.Name != nil {
		environment.Name = *request.Name
	}

	if request.Description != nil {
		environment.Description = *request.Description
	}

	if request.Variables != nil {
		environment.Variables = request.Variables
	}

	if request.Active != nil {
		environment.Active = *request.Active
	}

	if request.Metadata != nil {
		environment.Metadata = request.Metadata
	}

	environment.UpdatedAt = time.Now()

	if err := m.storage.SaveEnvironment(environment); err != nil {
		return nil, fmt.Errorf("failed to update environment: %w", err)
	}

	log.Printf("✅ Environment updated: %s (key=%s)", environment.ID, environment.Key)

	return environment, nil
}

// DeleteEnvironment deletes an environment
func (m *VariableManager) DeleteEnvironment(id string) error {
	return m.storage.DeleteEnvironment(id)
}

// GetVariableContext gets all variables for a workflow execution
func (m *VariableManager) GetVariableContext(workflowID string, environmentKey string) (*VariableContext, error) {
	context := &VariableContext{
		Global:      make(map[string]string),
		Environment: make(map[string]string),
		Workflow:    make(map[string]string),
	}

	// Get global variables
	globalVars, _, err := m.storage.ListVariables(VariableListFilters{
		Type: GlobalVariable,
	})
	if err == nil {
		for _, v := range globalVars {
			value := v.Value
			// Decrypt if encrypted
			if v.Encrypted {
				decryptedValue, err := m.decrypt(v.Value)
				if err == nil {
					value = decryptedValue
				}
			}
			context.Global[v.Key] = value
		}
	}

	// Get environment variables
	if environmentKey == "" {
		// Get active environment
		activeEnv, err := m.storage.GetActiveEnvironment()
		if err == nil {
			context.Environment = activeEnv.Variables
		}
	} else {
		// Get specific environment
		env, err := m.storage.GetEnvironmentByKey(environmentKey)
		if err == nil {
			context.Environment = env.Variables
		}
	}

	// Get workflow-specific variables
	workflowVars, err := m.storage.GetWorkflowVariables(workflowID)
	if err == nil {
		context.Workflow = workflowVars
	}

	return context, nil
}

// SaveWorkflowVariables saves variables for a workflow
func (m *VariableManager) SaveWorkflowVariables(workflowID string, variables map[string]string) error {
	return m.storage.SaveWorkflowVariables(workflowID, variables)
}

// GetWorkflowVariables gets variables for a workflow
func (m *VariableManager) GetWorkflowVariables(workflowID string) (map[string]string, error) {
	return m.storage.GetWorkflowVariables(workflowID)
}

// encrypt encrypts a string value using AES-256-GCM
func (m *VariableManager) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a string value using AES-256-GCM
func (m *VariableManager) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], string(data[nonceSize:])
	plaintext, err := gcm.Open(nil, nonce, []byte(ciphertext), nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// generateID generates a unique ID
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

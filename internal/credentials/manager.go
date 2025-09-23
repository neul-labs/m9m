package credentials

import (
	"fmt"
	"sync"

	"github.com/yourusername/n8n-go/internal/model"
)

// CredentialManager manages credentials for workflow nodes
type CredentialManager struct {
	store        *CredentialStore
	nodeMappings map[string]map[string]string // nodeID -> paramName -> credentialID
	mu           sync.RWMutex
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager() (*CredentialManager, error) {
	store, err := NewCredentialStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create credential store: %w", err)
	}

	return &CredentialManager{
		store:        store,
		nodeMappings: make(map[string]map[string]string),
	}, nil
}

// RegisterNodeCredentials registers a credential for a specific node parameter
func (cm *CredentialManager) RegisterNodeCredentials(nodeID, paramName, credentialID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.nodeMappings[nodeID] == nil {
		cm.nodeMappings[nodeID] = make(map[string]string)
	}
	cm.nodeMappings[nodeID][paramName] = credentialID
}

// StoreCredentialFromWorkflow stores a credential from workflow data
func (cm *CredentialManager) StoreCredentialFromWorkflow(credData map[string]interface{}) error {
	// Extract credential information
	id, ok := credData["id"].(string)
	if !ok {
		return fmt.Errorf("credential id is required")
	}

	name, ok := credData["name"].(string)
	if !ok {
		return fmt.Errorf("credential name is required")
	}

	credType, ok := credData["type"].(string)
	if !ok {
		return fmt.Errorf("credential type is required")
	}

	// Extract data fields
	data := make(map[string]string)
	if dataField, exists := credData["data"]; exists {
		if dataMap, ok := dataField.(map[string]interface{}); ok {
			for k, v := range dataMap {
				if str, ok := v.(string); ok {
					data[k] = str
				}
			}
		}
	}

	// Convert data to map[string]interface{}
	dataIntf := make(map[string]interface{})
	for k, v := range data {
		dataIntf[k] = v
	}

	credential := &Credential{
		ID:   id,
		Name: name,
		Type: credType,
		Data: dataIntf,
	}

	return cm.store.StoreCredential(credential)
}

// ResolveWorkflowCredentials resolves credentials for all nodes in a workflow
func (cm *CredentialManager) ResolveWorkflowCredentials(workflow *model.Workflow) error {
	for _, node := range workflow.Nodes {
		if node.Credentials != nil && len(node.Credentials) > 0 {
			for credType, credRef := range node.Credentials {
				if credRef.ID != "" {
					cm.RegisterNodeCredentials(node.ID, credType, credRef.ID)
				}
			}
		}
	}
	return nil
}

// InjectCredentialsIntoNodeParameters injects resolved credentials into node parameters
func (cm *CredentialManager) InjectCredentialsIntoNodeParameters(nodeID string, parameters map[string]interface{}) (map[string]interface{}, error) {
	creds, err := cm.GetNodeCredentials(nodeID)
	if err != nil {
		return nil, err
	}

	// Create a copy of parameters and inject credentials
	result := make(map[string]interface{})
	for k, v := range parameters {
		result[k] = v
	}

	// Inject credential values
	for key, value := range creds {
		result[key] = value
	}

	return result, nil
}

// GetNodeCredentials retrieves credentials for a specific node
func (cm *CredentialManager) GetNodeCredentials(nodeID string) (map[string]string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	// Get the credential mappings for this node
	mappings, exists := cm.nodeMappings[nodeID]
	if !exists {
		// No credentials registered for this node
		return make(map[string]string), nil
	}
	
	// Resolve each credential
	resolvedCredentials := make(map[string]string)
	
	for paramName, credentialID := range mappings {
		cred, err := cm.store.GetCredential(credentialID)
		if err != nil {
			// If credential not found, continue with empty value
			// This allows for graceful handling of missing credentials
			resolvedCredentials[paramName] = ""
			continue
		}
		
		// Resolve credential values, handling environment variables
		for key, value := range cred.Data {
			resolvedValue, err := cm.store.ResolveCredentialValue(value)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve credential value for %s.%s: %v", paramName, key, err)
			}
			
			// Store with prefixed key to avoid conflicts
			resolvedCredentials[fmt.Sprintf("%s_%s", paramName, key)] = resolvedValue
		}
	}
	
	return resolvedCredentials, nil
}
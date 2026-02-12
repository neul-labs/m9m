package m9m

import (
	"github.com/neul-labs/m9m/internal/credentials"
)

// CredentialManager manages credentials for workflow nodes.
// It provides secure storage and retrieval of sensitive data like API keys,
// passwords, and tokens.
type CredentialManager struct {
	internal *credentials.CredentialManager
}

// RegisterNodeCredentials registers a credential for a specific node parameter.
// This associates a credential ID with a node's parameter for later resolution.
func (cm *CredentialManager) RegisterNodeCredentials(nodeID, paramName, credentialID string) {
	cm.internal.RegisterNodeCredentials(nodeID, paramName, credentialID)
}

// StoreCredential stores a credential from workflow data.
// The credData map should contain:
// - "id": the credential ID
// - "name": the credential name
// - "type": the credential type
// - "data": a map of credential values
func (cm *CredentialManager) StoreCredential(credData map[string]interface{}) error {
	return cm.internal.StoreCredentialFromWorkflow(credData)
}

// GetNodeCredentials retrieves resolved credentials for a specific node.
// Returns a map of parameter names to their resolved values.
func (cm *CredentialManager) GetNodeCredentials(nodeID string) (map[string]string, error) {
	return cm.internal.GetNodeCredentials(nodeID)
}

// ResolveWorkflowCredentials resolves credentials for all nodes in a workflow.
// This should be called before executing a workflow to ensure all credential
// references are properly registered.
func (cm *CredentialManager) ResolveWorkflowCredentials(workflow *Workflow) error {
	if workflow == nil {
		return ErrNilWorkflow
	}
	return cm.internal.ResolveWorkflowCredentials(workflow.toInternal())
}

// InjectCredentialsIntoParameters injects resolved credentials into node parameters.
// This is called automatically during workflow execution but can be used manually
// for custom execution flows.
func (cm *CredentialManager) InjectCredentialsIntoParameters(nodeID string, params map[string]interface{}) (map[string]interface{}, error) {
	return cm.internal.InjectCredentialsIntoNodeParameters(nodeID, params)
}

// CredentialData represents credential data for storage.
type CredentialData struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// ToMap converts CredentialData to a map for storage.
func (c *CredentialData) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":   c.ID,
		"name": c.Name,
		"type": c.Type,
		"data": c.Data,
	}
}

// StoreCredentialData is a convenience method for storing a CredentialData struct.
func (cm *CredentialManager) StoreCredentialData(cred *CredentialData) error {
	return cm.StoreCredential(cred.ToMap())
}

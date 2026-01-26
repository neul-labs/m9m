/*
Package transform provides data transformation node implementations for n8n-go.
*/
package transform

import (
	"math"
	
	"github.com/neul-labs/m9m/internal/model"
	"github.com/neul-labs/m9m/internal/nodes/base"
)

// SplitInBatchesNode implements the Split In Batches node functionality
type SplitInBatchesNode struct {
	*base.BaseNode
}

// NewSplitInBatchesNode creates a new Split In Batches node
func NewSplitInBatchesNode() *SplitInBatchesNode {
	description := base.NodeDescription{
		Name:        "Split In Batches",
		Description: "Splits items into batches of specified size",
		Category:    "Data Transformation",
	}
	
	return &SplitInBatchesNode{
		BaseNode: base.NewBaseNode(description),
	}
}

// Description returns the node description
func (s *SplitInBatchesNode) Description() base.NodeDescription {
	return s.BaseNode.Description()
}

// ValidateParameters validates Split In Batches node parameters
func (s *SplitInBatchesNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return s.CreateError("parameters cannot be nil", nil)
	}
	
	// Check if batchSize exists
	batchSize, ok := params["batchSize"]
	if !ok {
		return s.CreateError("batchSize parameter is required", nil)
	}
	
	// Check if batchSize is a number
	batchSizeFloat, ok := batchSize.(float64)
	if !ok {
		// Also check if it's an int (which would be converted to float64 in JSON)
		if batchSizeInt, ok := batchSize.(int); ok {
			batchSizeFloat = float64(batchSizeInt)
		} else {
			return s.CreateError("batchSize must be a number", nil)
		}
	}
	
	// Check if batchSize is positive (0 is not allowed)
	if batchSizeFloat <= 0 {
		return s.CreateError("batchSize must be positive", nil)
	}
	
	// Check if options exist (optional)
	options, ok := params["options"]
	if ok {
		// Check if options is a map
		optionsMap, ok := options.(map[string]interface{})
		if !ok {
			return s.CreateError("options must be an object", nil)
		}
		
		// Validate reset option
		if reset, ok := optionsMap["reset"]; ok {
			if _, ok := reset.(bool); !ok {
				return s.CreateError("reset option must be a boolean", nil)
			}
		}
	}
	
	return nil
}

// Execute processes the Split In Batches node operation
func (s *SplitInBatchesNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	if len(inputData) == 0 {
		return []model.DataItem{}, nil
	}
	
	// Get batchSize from node parameters
	batchSize := s.GetIntParameter(nodeParams, "batchSize", 10)
	if batchSize <= 0 {
		// If batchSize is 0 or negative, default to 10
		batchSize = 10
	}
	
	// Get options from node parameters
	options := s.GetMapParameter(nodeParams, "options", make(map[string]interface{}))
	reset := s.GetBoolParameter(options, "reset", false)
	
	// Split data into batches
	batches := s.splitIntoBatches(inputData, batchSize, reset)
	
	// Return the first batch as output
	if len(batches) > 0 {
		return batches[0], nil
	}
	
	return []model.DataItem{}, nil
}

// splitIntoBatches splits data items into batches of specified size
func (s *SplitInBatchesNode) splitIntoBatches(data []model.DataItem, batchSize int, reset bool) [][]model.DataItem {
	if len(data) == 0 {
		return [][]model.DataItem{}
	}
	
	if batchSize <= 0 {
		batchSize = 10 // Default batch size
	}
	
	// Calculate number of batches needed
	numBatches := int(math.Ceil(float64(len(data)) / float64(batchSize)))
	
	// Create batches
	batches := make([][]model.DataItem, numBatches)
	
	for i := 0; i < numBatches; i++ {
		start := i * batchSize
		end := start + batchSize
		if end > len(data) {
			end = len(data)
		}
		
		// Create batch with proper size
		batch := make([]model.DataItem, end-start)
		copy(batch, data[start:end])
		
		batches[i] = batch
	}
	
	return batches
}

// GetMapParameter retrieves a map parameter with a default fallback
func (s *SplitInBatchesNode) GetMapParameter(params map[string]interface{}, name string, defaultValue map[string]interface{}) map[string]interface{} {
	if params == nil {
		return defaultValue
	}
	
	if value, exists := params[name]; exists {
		if mapValue, ok := value.(map[string]interface{}); ok {
			return mapValue
		}
	}
	
	return defaultValue
}

// GetBoolParameter retrieves a boolean parameter with a default fallback
func (s *SplitInBatchesNode) GetBoolParameter(params map[string]interface{}, name string, defaultValue bool) bool {
	if params == nil {
		return defaultValue
	}
	
	if value, exists := params[name]; exists {
		if boolValue, ok := value.(bool); ok {
			return boolValue
		}
	}
	
	return defaultValue
}

// GetIntParameter retrieves an integer parameter with a default fallback
func (s *SplitInBatchesNode) GetIntParameter(params map[string]interface{}, name string, defaultValue int) int {
	if params == nil {
		return defaultValue
	}
	
	if value, exists := params[name]; exists {
		if num, ok := value.(int); ok {
			return num
		}
		if floatVal, ok := value.(float64); ok {
			return int(floatVal)
		}
	}
	
	return defaultValue
}
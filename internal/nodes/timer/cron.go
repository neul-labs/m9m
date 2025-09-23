/*
Package timer provides timer and trigger node implementations for n8n-go.
*/
package timer

import (
	"time"
	
	"github.com/yourusername/n8n-go/internal/model"
	"github.com/yourusername/n8n-go/internal/nodes/base"
)

// CronNode implements the Cron node functionality
type CronNode struct {
	*base.BaseNode
}

// NewCronNode creates a new Cron node
func NewCronNode() *CronNode {
	description := base.NodeDescription{
		Name:        "Cron",
		Description: "Triggers workflows on a schedule using cron expressions",
		Category:    "Triggers",
	}
	
	return &CronNode{
		BaseNode: base.NewBaseNode(description),
	}
}

// Description returns the node description
func (c *CronNode) Description() base.NodeDescription {
	return c.BaseNode.Description()
}

// ValidateParameters validates Cron node parameters
func (c *CronNode) ValidateParameters(params map[string]interface{}) error {
	if params == nil {
		return c.CreateError("parameters cannot be nil", nil)
	}
	
	// Check if cron expression is provided
	cronExpression := c.GetStringParameter(params, "cronExpression", "")
	if cronExpression == "" {
		return c.CreateError("cronExpression is required", nil)
	}
	
	return nil
}

// Execute processes the Cron node operation
func (c *CronNode) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// For a cron node, we typically don't process input data
	// Instead, we generate a trigger event
	
	// Get cron expression from parameters
	cronExpression := c.GetStringParameter(nodeParams, "cronExpression", "")
	
	// Create a trigger event
	triggerData := model.DataItem{
		JSON: map[string]interface{}{
			"cronExpression": cronExpression,
			"triggeredAt":    time.Now().UTC().Format(time.RFC3339),
			"trigger":        true,
		},
	}
	
	return []model.DataItem{triggerData}, nil
}

// ParseCronExpression parses a cron expression and returns the next execution time
// This is a simplified implementation - a full implementation would parse the cron expression properly
func (c *CronNode) ParseCronExpression(cronExpression string) (time.Time, error) {
	// For simplicity, we'll just return the next minute
	// A real implementation would parse the cron expression and calculate the next execution time
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute()+1, 0, 0, now.Location())
	return next, nil
}
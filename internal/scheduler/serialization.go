package scheduler

import (
	"encoding/json"
	"fmt"
	"time"
)

// generateScheduleID generates a unique schedule ID
func generateScheduleID() string {
	return fmt.Sprintf("sched_%d", time.Now().UnixNano())
}

// generateExecutionID generates a unique execution ID
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

// ToJSON converts schedule config to JSON
func (c *ScheduleConfig) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}

// FromJSON creates schedule config from JSON
func (c *ScheduleConfig) FromJSON(data []byte) error {
	return json.Unmarshal(data, c)
}

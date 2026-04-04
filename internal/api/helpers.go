package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/neul-labs/m9m/internal/model"
)

func (s *APIServer) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *APIServer) sendError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]interface{}{
		"error":   true,
		"message": message,
		"code":    status,
	}

	if err != nil && s.config.DevMode {
		response["details"] = err.Error()
	}

	s.sendJSON(w, status, response)
}

func (s *APIServer) trackExecutionCancel(executionID string, cancel context.CancelFunc) {
	s.executionMu.Lock()
	defer s.executionMu.Unlock()
	s.executionCancels[executionID] = cancel
}

func (s *APIServer) untrackExecutionCancel(executionID string) {
	s.executionMu.Lock()
	defer s.executionMu.Unlock()
	delete(s.executionCancels, executionID)
}

func (s *APIServer) getExecutionCancel(executionID string) (context.CancelFunc, bool) {
	s.executionMu.RLock()
	defer s.executionMu.RUnlock()
	cancel, exists := s.executionCancels[executionID]
	return cancel, exists
}

// parseIntParam safely parses an integer parameter with default and max values.
func parseIntParam(value string, defaultVal, maxVal int) int {
	if value == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return defaultVal
	}
	if maxVal > 0 && parsed > maxVal {
		return maxVal
	}
	return parsed
}

func defaultExecutionInputData() []model.DataItem {
	return []model.DataItem{{JSON: make(map[string]interface{})}}
}

func parseExecutionInputData(r *http.Request) ([]model.DataItem, error) {
	if r.Body == nil {
		return defaultExecutionInputData(), nil
	}

	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		if errors.Is(err, io.EOF) {
			return defaultExecutionInputData(), nil
		}
		return nil, err
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return defaultExecutionInputData(), nil
	}

	var inputData []model.DataItem
	switch trimmed[0] {
	case '[':
		if err := json.Unmarshal(trimmed, &inputData); err != nil {
			return nil, err
		}
	case '{':
		var envelope struct {
			InputData []model.DataItem `json:"inputData"`
		}
		if err := json.Unmarshal(trimmed, &envelope); err != nil {
			return nil, err
		}
		inputData = envelope.InputData
	default:
		return nil, fmt.Errorf("expected request body to be an array of data items or an object with inputData")
	}

	if len(inputData) == 0 {
		return defaultExecutionInputData(), nil
	}

	return inputData, nil
}

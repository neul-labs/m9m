package main

/*
#include <stdlib.h>
#include <stdint.h>

// Error structure for returning errors to C
typedef struct {
	int code;
	char* message;
} m9m_error_t;

// Callback function type for custom nodes
typedef char* (*m9m_node_execute_fn)(
	void* user_data,
	const char* input_json,
	int input_len,
	const char* params_json,
	int params_len,
	m9m_error_t* err
);

// Node callback registration struct
typedef struct {
	m9m_node_execute_fn execute_fn;
	void* user_data;
} m9m_node_callback_t;

// Trampoline function to call the C callback from Go
// This is defined in C to properly handle the function pointer call
static inline char* call_node_execute(
	m9m_node_execute_fn fn,
	void* user_data,
	const char* input_json,
	int input_len,
	const char* params_json,
	int params_len,
	m9m_error_t* err
) {
	return fn(user_data, input_json, input_len, params_json, params_len, err);
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/dipankar/m9m/pkg/m9m"
)

// callbackNode is a node implementation that calls back to C/Python/Node.js
type callbackNode struct {
	*m9m.BaseNode
	callback *C.m9m_node_callback_t
}

// Execute calls the C callback function with JSON-encoded input and params
func (n *callbackNode) Execute(inputData []m9m.DataItem, nodeParams map[string]interface{}) ([]m9m.DataItem, error) {
	if n.callback == nil || n.callback.execute_fn == nil {
		return inputData, nil
	}

	// Serialize input data to JSON
	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize input: %w", err)
	}

	// Serialize parameters to JSON
	paramsJSON, err := json.Marshal(nodeParams)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize params: %w", err)
	}

	// Prepare C strings
	cInputJSON := C.CString(string(inputJSON))
	defer C.free(unsafe.Pointer(cInputJSON))

	cParamsJSON := C.CString(string(paramsJSON))
	defer C.free(unsafe.Pointer(cParamsJSON))

	// Prepare error struct
	var cErr C.m9m_error_t

	// Call the C callback via trampoline
	resultPtr := C.call_node_execute(
		n.callback.execute_fn,
		n.callback.user_data,
		cInputJSON,
		C.int(len(inputJSON)),
		cParamsJSON,
		C.int(len(paramsJSON)),
		&cErr,
	)

	// Check for errors from callback
	if cErr.code != 0 {
		errMsg := "callback error"
		if cErr.message != nil {
			errMsg = C.GoString(cErr.message)
			C.free(unsafe.Pointer(cErr.message))
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	// Parse result
	if resultPtr == nil {
		return inputData, nil
	}
	defer C.free(unsafe.Pointer(resultPtr))

	resultJSON := C.GoString(resultPtr)
	var result []m9m.DataItem
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to parse callback result: %w", err)
	}

	return result, nil
}

// callbackNodeRegistry stores callback nodes to prevent garbage collection
var callbackNodeRegistry = make(map[string]*callbackNode)

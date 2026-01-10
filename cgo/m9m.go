package main

/*
#include <stdlib.h>
#include <stdint.h>

// Error structure for returning errors to C
typedef struct {
	int code;
	char* message;
} m9m_error_t;

// Opaque handle types
typedef uint64_t m9m_engine_t;
typedef uint64_t m9m_workflow_t;
typedef uint64_t m9m_result_t;
typedef uint64_t m9m_credential_manager_t;

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
*/
import "C"

import (
	"encoding/json"
	"unsafe"

	"github.com/dipankar/m9m/pkg/m9m"
)

// Error codes
const (
	M9M_OK                = 0
	M9M_ERR_INVALID_ARG   = 1
	M9M_ERR_NULL_POINTER  = 2
	M9M_ERR_PARSE_ERROR   = 3
	M9M_ERR_EXECUTION     = 4
	M9M_ERR_NOT_FOUND     = 5
	M9M_ERR_INTERNAL      = 6
)

// setError sets an error on the error pointer if it's not nil
func setError(err *C.m9m_error_t, code int, message string) {
	if err != nil {
		err.code = C.int(code)
		err.message = C.CString(message)
	}
}

// clearError clears an error
func clearError(err *C.m9m_error_t) {
	if err != nil {
		err.code = C.int(M9M_OK)
		err.message = nil
	}
}

//export m9m_error_free
func m9m_error_free(err *C.m9m_error_t) {
	if err != nil && err.message != nil {
		C.free(unsafe.Pointer(err.message))
		err.message = nil
	}
}

//export m9m_free_string
func m9m_free_string(str *C.char) {
	if str != nil {
		C.free(unsafe.Pointer(str))
	}
}

// ============================================================================
// Engine Functions
// ============================================================================

//export m9m_engine_new
func m9m_engine_new(err *C.m9m_error_t) C.m9m_engine_t {
	clearError(err)
	engine := m9m.New()
	handle := NewHandle(engine)
	return C.m9m_engine_t(handle)
}

//export m9m_engine_new_with_credentials
func m9m_engine_new_with_credentials(credMgr C.m9m_credential_manager_t, err *C.m9m_error_t) C.m9m_engine_t {
	clearError(err)

	var cm *m9m.CredentialManager
	if credMgr != 0 {
		obj := Handle(credMgr).Get()
		if obj == nil {
			setError(err, M9M_ERR_NOT_FOUND, "credential manager handle not found")
			return 0
		}
		cm = obj.(*m9m.CredentialManager)
	}

	var engine *m9m.Engine
	if cm != nil {
		engine = m9m.NewWithOptions(m9m.WithCredentialManager(cm))
	} else {
		engine = m9m.New()
	}

	handle := NewHandle(engine)
	return C.m9m_engine_t(handle)
}

//export m9m_engine_free
func m9m_engine_free(engine C.m9m_engine_t) {
	if engine != 0 {
		Handle(engine).Delete()
	}
}

// ============================================================================
// Workflow Functions
// ============================================================================

//export m9m_workflow_from_json
func m9m_workflow_from_json(jsonData *C.char, jsonLen C.int, err *C.m9m_error_t) C.m9m_workflow_t {
	clearError(err)

	if jsonData == nil {
		setError(err, M9M_ERR_NULL_POINTER, "json data is null")
		return 0
	}

	data := C.GoBytes(unsafe.Pointer(jsonData), jsonLen)
	workflow, parseErr := m9m.ParseWorkflow(data)
	if parseErr != nil {
		setError(err, M9M_ERR_PARSE_ERROR, parseErr.Error())
		return 0
	}

	handle := NewHandle(workflow)
	return C.m9m_workflow_t(handle)
}

//export m9m_workflow_from_file
func m9m_workflow_from_file(path *C.char, err *C.m9m_error_t) C.m9m_workflow_t {
	clearError(err)

	if path == nil {
		setError(err, M9M_ERR_NULL_POINTER, "path is null")
		return 0
	}

	goPath := C.GoString(path)
	workflow, loadErr := m9m.LoadWorkflow(goPath)
	if loadErr != nil {
		setError(err, M9M_ERR_PARSE_ERROR, loadErr.Error())
		return 0
	}

	handle := NewHandle(workflow)
	return C.m9m_workflow_t(handle)
}

//export m9m_workflow_to_json
func m9m_workflow_to_json(workflow C.m9m_workflow_t, err *C.m9m_error_t) *C.char {
	clearError(err)

	if workflow == 0 {
		setError(err, M9M_ERR_NULL_POINTER, "workflow handle is null")
		return nil
	}

	obj := Handle(workflow).Get()
	if obj == nil {
		setError(err, M9M_ERR_NOT_FOUND, "workflow handle not found")
		return nil
	}

	wf := obj.(*m9m.Workflow)
	data, jsonErr := wf.ToJSON()
	if jsonErr != nil {
		setError(err, M9M_ERR_INTERNAL, jsonErr.Error())
		return nil
	}

	return C.CString(string(data))
}

//export m9m_workflow_free
func m9m_workflow_free(workflow C.m9m_workflow_t) {
	if workflow != 0 {
		Handle(workflow).Delete()
	}
}

// ============================================================================
// Execution Functions
// ============================================================================

//export m9m_engine_execute
func m9m_engine_execute(engine C.m9m_engine_t, workflow C.m9m_workflow_t, inputJSON *C.char, inputLen C.int, err *C.m9m_error_t) C.m9m_result_t {
	clearError(err)

	if engine == 0 {
		setError(err, M9M_ERR_NULL_POINTER, "engine handle is null")
		return 0
	}

	if workflow == 0 {
		setError(err, M9M_ERR_NULL_POINTER, "workflow handle is null")
		return 0
	}

	engineObj := Handle(engine).Get()
	if engineObj == nil {
		setError(err, M9M_ERR_NOT_FOUND, "engine handle not found")
		return 0
	}

	workflowObj := Handle(workflow).Get()
	if workflowObj == nil {
		setError(err, M9M_ERR_NOT_FOUND, "workflow handle not found")
		return 0
	}

	eng := engineObj.(*m9m.Engine)
	wf := workflowObj.(*m9m.Workflow)

	// Parse input data if provided
	var inputData []m9m.DataItem
	if inputJSON != nil && inputLen > 0 {
		data := C.GoBytes(unsafe.Pointer(inputJSON), inputLen)
		if parseErr := json.Unmarshal(data, &inputData); parseErr != nil {
			setError(err, M9M_ERR_PARSE_ERROR, "failed to parse input JSON: "+parseErr.Error())
			return 0
		}
	}

	// Execute the workflow
	result, execErr := eng.Execute(wf, inputData)
	if execErr != nil {
		setError(err, M9M_ERR_EXECUTION, execErr.Error())
		return 0
	}

	handle := NewHandle(result)
	return C.m9m_result_t(handle)
}

//export m9m_result_to_json
func m9m_result_to_json(result C.m9m_result_t, err *C.m9m_error_t) *C.char {
	clearError(err)

	if result == 0 {
		setError(err, M9M_ERR_NULL_POINTER, "result handle is null")
		return nil
	}

	obj := Handle(result).Get()
	if obj == nil {
		setError(err, M9M_ERR_NOT_FOUND, "result handle not found")
		return nil
	}

	res := obj.(*m9m.ExecutionResult)

	// Create a serializable struct
	output := struct {
		Data  []m9m.DataItem `json:"data"`
		Error string         `json:"error,omitempty"`
	}{
		Data: res.Data,
	}
	if res.Error != nil {
		output.Error = res.Error.Error()
	}

	data, jsonErr := json.Marshal(output)
	if jsonErr != nil {
		setError(err, M9M_ERR_INTERNAL, jsonErr.Error())
		return nil
	}

	return C.CString(string(data))
}

//export m9m_result_has_error
func m9m_result_has_error(result C.m9m_result_t) C.int {
	if result == 0 {
		return 0
	}

	obj := Handle(result).Get()
	if obj == nil {
		return 0
	}

	res := obj.(*m9m.ExecutionResult)
	if res.Error != nil {
		return 1
	}
	return 0
}

//export m9m_result_get_error
func m9m_result_get_error(result C.m9m_result_t) *C.char {
	if result == 0 {
		return nil
	}

	obj := Handle(result).Get()
	if obj == nil {
		return nil
	}

	res := obj.(*m9m.ExecutionResult)
	if res.Error != nil {
		return C.CString(res.Error.Error())
	}
	return nil
}

//export m9m_result_free
func m9m_result_free(result C.m9m_result_t) {
	if result != 0 {
		Handle(result).Delete()
	}
}

// ============================================================================
// Credential Manager Functions
// ============================================================================

//export m9m_credential_manager_new
func m9m_credential_manager_new(err *C.m9m_error_t) C.m9m_credential_manager_t {
	clearError(err)

	cm, cmErr := m9m.NewCredentialManager()
	if cmErr != nil {
		setError(err, M9M_ERR_INTERNAL, cmErr.Error())
		return 0
	}

	handle := NewHandle(cm)
	return C.m9m_credential_manager_t(handle)
}

//export m9m_credential_manager_free
func m9m_credential_manager_free(credMgr C.m9m_credential_manager_t) {
	if credMgr != 0 {
		Handle(credMgr).Delete()
	}
}

//export m9m_credential_manager_store
func m9m_credential_manager_store(credMgr C.m9m_credential_manager_t, credJSON *C.char, credLen C.int, err *C.m9m_error_t) C.int {
	clearError(err)

	if credMgr == 0 {
		setError(err, M9M_ERR_NULL_POINTER, "credential manager handle is null")
		return 0
	}

	if credJSON == nil {
		setError(err, M9M_ERR_NULL_POINTER, "credential JSON is null")
		return 0
	}

	obj := Handle(credMgr).Get()
	if obj == nil {
		setError(err, M9M_ERR_NOT_FOUND, "credential manager handle not found")
		return 0
	}

	cm := obj.(*m9m.CredentialManager)

	data := C.GoBytes(unsafe.Pointer(credJSON), credLen)
	var credData map[string]interface{}
	if parseErr := json.Unmarshal(data, &credData); parseErr != nil {
		setError(err, M9M_ERR_PARSE_ERROR, "failed to parse credential JSON: "+parseErr.Error())
		return 0
	}

	if storeErr := cm.StoreCredential(credData); storeErr != nil {
		setError(err, M9M_ERR_INTERNAL, storeErr.Error())
		return 0
	}

	return 1
}

// ============================================================================
// Node Registration Functions
// ============================================================================

//export m9m_engine_register_node
func m9m_engine_register_node(engine C.m9m_engine_t, nodeType *C.char, callback *C.m9m_node_callback_t, err *C.m9m_error_t) C.int {
	clearError(err)

	if engine == 0 {
		setError(err, M9M_ERR_NULL_POINTER, "engine handle is null")
		return 0
	}

	if nodeType == nil {
		setError(err, M9M_ERR_NULL_POINTER, "node type is null")
		return 0
	}

	if callback == nil {
		setError(err, M9M_ERR_NULL_POINTER, "callback is null")
		return 0
	}

	obj := Handle(engine).Get()
	if obj == nil {
		setError(err, M9M_ERR_NOT_FOUND, "engine handle not found")
		return 0
	}

	eng := obj.(*m9m.Engine)
	goNodeType := C.GoString(nodeType)

	// Create a custom node that calls back to C
	node := &callbackNode{
		BaseNode: m9m.NewBaseNode(m9m.NodeDescription{
			Name:        goNodeType,
			Description: "Custom callback node",
			Category:    "custom",
		}),
		callback: callback,
	}

	eng.RegisterNode(goNodeType, node)
	return 1
}

// ============================================================================
// Version Information
// ============================================================================

//export m9m_version
func m9m_version() *C.char {
	return C.CString("1.0.0")
}

// main is required for CGO but does nothing
func main() {}

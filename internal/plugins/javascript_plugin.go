package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dop251/goja"
	"github.com/dipankar/m9m/internal/model"
	"github.com/dipankar/m9m/internal/nodes/base"
)

// JavaScriptNodePlugin represents a node implemented in JavaScript
type JavaScriptNodePlugin struct {
	FilePath    string
	Name        string
	Description base.NodeDescription
	vm          *goja.Runtime
	executeFunc goja.Callable
	validateFunc goja.Callable
}

// JavaScriptPluginConfig holds configuration for the JavaScript runtime
type JavaScriptPluginConfig struct {
	MaxExecutionTime time.Duration
	EnableConsole    bool
	AllowNetworkAccess bool
}

// DefaultJavaScriptPluginConfig returns default configuration
func DefaultJavaScriptPluginConfig() *JavaScriptPluginConfig {
	return &JavaScriptPluginConfig{
		MaxExecutionTime: 30 * time.Second,
		EnableConsole:    true,
		AllowNetworkAccess: true,
	}
}

// LoadJavaScriptPlugin loads a JavaScript plugin from a file
func LoadJavaScriptPlugin(filePath string, config *JavaScriptPluginConfig) (*JavaScriptNodePlugin, error) {
	if config == nil {
		config = DefaultJavaScriptPluginConfig()
	}

	// Read JavaScript file
	code, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin file: %w", err)
	}

	// Create JavaScript VM
	vm := goja.New()

	// Setup console if enabled
	if config.EnableConsole {
		setupConsole(vm)
	}

	// Setup module.exports pattern
	vm.Set("module", vm.NewObject())
	module := vm.Get("module").ToObject(vm)
	module.Set("exports", vm.NewObject())

	// Execute plugin code
	_, err = vm.RunString(string(code))
	if err != nil {
		return nil, fmt.Errorf("failed to execute plugin code: %w", err)
	}

	// Get exports
	exports := module.Get("exports")
	if exports == nil || goja.IsUndefined(exports) {
		return nil, fmt.Errorf("plugin must export module.exports")
	}

	exportsObj := exports.ToObject(vm)

	// Extract description
	descriptionVal := exportsObj.Get("description")
	if descriptionVal == nil || goja.IsUndefined(descriptionVal) {
		return nil, fmt.Errorf("plugin must export 'description' object")
	}

	descriptionObj := descriptionVal.ToObject(vm)
	description := base.NodeDescription{
		Name:        getStringProperty(descriptionObj, "name", "Unknown"),
		Description: getStringProperty(descriptionObj, "description", ""),
		Category:    getStringProperty(descriptionObj, "category", "custom"),
	}

	// Extract execute function
	executeVal := exportsObj.Get("execute")
	if executeVal == nil || goja.IsUndefined(executeVal) {
		return nil, fmt.Errorf("plugin must export 'execute' function")
	}

	executeFunc, ok := goja.AssertFunction(executeVal)
	if !ok {
		return nil, fmt.Errorf("'execute' must be a function")
	}

	// Extract optional validate function
	var validateFunc goja.Callable
	validateVal := exportsObj.Get("validateParameters")
	if validateVal != nil && !goja.IsUndefined(validateVal) {
		validateFunc, _ = goja.AssertFunction(validateVal)
	}

	plugin := &JavaScriptNodePlugin{
		FilePath:    filePath,
		Name:        description.Name,
		Description: description,
		vm:          vm,
		executeFunc: executeFunc,
		validateFunc: validateFunc,
	}

	return plugin, nil
}

// Execute runs the plugin's execute function
func (p *JavaScriptNodePlugin) Execute(inputData []model.DataItem, nodeParams map[string]interface{}) ([]model.DataItem, error) {
	// Convert input data to JavaScript-friendly format
	jsInputData := convertToJSArray(p.vm, inputData)
	jsParams := p.vm.ToValue(nodeParams)

	// Call execute function
	result, err := p.executeFunc(goja.Undefined(), jsInputData, jsParams)
	if err != nil {
		return nil, fmt.Errorf("plugin execution error: %w", err)
	}

	// Convert result back to []model.DataItem
	outputData, err := convertFromJSArray(result)
	if err != nil {
		return nil, fmt.Errorf("failed to convert plugin result: %w", err)
	}

	return outputData, nil
}

// ValidateParameters validates node parameters using the plugin's validate function
func (p *JavaScriptNodePlugin) ValidateParameters(params map[string]interface{}) error {
	if p.validateFunc == nil {
		// No validation function provided
		return nil
	}

	jsParams := p.vm.ToValue(params)

	_, err := p.validateFunc(goja.Undefined(), jsParams)
	if err != nil {
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	return nil
}

// GetDescription returns the node description
func (p *JavaScriptNodePlugin) GetDescription() base.NodeDescription {
	return p.Description
}

// setupConsole adds console.log, console.error, etc. to the VM
func setupConsole(vm *goja.Runtime) {
	console := vm.NewObject()

	console.Set("log", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		fmt.Println(args...)
		return goja.Undefined()
	})

	console.Set("error", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		fmt.Fprintln(os.Stderr, args...)
		return goja.Undefined()
	})

	console.Set("warn", func(call goja.FunctionCall) goja.Value {
		args := make([]interface{}, len(call.Arguments))
		for i, arg := range call.Arguments {
			args[i] = arg.Export()
		}
		allArgs := append([]interface{}{"[WARN]"}, args...)
		fmt.Println(allArgs...)
		return goja.Undefined()
	})

	vm.Set("console", console)
}

// getStringProperty safely extracts a string property from a Goja object
func getStringProperty(obj *goja.Object, key string, defaultValue string) string {
	val := obj.Get(key)
	if val == nil || goja.IsUndefined(val) {
		return defaultValue
	}
	return val.String()
}

// convertToJSArray converts []model.DataItem to a JavaScript array
func convertToJSArray(vm *goja.Runtime, items []model.DataItem) goja.Value {
	arr := vm.NewArray()

	for i, item := range items {
		jsItem := vm.NewObject()

		// Add JSON data
		if item.JSON != nil {
			jsItem.Set("json", vm.ToValue(item.JSON))
		} else {
			jsItem.Set("json", vm.NewObject())
		}

		// Add binary data if present
		if item.Binary != nil {
			jsBinary := vm.NewObject()
			for key, binaryData := range item.Binary {
				binaryObj := vm.NewObject()
				binaryObj.Set("data", vm.ToValue(binaryData.Data))
				binaryObj.Set("mimeType", vm.ToValue(binaryData.MimeType))
				binaryObj.Set("fileName", vm.ToValue(binaryData.FileName))
				binaryObj.Set("fileExtension", vm.ToValue(binaryData.FileExtension))
				jsBinary.Set(key, binaryObj)
			}
			jsItem.Set("binary", jsBinary)
		}

		// Add paired item if present
		if item.PairedItem != nil {
			jsItem.Set("pairedItem", vm.ToValue(item.PairedItem))
		}

		arr.Set(fmt.Sprintf("%d", i), jsItem)
	}

	return arr
}

// convertFromJSArray converts a JavaScript array back to []model.DataItem
func convertFromJSArray(jsValue goja.Value) ([]model.DataItem, error) {
	// Export to Go value
	exported := jsValue.Export()

	// Marshal to JSON and unmarshal to get proper types
	jsonData, err := json.Marshal(exported)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var rawItems []map[string]interface{}
	if err := json.Unmarshal(jsonData, &rawItems); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	// Convert to model.DataItem
	var items []model.DataItem
	for _, rawItem := range rawItems {
		item := model.DataItem{}

		// Extract JSON data
		if jsonData, ok := rawItem["json"].(map[string]interface{}); ok {
			item.JSON = jsonData
		} else {
			item.JSON = rawItem
		}

		// Extract binary data if present
		if binaryData, ok := rawItem["binary"].(map[string]interface{}); ok {
			item.Binary = make(map[string]model.BinaryData)
			for key, value := range binaryData {
				if binaryMap, ok := value.(map[string]interface{}); ok {
					binary := model.BinaryData{}
					if data, ok := binaryMap["data"].(string); ok {
						binary.Data = data
					}
					if mimeType, ok := binaryMap["mimeType"].(string); ok {
						binary.MimeType = mimeType
					}
					if fileName, ok := binaryMap["fileName"].(string); ok {
						binary.FileName = fileName
					}
					if fileExt, ok := binaryMap["fileExtension"].(string); ok {
						binary.FileExtension = fileExt
					}
					item.Binary[key] = binary
				}
			}
		}

		items = append(items, item)
	}

	return items, nil
}

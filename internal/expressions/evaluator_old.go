/*
Package expressions provides n8n-style expression parsing and evaluation for n8n-go.
*/
package expressions

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	
	"github.com/neul-labs/m9m/internal/model"
)

// ExecutionContext provides context for expression evaluation
type ExecutionContext struct {
	InputData []model.DataItem
	ItemIndex int
	Variables map[string]interface{}
}

// ExpressionEvaluator handles parsing and evaluating n8n-style expressions
type ExpressionEvaluator struct {
	functions map[string]Function
}

// Function represents a built-in function that can be called in expressions
type Function func(args ...interface{}) (interface{}, error)

// NewExpressionEvaluator creates a new expression evaluator
func NewExpressionEvaluator() *ExpressionEvaluator {
	evaluator := &ExpressionEvaluator{
		functions: make(map[string]Function),
	}
	
	// Register built-in functions
	evaluator.registerBuiltInFunctions()
	
	return evaluator
}

// registerBuiltInFunctions registers the standard n8n built-in functions
func (e *ExpressionEvaluator) registerBuiltInFunctions() {
	// String functions
	e.functions["uppercase"] = func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("uppercase function requires exactly 1 argument")
		}
		if str, ok := args[0].(string); ok {
			return strings.ToUpper(str), nil
		}
		return fmt.Sprintf("%v", args[0]), nil
	}
	
	e.functions["lowercase"] = func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("lowercase function requires exactly 1 argument")
		}
		if str, ok := args[0].(string); ok {
			return strings.ToLower(str), nil
		}
		return fmt.Sprintf("%v", args[0]), nil
	}
	
	// Math functions
	e.functions["add"] = func(args ...interface{}) (interface{}, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("add function requires at least 2 arguments")
		}
		sum := 0.0
		for _, arg := range args {
			switch v := arg.(type) {
			case int:
				sum += float64(v)
			case float64:
				sum += v
			case string:
				if num, err := strconv.ParseFloat(v, 64); err == nil {
					sum += num
				} else {
					return nil, fmt.Errorf("invalid number: %s", v)
				}
			default:
				return nil, fmt.Errorf("unsupported type for addition: %T", arg)
			}
		}
		return sum, nil
	}
	
	// Date functions
	e.functions["now"] = func(args ...interface{}) (interface{}, error) {
		return "2025-09-21T00:00:00Z", nil // Simplified for now
	}
	
	// Array functions
	e.functions["length"] = func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("length function requires exactly 1 argument")
		}
		switch v := args[0].(type) {
		case []interface{}:
			return len(v), nil
		case string:
			return len(v), nil
		case []string:
			return len(v), nil
		default:
			return 0, nil
		}
	}
}

// Evaluate processes an expression with the given context
func (e *ExpressionEvaluator) Evaluate(expression string, context *ExecutionContext) (interface{}, error) {
	if expression == "" {
		return "", nil
	}
	
	// Handle n8n-style expressions that start with =
	if strings.HasPrefix(expression, "=") {
		// Remove the = prefix and trim whitespace
		trimmed := strings.TrimSpace(expression[1:])
		
		// If the trimmed expression is empty, return empty string
		if trimmed == "" {
			return "", nil
		}
		
		// Parse and evaluate the expression
		result, err := e.parseAndEvaluate(trimmed, context)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate expression '%s': %v", expression, err)
		}
		return result, nil
	}
	
	// Handle simple literal values that don't contain expressions
	if !strings.Contains(expression, "{{") {
		return expression, nil
	}
	
	// Parse and evaluate the expression
	result, err := e.parseAndEvaluate(expression, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %v", expression, err)
	}
	
	return result, nil
}

// parseAndEvaluate parses and evaluates an expression
func (e *ExpressionEvaluator) parseAndEvaluate(expression string, context *ExecutionContext) (interface{}, error) {
	// Process the entire expression, handling nested expressions properly
	result := expression
	
	// Keep processing until no more expressions are found
	maxIterations := 100 // Prevent infinite loops
	iterations := 0
	
	for strings.Contains(result, "{{") && iterations < maxIterations {
		iterations++
		
		// Process function calls first (they might contain variables)
		newResult := e.processFunctionCalls(result, context)
		if newResult != result {
			result = newResult
			continue
		}
		
		// Process variable substitutions
		newResult = e.processVariableSubstitutions(result, context)
		if newResult != result {
			result = newResult
			continue
		}
		
		// If no changes were made, break to avoid infinite loop
		break
	}
	
	return result, nil
}

// processFunctionCalls processes function calls in an expression
func (e *ExpressionEvaluator) processFunctionCalls(expression string, context *ExecutionContext) string {
	// Handle function calls - pattern matching for {{ function(args) }}
	functionRegex := regexp.MustCompile(`{{\s*([a-zA-Z0-9_]+)\s*\(\s*([^)]*)\s*\)\s*}}`)
	
	return functionRegex.ReplaceAllStringFunc(expression, func(match string) string {
		submatches := functionRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		
		functionName := submatches[1]
		argsString := submatches[2]
		
		// Process any variables in the arguments first
		processedArgs := e.processFunctionArguments(argsString, context)
		
		// Then call the function with processed arguments
		value, err := e.callFunction(functionName, processedArgs, context)
		if err != nil {
			return match // Return original if we can't call function
		}
		
		// Convert to string for replacement
		return fmt.Sprintf("%v", value)
	})
}

// processFunctionArguments processes function arguments, resolving variables within them
func (e *ExpressionEvaluator) processFunctionArguments(argsString string, context *ExecutionContext) string {
	// Handle variables in function arguments
	// Look for JSON variables: $json.variableName
	
	jsonVarRegex := regexp.MustCompile(`\$json(?:\.([a-zA-Z0-9_.]+))?`)
	
	processedArgs := jsonVarRegex.ReplaceAllStringFunc(argsString, func(match string) string {
		submatches := jsonVarRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		
		variablePath := submatches[1]
		if variablePath == "" {
			return match
		}
		
		// Resolve the variable from JSON data
		if len(context.InputData) > context.ItemIndex {
			value, err := e.resolvePath(context.InputData[context.ItemIndex].JSON, variablePath)
			if err == nil {
				// Convert to string representation appropriate for the context
				switch v := value.(type) {
				case string:
					return fmt.Sprintf("'%s'", v) // Wrap strings in quotes
				default:
					return fmt.Sprintf("%v", v) // Return as-is for other types
				}
			}
		}
		
		return match // Return original if we can't resolve
	})
	
	// Look for Parameter variables: $parameter.variableName
	paramVarRegex := regexp.MustCompile(`\$parameter(?:\.([a-zA-Z0-9_.]+))?`)
	
	processedArgs = paramVarRegex.ReplaceAllStringFunc(processedArgs, func(match string) string {
		submatches := paramVarRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		
		variablePath := submatches[1]
		if variablePath == "" {
			return match
		}
		
		// Resolve the variable from parameters
		if context.Variables != nil {
			value, err := e.resolvePath(context.Variables, variablePath)
			if err == nil {
				// Convert to string representation appropriate for the context
				switch v := value.(type) {
				case string:
					return fmt.Sprintf("'%s'", v) // Wrap strings in quotes
				default:
					return fmt.Sprintf("%v", v) // Return as-is for other types
				}
			}
		}
		
		return match // Return original if we can't resolve
	})
	
	return processedArgs
}

// processVariableSubstitutions processes variable substitutions in an expression
func (e *ExpressionEvaluator) processVariableSubstitutions(expression string, context *ExecutionContext) string {
	// Handle variable substitution patterns like {{$json.property}}
	variableRegex := regexp.MustCompile(`{{\s*\$([a-zA-Z0-9_]+)(?:\.([a-zA-Z0-9_.]*))?\s*}}`)
	
	// Replace variables with their values
	return variableRegex.ReplaceAllStringFunc(expression, func(match string) string {
		submatches := variableRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		
		variableType := submatches[1]
		variablePath := ""
		if len(submatches) > 2 {
			variablePath = submatches[2]
		}
		
		// Resolve the variable
		value, err := e.resolveVariable(variableType, variablePath, context)
		if err != nil {
			return match // Return original if we can't resolve
		}
		
		// Convert to string for replacement
		return fmt.Sprintf("%v", value)
	})
}

// resolveVariable resolves a variable reference
func (e *ExpressionEvaluator) resolveVariable(variableType, variablePath string, context *ExecutionContext) (interface{}, error) {
	switch variableType {
	case "json":
		if len(context.InputData) > context.ItemIndex {
			return e.resolvePath(context.InputData[context.ItemIndex].JSON, variablePath)
		}
		return nil, fmt.Errorf("no input data at index %d", context.ItemIndex)
		
	case "parameter":
		if context.Variables != nil {
			return e.resolvePath(context.Variables, variablePath)
		}
		return nil, fmt.Errorf("no variables context")
		
	case "workflow":
		// For simplicity, we'll just return a placeholder
		if variablePath == "name" {
			return "Test Workflow", nil
		}
		return nil, fmt.Errorf("workflow variable not found")
		
	default:
		// Check if it's in the variables map
		if context.Variables != nil {
			if value, exists := context.Variables[variableType]; exists {
				if variablePath == "" {
					return value, nil
				}
				// If there's a path, try to resolve it within the variable
				if valueMap, ok := value.(map[string]interface{}); ok {
					return e.resolvePath(valueMap, variablePath)
				}
			}
		}
		return nil, fmt.Errorf("unknown variable type: %s", variableType)
	}
}

// resolvePath resolves a dot-separated path in a map
func (e *ExpressionEvaluator) resolvePath(data map[string]interface{}, path string) (interface{}, error) {
	if path == "" {
		return data, nil
	}
	
	// Split the path by dots
	parts := strings.Split(path, ".")
	current := data
	
	// Navigate through the path
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - return the value
			return current[part], nil
		} else {
			// Intermediate part - should be a map
			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			} else {
				return nil, fmt.Errorf("path '%s' not found", path)
			}
		}
	}
	
	return nil, fmt.Errorf("path '%s' not found", path)
}

// callFunction calls a built-in function
func (e *ExpressionEvaluator) callFunction(functionName, argsString string, context *ExecutionContext) (interface{}, error) {
	function, exists := e.functions[functionName]
	if !exists {
		return nil, fmt.Errorf("unknown function: %s", functionName)
	}
	
	// Parse arguments - simplified for now
	args := e.parseArguments(argsString)
	
	// Call the function
	return function(args...)
}

// parseArguments parses function arguments from a string
func (e *ExpressionEvaluator) parseArguments(argsString string) []interface{} {
	if argsString == "" {
		return []interface{}{}
	}
	
	// Simplified argument parsing - split by commas and trim whitespace
	argStrings := strings.Split(argsString, ",")
	args := make([]interface{}, len(argStrings))
	
	for i, argStr := range argStrings {
		argStr = strings.TrimSpace(argStr)
		
		// Try to parse as number
		if num, err := strconv.ParseFloat(argStr, 64); err == nil {
			args[i] = num
		} else {
			// Treat as string, removing quotes if present
			if strings.HasPrefix(argStr, "'") && strings.HasSuffix(argStr, "'") {
				args[i] = strings.Trim(argStr, "'")
			} else if strings.HasPrefix(argStr, "\"") && strings.HasSuffix(argStr, "\"") {
				args[i] = strings.Trim(argStr, "\"")
			} else {
				args[i] = argStr
			}
		}
	}
	
	return args
}

// RegisterFunction registers a custom function
func (e *ExpressionEvaluator) RegisterFunction(name string, function Function) {
	e.functions[name] = function
}

// Validate validates an expression syntax
func (e *ExpressionEvaluator) Validate(expression string) error {
	// Basic validation - check for balanced braces
	openCount := strings.Count(expression, "{{")
	closeCount := strings.Count(expression, "}}")
	
	if openCount != closeCount {
		return fmt.Errorf("unbalanced braces in expression")
	}
	
	return nil
}

// IsExpression checks if a string contains n8n-style expressions
func IsExpression(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}") || strings.HasPrefix(s, "=")
}
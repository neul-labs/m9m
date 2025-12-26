package runtime

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dipankar/m9m/internal/model"
)

// N8n-specific helper functions that provide the familiar n8n syntax

func (js *JavaScriptRuntime) createJsonHelper() interface{} {
	return map[string]interface{}{
		// Direct access to current item's JSON
		"data": func() interface{} {
			if js.executionContext != nil && len(js.getCurrentItems()) > 0 {
				currentItem := js.getCurrentItems()[js.executionContext.ItemIndex]
				return currentItem.JSON
			}
			return nil
		},
	}
}

func (js *JavaScriptRuntime) createNodeHelper() interface{} {
	return map[string]interface{}{
		// Access data from other nodes
		"get": func(nodeName string, outputIndex ...int) interface{} {
			// Mock implementation - in real version, would access workflow execution data
			return map[string]interface{}{
				"json": map[string]interface{}{
					"mock": true,
					"node": nodeName,
				},
			}
		},
		// Get all items from a node
		"getAll": func(nodeName string, outputIndex ...int) []interface{} {
			return []interface{}{
				map[string]interface{}{
					"json": map[string]interface{}{
						"mock": true,
						"node": nodeName,
					},
				},
			}
		},
	}
}

func (js *JavaScriptRuntime) createParameterHelper() interface{} {
	return map[string]interface{}{
		// Get parameter value
		"get": func(parameterName string) interface{} {
			if js.executionContext != nil {
				if value, exists := js.executionContext.Variables[parameterName]; exists {
					return value
				}
			}
			return nil
		},
	}
}

func (js *JavaScriptRuntime) createWorkflowHelper() interface{} {
	return map[string]interface{}{
		"id":     js.executionContext.WorkflowID,
		"name":   "Mock Workflow",
		"active": true,
	}
}

func (js *JavaScriptRuntime) createExecutionHelper() interface{} {
	return map[string]interface{}{
		"id":        js.executionContext.ExecutionID,
		"mode":      js.executionContext.Mode,
		"resumeUrl": "",
	}
}

func (js *JavaScriptRuntime) createItemsHelper() interface{} {
	return map[string]interface{}{
		// Get all items
		"all": func() []interface{} {
			items := js.getCurrentItems()
			var result []interface{}
			for _, item := range items {
				result = append(result, map[string]interface{}{
					"json": item.JSON,
				})
			}
			return result
		},
		// Get current item
		"current": func() interface{} {
			items := js.getCurrentItems()
			if len(items) > js.executionContext.ItemIndex {
				return map[string]interface{}{
					"json": items[js.executionContext.ItemIndex].JSON,
				}
			}
			return nil
		},
		// Get first item
		"first": func() interface{} {
			items := js.getCurrentItems()
			if len(items) > 0 {
				return map[string]interface{}{
					"json": items[0].JSON,
				}
			}
			return nil
		},
		// Get last item
		"last": func() interface{} {
			items := js.getCurrentItems()
			if len(items) > 0 {
				return map[string]interface{}{
					"json": items[len(items)-1].JSON,
				}
			}
			return nil
		},
	}
}

func (js *JavaScriptRuntime) createItemsHelperWithData(items []model.DataItem) interface{} {
	return map[string]interface{}{
		// Get all items
		"all": func() []interface{} {
			var result []interface{}
			for _, item := range items {
				result = append(result, map[string]interface{}{
					"json": item.JSON,
				})
			}
			return result
		},
		// Get current item
		"current": func() interface{} {
			if len(items) > js.executionContext.ItemIndex {
				return map[string]interface{}{
					"json": items[js.executionContext.ItemIndex].JSON,
				}
			}
			return nil
		},
		// Get first item
		"first": func() interface{} {
			if len(items) > 0 {
				return map[string]interface{}{
					"json": items[0].JSON,
				}
			}
			return nil
		},
		// Get last item
		"last": func() interface{} {
			if len(items) > 0 {
				return map[string]interface{}{
					"json": items[len(items)-1].JSON,
				}
			}
			return nil
		},
	}
}

func (js *JavaScriptRuntime) createInputHelper() interface{} {
	return map[string]interface{}{
		"all": func() []interface{} {
			return js.createItemsHelper().(map[string]interface{})["all"].(func() []interface{})()
		},
		"first": func() interface{} {
			return js.createItemsHelper().(map[string]interface{})["first"].(func() interface{})()
		},
		"last": func() interface{} {
			return js.createItemsHelper().(map[string]interface{})["last"].(func() interface{})()
		},
		"item": func() interface{} {
			return js.createItemsHelper().(map[string]interface{})["current"].(func() interface{})()
		},
	}
}

func (js *JavaScriptRuntime) createDollarHelper() interface{} {
	return map[string]interface{}{
		"json": js.createJsonHelper(),
		"node": js.createNodeHelper(),
		"parameter": js.createParameterHelper(),
		"workflow": js.createWorkflowHelper(),
		"execution": js.createExecutionHelper(),
		"items": js.createItemsHelper(),
		"input": js.createInputHelper(),
		"now": func() string {
			return time.Now().Format(time.RFC3339)
		},
		"today": func() string {
			return time.Now().Format("2006-01-02")
		},
		"binary": map[string]interface{}{
			"base64": func(data string) string {
				// Base64 encode
				return data // Mock implementation
			},
		},
		"env": js.executionContext.Variables,
	}
}

func (js *JavaScriptRuntime) createMomentHelper() interface{} {
	// Enhanced moment.js compatible helper
	var momentFunc func(call goja.FunctionCall) goja.Value
	momentFunc = func(call goja.FunctionCall) goja.Value {
		var date time.Time
		if len(call.Arguments) > 0 {
			input := call.Argument(0).String()
			parsed, err := time.Parse(time.RFC3339, input)
			if err != nil {
				parsed, err = time.Parse("2006-01-02", input)
				if err != nil {
					date = time.Now()
				} else {
					date = parsed
				}
			} else {
				date = parsed
			}
		} else {
			date = time.Now()
		}

		momentObj := js.vm.NewObject()

		momentObj.Set("format", func(call goja.FunctionCall) goja.Value {
			format := "2006-01-02 15:04:05"
			if len(call.Arguments) > 0 {
				format = js.convertMomentFormat(call.Argument(0).String())
			}
			return js.vm.ToValue(date.Format(format))
		})

		momentObj.Set("add", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) >= 2 {
				amount := call.Argument(0).ToInteger()
				unit := call.Argument(1).String()

				switch unit {
				case "days", "day", "d":
					date = date.AddDate(0, 0, int(amount))
				case "hours", "hour", "h":
					date = date.Add(time.Duration(amount) * time.Hour)
				case "minutes", "minute", "m":
					date = date.Add(time.Duration(amount) * time.Minute)
				case "seconds", "second", "s":
					date = date.Add(time.Duration(amount) * time.Second)
				case "months", "month", "M":
					date = date.AddDate(0, int(amount), 0)
				case "years", "year", "y":
					date = date.AddDate(int(amount), 0, 0)
				}
			}

			// Return new moment object
			newCall := goja.FunctionCall{Arguments: []goja.Value{js.vm.ToValue(date.Format(time.RFC3339))}}
			return momentFunc(newCall)
		})

		momentObj.Set("subtract", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) >= 2 {
				amount := -call.Argument(0).ToInteger()
				unit := call.Argument(1).String()

				// Call add with negated amount
				newDate := date
				switch unit {
				case "days", "day", "d":
					newDate = newDate.AddDate(0, 0, int(amount))
				case "hours", "hour", "h":
					newDate = newDate.Add(time.Duration(amount) * time.Hour)
				case "minutes", "minute", "m":
					newDate = newDate.Add(time.Duration(amount) * time.Minute)
				case "seconds", "second", "s":
					newDate = newDate.Add(time.Duration(amount) * time.Second)
				case "months", "month", "M":
					newDate = newDate.AddDate(0, int(amount), 0)
				case "years", "year", "y":
					newDate = newDate.AddDate(int(amount), 0, 0)
				}
				newCall := goja.FunctionCall{Arguments: []goja.Value{js.vm.ToValue(newDate.Format(time.RFC3339))}}
				return momentFunc(newCall)
			}
			return js.vm.ToValue(momentObj)
		})

		momentObj.Set("unix", func(call goja.FunctionCall) goja.Value {
			return js.vm.ToValue(date.Unix())
		})

		momentObj.Set("valueOf", func(call goja.FunctionCall) goja.Value {
			return js.vm.ToValue(date.UnixMilli())
		})

		momentObj.Set("toDate", func(call goja.FunctionCall) goja.Value {
			return js.vm.ToValue(date)
		})

		momentObj.Set("isValid", func(call goja.FunctionCall) goja.Value {
			return js.vm.ToValue(!date.IsZero())
		})

		momentObj.Set("diff", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) > 0 {
				_ = call.Argument(0) // otherMoment - not used in this simplified impl
				// Extract date from other moment object
				otherDate := time.Now() // Mock implementation
				diff := date.Sub(otherDate)

				unit := "milliseconds"
				if len(call.Arguments) > 1 {
					unit = call.Argument(1).String()
				}

				switch unit {
				case "seconds":
					return js.vm.ToValue(int64(diff.Seconds()))
				case "minutes":
					return js.vm.ToValue(int64(diff.Minutes()))
				case "hours":
					return js.vm.ToValue(int64(diff.Hours()))
				case "days":
					return js.vm.ToValue(int64(diff.Hours() / 24))
				default:
					return js.vm.ToValue(diff.Milliseconds())
				}
			}
			return js.vm.ToValue(0)
		})

		return js.vm.ToValue(momentObj)
	}

	moment := js.vm.NewObject()
	moment.Set("call", momentFunc)

	// Static methods
	moment.Set("now", func() goja.Value {
		return js.vm.ToValue(time.Now().UnixMilli())
	})

	moment.Set("utc", func(call goja.FunctionCall) goja.Value {
		return momentFunc(call)
	})

	return moment
}

func (js *JavaScriptRuntime) createLodashHelper() interface{} {
	// Enhanced lodash compatible helper
	return map[string]interface{}{
		"get": func(object interface{}, path string, defaultValue interface{}) interface{} {
			return js.getNestedValue(object, path, defaultValue)
		},
		"set": func(object interface{}, path string, value interface{}) interface{} {
			return js.setNestedValue(object, path, value)
		},
		"has": func(object interface{}, path string) bool {
			return js.hasNestedValue(object, path)
		},
		"clone": func(obj interface{}) interface{} {
			data, _ := json.Marshal(obj)
			var result interface{}
			json.Unmarshal(data, &result)
			return result
		},
		"cloneDeep": func(obj interface{}) interface{} {
			data, _ := json.Marshal(obj)
			var result interface{}
			json.Unmarshal(data, &result)
			return result
		},
		"merge": func(target interface{}, sources ...interface{}) interface{} {
			// Simple merge implementation
			if targetMap, ok := target.(map[string]interface{}); ok {
				for _, source := range sources {
					if sourceMap, ok := source.(map[string]interface{}); ok {
						for key, value := range sourceMap {
							targetMap[key] = value
						}
					}
				}
				return targetMap
			}
			return target
		},
		"isEmpty": func(value interface{}) bool {
			if value == nil {
				return true
			}
			if str, ok := value.(string); ok {
				return str == ""
			}
			if arr, ok := value.([]interface{}); ok {
				return len(arr) == 0
			}
			if obj, ok := value.(map[string]interface{}); ok {
				return len(obj) == 0
			}
			return false
		},
		"isArray": func(value interface{}) bool {
			_, ok := value.([]interface{})
			return ok
		},
		"isObject": func(value interface{}) bool {
			_, ok := value.(map[string]interface{})
			return ok
		},
		"isString": func(value interface{}) bool {
			_, ok := value.(string)
			return ok
		},
		"isNumber": func(value interface{}) bool {
			switch value.(type) {
			case int, int32, int64, float32, float64:
				return true
			default:
				return false
			}
		},
		"isBoolean": func(value interface{}) bool {
			_, ok := value.(bool)
			return ok
		},
		"isNull": func(value interface{}) bool {
			return value == nil
		},
		"isUndefined": func(value interface{}) bool {
			return value == nil
		},
		"keys": func(object interface{}) []string {
			if obj, ok := object.(map[string]interface{}); ok {
				var keys []string
				for key := range obj {
					keys = append(keys, key)
				}
				return keys
			}
			return []string{}
		},
		"values": func(object interface{}) []interface{} {
			if obj, ok := object.(map[string]interface{}); ok {
				var values []interface{}
				for _, value := range obj {
					values = append(values, value)
				}
				return values
			}
			return []interface{}{}
		},
		"pick": func(object interface{}, keys ...string) interface{} {
			if obj, ok := object.(map[string]interface{}); ok {
				result := make(map[string]interface{})
				for _, key := range keys {
					if value, exists := obj[key]; exists {
						result[key] = value
					}
				}
				return result
			}
			return object
		},
		"omit": func(object interface{}, keys ...string) interface{} {
			if obj, ok := object.(map[string]interface{}); ok {
				result := make(map[string]interface{})
				omitSet := make(map[string]bool)
				for _, key := range keys {
					omitSet[key] = true
				}
				for key, value := range obj {
					if !omitSet[key] {
						result[key] = value
					}
				}
				return result
			}
			return object
		},
		"map": func(collection interface{}, iteratee goja.Value) interface{} {
			if arr, ok := collection.([]interface{}); ok {
				var result []interface{}
				if callable, ok := goja.AssertFunction(iteratee); ok {
					for i, item := range arr {
						value, _ := callable(goja.Undefined(), js.vm.ToValue(item), js.vm.ToValue(i), js.vm.ToValue(collection))
						result = append(result, value.Export())
					}
				}
				return result
			}
			return collection
		},
		"filter": func(collection interface{}, predicate goja.Value) interface{} {
			if arr, ok := collection.([]interface{}); ok {
				var result []interface{}
				if callable, ok := goja.AssertFunction(predicate); ok {
					for i, item := range arr {
						value, _ := callable(goja.Undefined(), js.vm.ToValue(item), js.vm.ToValue(i), js.vm.ToValue(collection))
						if js.isTruthy(value.Export()) {
							result = append(result, item)
						}
					}
				}
				return result
			}
			return collection
		},
		"find": func(collection interface{}, predicate goja.Value) interface{} {
			if arr, ok := collection.([]interface{}); ok {
				if callable, ok := goja.AssertFunction(predicate); ok {
					for i, item := range arr {
						value, _ := callable(goja.Undefined(), js.vm.ToValue(item), js.vm.ToValue(i), js.vm.ToValue(collection))
						if js.isTruthy(value.Export()) {
							return item
						}
					}
				}
			}
			return nil
		},
		"reduce": func(collection interface{}, iteratee goja.Value, accumulator interface{}) interface{} {
			if arr, ok := collection.([]interface{}); ok {
				if callable, ok := goja.AssertFunction(iteratee); ok {
					acc := accumulator
					for i, item := range arr {
						value, _ := callable(goja.Undefined(), js.vm.ToValue(acc), js.vm.ToValue(item), js.vm.ToValue(i), js.vm.ToValue(collection))
						acc = value.Export()
					}
					return acc
				}
			}
			return accumulator
		},
		"uniq": func(array interface{}) interface{} {
			if arr, ok := array.([]interface{}); ok {
				seen := make(map[string]bool)
				var result []interface{}
				for _, item := range arr {
					key := fmt.Sprintf("%v", item)
					if !seen[key] {
						seen[key] = true
						result = append(result, item)
					}
				}
				return result
			}
			return array
		},
		"flatten": func(array interface{}) interface{} {
			if arr, ok := array.([]interface{}); ok {
				var result []interface{}
				for _, item := range arr {
					if subArr, ok := item.([]interface{}); ok {
						result = append(result, subArr...)
					} else {
						result = append(result, item)
					}
				}
				return result
			}
			return array
		},
		"compact": func(array interface{}) interface{} {
			if arr, ok := array.([]interface{}); ok {
				var result []interface{}
				for _, item := range arr {
					if js.isTruthy(item) {
						result = append(result, item)
					}
				}
				return result
			}
			return array
		},
	}
}

// Helper methods for n8n functionality

func (js *JavaScriptRuntime) getCurrentItems() []model.DataItem {
	// This would be set during execution - for now return empty
	return []model.DataItem{}
}

func (js *JavaScriptRuntime) getNestedValue(object interface{}, path string, defaultValue interface{}) interface{} {
	if object == nil {
		return defaultValue
	}

	keys := strings.Split(path, ".")
	current := object

	for _, key := range keys {
		// Handle array indices
		if strings.Contains(key, "[") && strings.Contains(key, "]") {
			re := regexp.MustCompile(`([^[]+)\[(\d+)\]`)
			matches := re.FindStringSubmatch(key)
			if len(matches) == 3 {
				arrayKey := matches[1]
				index, _ := strconv.Atoi(matches[2])

				if obj, ok := current.(map[string]interface{}); ok {
					if arr, ok := obj[arrayKey].([]interface{}); ok {
						if index >= 0 && index < len(arr) {
							current = arr[index]
						} else {
							return defaultValue
						}
					} else {
						return defaultValue
					}
				} else {
					return defaultValue
				}
			}
		} else {
			// Regular object property access
			if obj, ok := current.(map[string]interface{}); ok {
				if value, exists := obj[key]; exists {
					current = value
				} else {
					return defaultValue
				}
			} else {
				return defaultValue
			}
		}
	}

	return current
}

func (js *JavaScriptRuntime) setNestedValue(object interface{}, path string, value interface{}) interface{} {
	if obj, ok := object.(map[string]interface{}); ok {
		keys := strings.Split(path, ".")
		current := obj

		for i, key := range keys {
			if i == len(keys)-1 {
				// Set the final value
				current[key] = value
			} else {
				// Navigate or create intermediate objects
				if _, exists := current[key]; !exists {
					current[key] = make(map[string]interface{})
				}
				if nextObj, ok := current[key].(map[string]interface{}); ok {
					current = nextObj
				} else {
					// Can't navigate further
					break
				}
			}
		}
	}
	return object
}

func (js *JavaScriptRuntime) hasNestedValue(object interface{}, path string) bool {
	if object == nil {
		return false
	}

	keys := strings.Split(path, ".")
	current := object

	for _, key := range keys {
		if obj, ok := current.(map[string]interface{}); ok {
			if value, exists := obj[key]; exists {
				current = value
			} else {
				return false
			}
		} else {
			return false
		}
	}

	return true
}

func (js *JavaScriptRuntime) convertMomentFormat(momentFormat string) string {
	// Convert moment.js format to Go time format
	replacements := map[string]string{
		"YYYY": "2006",
		"YY":   "06",
		"MM":   "01",
		"M":    "1",
		"DD":   "02",
		"D":    "2",
		"HH":   "15",
		"H":    "15", // Go doesn't support single digit 24-hour
		"hh":   "03",
		"h":    "3",
		"mm":   "04",
		"m":    "4",
		"ss":   "05",
		"s":    "5",
		"A":    "PM",
		"a":    "pm",
	}

	goFormat := momentFormat
	for moment, goTime := range replacements {
		goFormat = strings.ReplaceAll(goFormat, moment, goTime)
	}

	return goFormat
}

func (js *JavaScriptRuntime) isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	case int, int32, int64:
		return v != 0
	case float32, float64:
		return v != 0.0
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true
	}
}
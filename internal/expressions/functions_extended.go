package expressions

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// flattenArgs extracts float64 values from arguments, unpacking arrays when a single
// array argument is passed (e.g. sum([1,2,3]) instead of sum(1,2,3)).
func flattenArgs(vm *goja.Runtime, args []goja.Value) []float64 {
	if len(args) == 1 {
		// Check if the single argument is an array
		if obj := args[0].ToObject(vm); obj != nil {
			if arrLen := obj.Get("length"); arrLen != nil && !goja.IsUndefined(arrLen) {
				n := int(arrLen.ToInteger())
				values := make([]float64, 0, n)
				for i := 0; i < n; i++ {
					v := obj.Get(fmt.Sprintf("%d", i))
					if v != nil {
						values = append(values, v.ToFloat())
					}
				}
				return values
			}
		}
	}
	values := make([]float64, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.ToFloat())
	}
	return values
}

// MathExtensions provides mathematical functions
type MathExtensions struct{}

func (m *MathExtensions) GetCategory() string {
	return "math"
}

func (m *MathExtensions) GetFunctionNames() []string {
	return []string{
		"min", "max", "average", "sum", "round", "ceil", "floor",
		"abs", "random", "randomInt", "add", "subtract", "multiply",
		"divide", "modulo", "pow", "sqrt", "sin", "cos", "tan",
	}
}

func (m *MathExtensions) RegisterFunctions(vm *goja.Runtime) error {
	functions := map[string]func(call goja.FunctionCall) goja.Value{
		"min":        m.min(vm),
		"max":        m.max(vm),
		"average":    m.average(vm),
		"sum":        m.sum(vm),
		"round":      m.round(vm),
		"ceil":       m.ceil(vm),
		"floor":      m.floor(vm),
		"abs":        m.abs(vm),
		"random":     m.random(vm),
		"randomInt":  m.randomInt(vm),
		"add":        m.add(vm),
		"subtract":   m.subtract(vm),
		"multiply":   m.multiply(vm),
		"divide":     m.divide(vm),
		"modulo":     m.modulo(vm),
		"pow":        m.pow(vm),
		"sqrt":       m.sqrt(vm),
		"sin":        m.sin(vm),
		"cos":        m.cos(vm),
		"tan":        m.tan(vm),
	}

	for name, fn := range functions {
		vm.Set(name, fn)
	}

	return nil
}

func (m *MathExtensions) min(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		values := flattenArgs(vm, call.Arguments)
		if len(values) == 0 {
			return vm.ToValue(math.Inf(1))
		}

		min := values[0]
		for i := 1; i < len(values); i++ {
			if values[i] < min {
				min = values[i]
			}
		}
		return vm.ToValue(float64(min))
	}
}

func (m *MathExtensions) max(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		values := flattenArgs(vm, call.Arguments)
		if len(values) == 0 {
			return vm.ToValue(math.Inf(-1))
		}

		max := values[0]
		for i := 1; i < len(values); i++ {
			if values[i] > max {
				max = values[i]
			}
		}
		return vm.ToValue(max)
	}
}

func (m *MathExtensions) average(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		values := flattenArgs(vm, call.Arguments)
		if len(values) == 0 {
			return vm.ToValue(0)
		}

		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return vm.ToValue(sum / float64(len(values)))
	}
}

func (m *MathExtensions) sum(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		values := flattenArgs(vm, call.Arguments)
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		// Use RunString to create a proper JavaScript number
		val, _ := vm.RunString(fmt.Sprintf("%.10g", sum))
		return val
	}
}

func (m *MathExtensions) add(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return m.sum(vm) // add is an alias for sum
}

func (m *MathExtensions) round(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("round requires 1 argument"))
		}

		val := call.Arguments[0].ToFloat()
		precision := 0
		if len(call.Arguments) > 1 {
			precision = int(call.Arguments[1].ToInteger())
		}

		multiplier := math.Pow(10, float64(precision))
		return vm.ToValue(math.Round(val*multiplier) / multiplier)
	}
}

func (m *MathExtensions) random(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(rand.Float64())
	}
}

func (m *MathExtensions) randomInt(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("randomInt requires 1 argument"))
		}

		max := int(call.Arguments[0].ToInteger())
		min := 0
		if len(call.Arguments) > 1 {
			min = max
			max = int(call.Arguments[1].ToInteger())
		}

		if min > max {
			min, max = max, min
		}

		return vm.ToValue(rand.Intn(max-min+1) + min)
	}
}

func (m *MathExtensions) ceil(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("ceil requires 1 argument"))
		}
		return vm.ToValue(math.Ceil(call.Arguments[0].ToFloat()))
	}
}

func (m *MathExtensions) floor(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("floor requires 1 argument"))
		}
		return vm.ToValue(math.Floor(call.Arguments[0].ToFloat()))
	}
}

func (m *MathExtensions) abs(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("abs requires 1 argument"))
		}
		return vm.ToValue(math.Abs(call.Arguments[0].ToFloat()))
	}
}

func (m *MathExtensions) subtract(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("subtract requires 2 arguments"))
		}
		return vm.ToValue(call.Arguments[0].ToFloat() - call.Arguments[1].ToFloat())
	}
}

func (m *MathExtensions) multiply(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("multiply requires 2 arguments"))
		}
		result := call.Arguments[0].ToFloat()
		for i := 1; i < len(call.Arguments); i++ {
			result *= call.Arguments[i].ToFloat()
		}
		return vm.ToValue(result)
	}
}

func (m *MathExtensions) divide(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("divide requires 2 arguments"))
		}
		dividend := call.Arguments[0].ToFloat()
		divisor := call.Arguments[1].ToFloat()
		if divisor == 0 {
			return vm.ToValue(math.Inf(1))
		}
		return vm.ToValue(dividend / divisor)
	}
}

func (m *MathExtensions) modulo(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("modulo requires 2 arguments"))
		}
		return vm.ToValue(math.Mod(call.Arguments[0].ToFloat(), call.Arguments[1].ToFloat()))
	}
}

func (m *MathExtensions) pow(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("pow requires 2 arguments"))
		}
		return vm.ToValue(math.Pow(call.Arguments[0].ToFloat(), call.Arguments[1].ToFloat()))
	}
}

func (m *MathExtensions) sqrt(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("sqrt requires 1 argument"))
		}
		return vm.ToValue(math.Sqrt(call.Arguments[0].ToFloat()))
	}
}

func (m *MathExtensions) sin(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("sin requires 1 argument"))
		}
		return vm.ToValue(math.Sin(call.Arguments[0].ToFloat()))
	}
}

func (m *MathExtensions) cos(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("cos requires 1 argument"))
		}
		return vm.ToValue(math.Cos(call.Arguments[0].ToFloat()))
	}
}

func (m *MathExtensions) tan(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("tan requires 1 argument"))
		}
		return vm.ToValue(math.Tan(call.Arguments[0].ToFloat()))
	}
}

func (m *MathExtensions) GetFunctionHelp(name string) *FunctionHelp {
	helpMap := map[string]*FunctionHelp{
		"add": {
			Name:        "add",
			Description: "Adds numbers together",
			Parameters: []ParameterInfo{
				{Name: "numbers", Type: "number", Required: true, Description: "Numbers to add"},
			},
			ReturnType: "number",
			Examples:   []string{`add(1, 2, 3)`, `add(10, 20)`},
		},
		"min": {
			Name:        "min",
			Description: "Returns the smallest number",
			Parameters: []ParameterInfo{
				{Name: "numbers", Type: "number", Required: true, Description: "Numbers to compare"},
			},
			ReturnType: "number",
			Examples:   []string{`min(1, 2, 3)`, `min(10, 5)`},
		},
		// Add more help entries...
	}

	if help, exists := helpMap[name]; exists {
		return help
	}

	return &FunctionHelp{
		Name:        name,
		Description: "Math function",
		ReturnType:  "number",
	}
}

// ArrayExtensions provides array manipulation functions
type ArrayExtensions struct{}

func (a *ArrayExtensions) GetCategory() string {
	return "array"
}

func (a *ArrayExtensions) GetFunctionNames() []string {
	return []string{
		"first", "last", "unique", "compact", "flatten", "chunk",
		"pluck", "randomItem", "length", "indexOf", "includes",
		"slice", "sort", "reverse", "filter", "map", "reduce",
	}
}

func (a *ArrayExtensions) RegisterFunctions(vm *goja.Runtime) error {
	functions := map[string]func(call goja.FunctionCall) goja.Value{
		"first":      a.first(vm),
		"last":       a.last(vm),
		"unique":     a.unique(vm),
		"compact":    a.compact(vm),
		"flatten":    a.flatten(vm),
		"chunk":      a.chunk(vm),
		"pluck":      a.pluck(vm),
		"randomItem": a.randomItem(vm),
		"length":     a.length(vm),
		"indexOf":    a.indexOf(vm),
		"includes":   a.includes(vm),
		"slice":      a.slice(vm),
		"sort":       a.sort(vm),
		"reverse":    a.reverse(vm),
	}

	for name, fn := range functions {
		vm.Set(name, fn)
	}

	return nil
}

func (a *ArrayExtensions) first(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("first requires 1 argument"))
		}

		arr := a.toSlice(call.Arguments[0])
		if len(arr) == 0 {
			return goja.Undefined()
		}
		// Convert int64 to int for compatibility
		if val, ok := arr[0].(int64); ok {
			return vm.ToValue(int(val))
		}
		return vm.ToValue(arr[0])
	}
}

func (a *ArrayExtensions) last(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("last requires 1 argument"))
		}

		arr := a.toSlice(call.Arguments[0])
		if len(arr) == 0 {
			return goja.Undefined()
		}
		// Convert int64 to int for compatibility
		lastVal := arr[len(arr)-1]
		if val, ok := lastVal.(int64); ok {
			return vm.ToValue(int(val))
		}
		return vm.ToValue(lastVal)
	}
}

func (a *ArrayExtensions) unique(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("unique requires 1 argument"))
		}

		arr := a.toSlice(call.Arguments[0])
		seen := make(map[string]bool)
		var result []interface{}

		for _, item := range arr {
			key := fmt.Sprintf("%v", item)
			if !seen[key] {
				seen[key] = true
				result = append(result, item)
			}
		}

		return vm.ToValue(result)
	}
}

func (a *ArrayExtensions) compact(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("compact requires 1 argument"))
		}

		arr := a.toSlice(call.Arguments[0])
		var result []interface{}

		for _, item := range arr {
			if !a.isFalsy(item) {
				result = append(result, item)
			}
		}

		return vm.ToValue(result)
	}
}

func (a *ArrayExtensions) flatten(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("flatten requires 1 argument"))
		}

		arr := a.toSlice(call.Arguments[0])
		depth := 1
		if len(call.Arguments) > 1 {
			depth = int(call.Arguments[1].ToInteger())
		}

		result := a.flattenRecursive(arr, depth)
		return vm.ToValue(result)
	}
}

func (a *ArrayExtensions) chunk(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("chunk requires 2 arguments"))
		}

		arr := a.toSlice(call.Arguments[0])
		size := int(call.Arguments[1].ToInteger())

		if size <= 0 {
			return vm.ToValue([]interface{}{})
		}

		var result []interface{}
		for i := 0; i < len(arr); i += size {
			end := i + size
			if end > len(arr) {
				end = len(arr)
			}
			result = append(result, arr[i:end])
		}

		return vm.ToValue(result)
	}
}

func (a *ArrayExtensions) pluck(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("pluck requires 2 arguments"))
		}

		arr := a.toSlice(call.Arguments[0])
		property := call.Arguments[1].String()

		var result []interface{}
		for _, item := range arr {
			if obj, ok := item.(map[string]interface{}); ok {
				if value, exists := obj[property]; exists {
					result = append(result, value)
				}
			}
		}

		return vm.ToValue(result)
	}
}

func (a *ArrayExtensions) randomItem(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("randomItem requires 1 argument"))
		}

		arr := a.toSlice(call.Arguments[0])
		if len(arr) == 0 {
			return goja.Undefined()
		}

		index := rand.Intn(len(arr))
		return vm.ToValue(arr[index])
	}
}

func (a *ArrayExtensions) length(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("length requires 1 argument"))
		}

		// Handle both arrays and strings
		arg := call.Arguments[0]
		if str := arg.String(); arg.ExportType().Kind().String() == "string" {
			return vm.ToValue(int(len(str)))
		}

		arr := a.toSlice(arg)
		return vm.ToValue(int(len(arr)))
	}
}

func (a *ArrayExtensions) indexOf(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("indexOf requires 2 arguments"))
		}

		arr := a.toSlice(call.Arguments[0])
		searchItem := call.Arguments[1].Export()

		for i, item := range arr {
			if fmt.Sprintf("%v", item) == fmt.Sprintf("%v", searchItem) {
				return vm.ToValue(i)
			}
		}

		return vm.ToValue(-1)
	}
}

func (a *ArrayExtensions) includes(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("includes requires 2 arguments"))
		}

		arr := a.toSlice(call.Arguments[0])
		searchItem := call.Arguments[1].Export()

		for _, item := range arr {
			if fmt.Sprintf("%v", item) == fmt.Sprintf("%v", searchItem) {
				return vm.ToValue(true)
			}
		}

		return vm.ToValue(false)
	}
}

func (a *ArrayExtensions) slice(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("slice requires 2 arguments"))
		}

		arr := a.toSlice(call.Arguments[0])
		start := int(call.Arguments[1].ToInteger())

		var end int = len(arr)
		if len(call.Arguments) > 2 {
			end = int(call.Arguments[2].ToInteger())
		}

		// Handle negative indices
		if start < 0 {
			start = len(arr) + start
		}
		if end < 0 {
			end = len(arr) + end
		}

		// Ensure bounds
		if start < 0 {
			start = 0
		}
		if end > len(arr) {
			end = len(arr)
		}
		if start > end {
			return vm.ToValue([]interface{}{})
		}

		return vm.ToValue(arr[start:end])
	}
}

func (a *ArrayExtensions) sort(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("sort requires 1 argument"))
		}

		arr := a.toSlice(call.Arguments[0])
		sorted := make([]interface{}, len(arr))
		copy(sorted, arr)

		sort.Slice(sorted, func(i, j int) bool {
			return fmt.Sprintf("%v", sorted[i]) < fmt.Sprintf("%v", sorted[j])
		})

		return vm.ToValue(sorted)
	}
}

func (a *ArrayExtensions) reverse(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("reverse requires 1 argument"))
		}

		arr := a.toSlice(call.Arguments[0])
		reversed := make([]interface{}, len(arr))

		for i, item := range arr {
			reversed[len(arr)-1-i] = item
		}

		return vm.ToValue(reversed)
	}
}

// Helper methods for ArrayExtensions
func (a *ArrayExtensions) toSlice(val goja.Value) []interface{} {
	exported := val.Export()

	// Handle different types
	switch v := exported.(type) {
	case []interface{}:
		return v
	case []string:
		result := make([]interface{}, len(v))
		for i, s := range v {
			result[i] = s
		}
		return result
	case []int:
		result := make([]interface{}, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	case []float64:
		result := make([]interface{}, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	default:
		// Use reflection as fallback
		rv := reflect.ValueOf(exported)
		if rv.Kind() == reflect.Slice {
			result := make([]interface{}, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				result[i] = rv.Index(i).Interface()
			}
			return result
		}
		return []interface{}{}
	}
}

func (a *ArrayExtensions) isFalsy(val interface{}) bool {
	if val == nil {
		return true
	}

	switch v := val.(type) {
	case bool:
		return !v
	case int:
		return v == 0
	case float64:
		return v == 0
	case string:
		return v == ""
	default:
		return false
	}
}

func (a *ArrayExtensions) flattenRecursive(arr []interface{}, depth int) []interface{} {
	if depth <= 0 {
		return arr
	}

	var result []interface{}
	for _, item := range arr {
		// Try to treat as array recursively
		if reflect.TypeOf(item).Kind() == reflect.Slice && depth > 0 {
			subArr := a.toSliceReflection(item)
			if len(subArr) > 0 {
				flattened := a.flattenRecursive(subArr, depth-1)
				result = append(result, flattened...)
			} else {
				result = append(result, item)
			}
		} else {
			result = append(result, item)
		}
	}

	return result
}

// toSliceReflection converts any slice type to []interface{} using reflection
func (a *ArrayExtensions) toSliceReflection(val interface{}) []interface{} {
	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Slice {
		return []interface{}{}
	}

	result := make([]interface{}, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		result[i] = rv.Index(i).Interface()
	}
	return result
}

func (a *ArrayExtensions) GetFunctionHelp(name string) *FunctionHelp {
	helpMap := map[string]*FunctionHelp{
		"first": {
			Name:        "first",
			Description: "Returns the first element of an array",
			Parameters: []ParameterInfo{
				{Name: "array", Type: "array", Required: true, Description: "The array to get first element from"},
			},
			ReturnType: "any",
			Examples:   []string{`first([1, 2, 3])`, `first($json.items)`},
		},
		// Add more help entries...
	}

	if help, exists := helpMap[name]; exists {
		return help
	}

	return &FunctionHelp{
		Name:        name,
		Description: "Array function",
		ReturnType:  "any",
	}
}

// DateExtensions provides date/time functions
type DateExtensions struct{}

func (d *DateExtensions) GetCategory() string {
	return "date"
}

func (d *DateExtensions) GetFunctionNames() []string {
	return []string{
		"now", "formatDate", "toDate", "addDays", "subtractDays",
		"diffDays", "getTime", "dateFormat", "addHours", "addMinutes",
	}
}

func (d *DateExtensions) RegisterFunctions(vm *goja.Runtime) error {
	functions := map[string]func(call goja.FunctionCall) goja.Value{
		"now":          d.now(vm),
		"formatDate":   d.formatDate(vm),
		"toDate":       d.toDate(vm),
		"addDays":      d.addDays(vm),
		"subtractDays": d.subtractDays(vm),
		"diffDays":     d.diffDays(vm),
		"getTime":      d.getTime(vm),
		"dateFormat":   d.dateFormat(vm),
		"addHours":     d.addHours(vm),
		"addMinutes":   d.addMinutes(vm),
	}

	for name, fn := range functions {
		vm.Set(name, fn)
	}

	return nil
}

func (d *DateExtensions) now(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		return vm.ToValue(float64(time.Now().Unix() * 1000)) // JavaScript timestamp
	}
}

func (d *DateExtensions) formatDate(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("formatDate requires 2 arguments"))
		}

		dateVal := call.Arguments[0]
		format := call.Arguments[1].String()

		var t time.Time
		switch v := dateVal.Export().(type) {
		case float64:
			// JavaScript timestamp (milliseconds)
			t = time.Unix(int64(v)/1000, 0)
		case int64:
			// JavaScript timestamp (milliseconds) as int64
			t = time.Unix(v/1000, 0)
		case string:
			var err error
			t, err = time.Parse(time.RFC3339, v)
			if err != nil {
				panic(vm.ToValue("Invalid date string"))
			}
		default:
			panic(vm.ToValue("First argument must be a date or timestamp"))
		}

		// Convert common date format patterns
		goFormat := d.convertDateFormat(format)
		return vm.ToValue(t.Format(goFormat))
	}
}

func (d *DateExtensions) toDate(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("toDate requires 1 argument"))
		}

		val := call.Arguments[0].Export()

		switch v := val.(type) {
		case string:
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				// Try other common formats
				formats := []string{
					"2006-01-02",
					"2006-01-02 15:04:05",
					"01/02/2006",
					"01-02-2006",
				}

				for _, format := range formats {
					if t, err = time.Parse(format, v); err == nil {
						break
					}
				}

				if err != nil {
					panic(vm.ToValue("Invalid date string"))
				}
			}
			return vm.ToValue(t.Unix() * 1000)
		case float64:
			// Already a timestamp
			return vm.ToValue(v)
		case int64:
			// Already a timestamp as int64
			return vm.ToValue(float64(v))
		default:
			panic(vm.ToValue("Argument must be a string or number"))
		}
	}
}

func (d *DateExtensions) addDays(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("addDays requires 2 arguments"))
		}

		timestamp := call.Arguments[0].ToFloat()
		days := call.Arguments[1].ToFloat()

		t := time.Unix(int64(timestamp)/1000, 0)
		newTime := t.AddDate(0, 0, int(days))
		return vm.ToValue(newTime.Unix() * 1000)
	}
}

func (d *DateExtensions) subtractDays(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("subtractDays requires 2 arguments"))
		}

		timestamp := call.Arguments[0].ToFloat()
		days := call.Arguments[1].ToFloat()

		t := time.Unix(int64(timestamp)/1000, 0)
		newTime := t.AddDate(0, 0, -int(days))
		return vm.ToValue(newTime.Unix() * 1000)
	}
}

func (d *DateExtensions) diffDays(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("diffDays requires 2 arguments"))
		}

		timestamp1 := call.Arguments[0].ToFloat()
		timestamp2 := call.Arguments[1].ToFloat()

		t1 := time.Unix(int64(timestamp1)/1000, 0)
		t2 := time.Unix(int64(timestamp2)/1000, 0)

		diff := t2.Sub(t1)
		days := diff.Hours() / 24

		return vm.ToValue(days)
	}
}

func (d *DateExtensions) getTime(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("getTime requires 1 argument"))
		}

		// Convert date to timestamp
		return d.toDate(vm)(call)
	}
}

func (d *DateExtensions) dateFormat(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return d.formatDate(vm) // Alias for formatDate
}

func (d *DateExtensions) addHours(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("addHours requires 2 arguments"))
		}

		timestamp := call.Arguments[0].ToFloat()
		hours := call.Arguments[1].ToFloat()

		t := time.Unix(int64(timestamp)/1000, 0)
		newTime := t.Add(time.Duration(hours) * time.Hour)
		return vm.ToValue(newTime.Unix() * 1000)
	}
}

func (d *DateExtensions) addMinutes(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("addMinutes requires 2 arguments"))
		}

		timestamp := call.Arguments[0].ToFloat()
		minutes := call.Arguments[1].ToFloat()

		t := time.Unix(int64(timestamp)/1000, 0)
		newTime := t.Add(time.Duration(minutes) * time.Minute)
		return vm.ToValue(newTime.Unix() * 1000)
	}
}

func (d *DateExtensions) convertDateFormat(format string) string {
	// Convert common date format patterns to Go format
	replacements := map[string]string{
		"yyyy": "2006",
		"MM":   "01",
		"dd":   "02",
		"HH":   "15",
		"mm":   "04",
		"ss":   "05",
	}

	result := format
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	return result
}

func (d *DateExtensions) GetFunctionHelp(name string) *FunctionHelp {
	helpMap := map[string]*FunctionHelp{
		"now": {
			Name:        "now",
			Description: "Returns the current timestamp",
			Parameters:  []ParameterInfo{},
			ReturnType:  "number",
			Examples:    []string{`now()`},
		},
		// Add more help entries...
	}

	if help, exists := helpMap[name]; exists {
		return help
	}

	return &FunctionHelp{
		Name:        name,
		Description: "Date function",
		ReturnType:  "number",
	}
}

// LogicExtensions provides logical functions
type LogicExtensions struct{}

func (l *LogicExtensions) GetCategory() string {
	return "logic"
}

func (l *LogicExtensions) GetFunctionNames() []string {
	return []string{
		"if", "and", "or", "not", "isEmpty", "isNotEmpty",
		"equal", "notEqual", "greaterThan", "lessThan",
	}
}

func (l *LogicExtensions) RegisterFunctions(vm *goja.Runtime) error {
	functions := map[string]func(call goja.FunctionCall) goja.Value{
		// Note: 'if' is registered separately due to JS reserved keyword
		"and":         l.and(vm),
		"or":          l.or(vm),
		"not":         l.not(vm),
		"isEmpty":     l.isEmpty(vm),
		"isNotEmpty":  l.isNotEmpty(vm),
		"equal":       l.equal(vm),
		"notEqual":    l.notEqual(vm),
		"greaterThan": l.greaterThan(vm),
		"lessThan":    l.lessThan(vm),
	}

	for name, fn := range functions {
		vm.Set(name, fn)
	}

	// Register 'if' function using alias method
	ifFunc := l.ifFunc(vm)
	vm.Set("_if", ifFunc)

	// Set 'if' function on the global object using this instead of globalThis
	vm.RunString(`this['if'] = this._if;`)

	return nil
}

func (l *LogicExtensions) ifFunc(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			panic(vm.ToValue("if requires 3 arguments"))
		}

		condition := call.Arguments[0]
		trueValue := call.Arguments[1]
		falseValue := call.Arguments[2]

		if l.isTruthy(condition.Export()) {
			return trueValue
		}
		return falseValue
	}
}

func (l *LogicExtensions) and(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			if !l.isTruthy(arg.Export()) {
				return vm.ToValue(false)
			}
		}
		return vm.ToValue(true)
	}
}

func (l *LogicExtensions) or(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			if l.isTruthy(arg.Export()) {
				return vm.ToValue(true)
			}
		}
		return vm.ToValue(false)
	}
}

func (l *LogicExtensions) not(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("not requires 1 argument"))
		}

		return vm.ToValue(!l.isTruthy(call.Arguments[0].Export()))
	}
}

func (l *LogicExtensions) isEmpty(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return vm.ToValue(true)
		}

		val := call.Arguments[0].Export()
		return vm.ToValue(l.isEmptyValue(val))
	}
}

func (l *LogicExtensions) isNotEmpty(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return vm.ToValue(false)
		}

		val := call.Arguments[0].Export()
		return vm.ToValue(!l.isEmptyValue(val))
	}
}

func (l *LogicExtensions) equal(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("equal requires 2 arguments"))
		}

		val1 := call.Arguments[0].Export()
		val2 := call.Arguments[1].Export()

		return vm.ToValue(fmt.Sprintf("%v", val1) == fmt.Sprintf("%v", val2))
	}
}

func (l *LogicExtensions) notEqual(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("notEqual requires 2 arguments"))
		}

		val1 := call.Arguments[0].Export()
		val2 := call.Arguments[1].Export()

		return vm.ToValue(fmt.Sprintf("%v", val1) != fmt.Sprintf("%v", val2))
	}
}

func (l *LogicExtensions) greaterThan(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("greaterThan requires 2 arguments"))
		}

		val1 := call.Arguments[0].ToFloat()
		val2 := call.Arguments[1].ToFloat()

		return vm.ToValue(val1 > val2)
	}
}

func (l *LogicExtensions) lessThan(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("lessThan requires 2 arguments"))
		}

		val1 := call.Arguments[0].ToFloat()
		val2 := call.Arguments[1].ToFloat()

		return vm.ToValue(val1 < val2)
	}
}

func (l *LogicExtensions) isTruthy(val interface{}) bool {
	if val == nil {
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	default:
		return true
	}
}

func (l *LogicExtensions) isEmptyValue(val interface{}) bool {
	if val == nil {
		return true
	}

	switch v := val.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		// Use reflection for other slice/map types
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return rv.Len() == 0
		default:
			return false
		}
	}
}

func (l *LogicExtensions) GetFunctionHelp(name string) *FunctionHelp {
	helpMap := map[string]*FunctionHelp{
		"if": {
			Name:        "if",
			Description: "Returns one value or another based on a condition",
			Parameters: []ParameterInfo{
				{Name: "condition", Type: "boolean", Required: true, Description: "The condition to test"},
				{Name: "trueValue", Type: "any", Required: true, Description: "Value to return if true"},
				{Name: "falseValue", Type: "any", Required: true, Description: "Value to return if false"},
			},
			ReturnType: "any",
			Examples:   []string{`if($json.age > 18, 'adult', 'minor')`},
		},
		// Add more help entries...
	}

	if help, exists := helpMap[name]; exists {
		return help
	}

	return &FunctionHelp{
		Name:        name,
		Description: "Logic function",
		ReturnType:  "boolean",
	}
}

// ObjectExtensions provides object manipulation functions
type ObjectExtensions struct{}

func (o *ObjectExtensions) GetCategory() string {
	return "object"
}

func (o *ObjectExtensions) GetFunctionNames() []string {
	return []string{
		"keys", "values", "merge", "pick", "omit", "has",
	}
}

func (o *ObjectExtensions) RegisterFunctions(vm *goja.Runtime) error {
	functions := map[string]func(call goja.FunctionCall) goja.Value{
		"keys":   o.keys(vm),
		"values": o.values(vm),
		"merge":  o.merge(vm),
		"pick":   o.pick(vm),
		"omit":   o.omit(vm),
		"has":    o.has(vm),
	}

	for name, fn := range functions {
		vm.Set(name, fn)
	}

	return nil
}

func (o *ObjectExtensions) keys(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("keys requires 1 argument"))
		}

		obj := call.Arguments[0].Export()
		if objMap, ok := obj.(map[string]interface{}); ok {
			var keys []string
			for key := range objMap {
				keys = append(keys, key)
			}
			return vm.ToValue(keys)
		}

		panic(vm.ToValue("Argument must be an object"))
	}
}

func (o *ObjectExtensions) values(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("values requires 1 argument"))
		}

		obj := call.Arguments[0].Export()
		if objMap, ok := obj.(map[string]interface{}); ok {
			var values []interface{}
			for _, value := range objMap {
				values = append(values, value)
			}
			return vm.ToValue(values)
		}

		panic(vm.ToValue("Argument must be an object"))
	}
}

func (o *ObjectExtensions) merge(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		result := make(map[string]interface{})

		for _, arg := range call.Arguments {
			obj := arg.Export()
			if objMap, ok := obj.(map[string]interface{}); ok {
				for key, value := range objMap {
					result[key] = value
				}
			}
		}

		return vm.ToValue(result)
	}
}

func (o *ObjectExtensions) pick(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("pick requires 2 arguments"))
		}

		obj := call.Arguments[0].Export()
		keysArg := call.Arguments[1].Export()

		if objMap, ok := obj.(map[string]interface{}); ok {
			result := make(map[string]interface{})

			// Handle keys as array or single key
			if keysArr, ok := keysArg.([]interface{}); ok {
				for _, keyVal := range keysArr {
					if key, ok := keyVal.(string); ok {
						if value, exists := objMap[key]; exists {
							result[key] = value
						}
					}
				}
			} else if key, ok := keysArg.(string); ok {
				if value, exists := objMap[key]; exists {
					result[key] = value
				}
			}

			return vm.ToValue(result)
		}

		panic(vm.ToValue("First argument must be an object"))
	}
}

func (o *ObjectExtensions) omit(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("omit requires 2 arguments"))
		}

		obj := call.Arguments[0].Export()
		keysArg := call.Arguments[1].Export()

		if objMap, ok := obj.(map[string]interface{}); ok {
			result := make(map[string]interface{})

			// Copy all keys first
			for key, value := range objMap {
				result[key] = value
			}

			// Remove specified keys
			if keysArr, ok := keysArg.([]interface{}); ok {
				for _, keyVal := range keysArr {
					if key, ok := keyVal.(string); ok {
						delete(result, key)
					}
				}
			} else if key, ok := keysArg.(string); ok {
				delete(result, key)
			}

			return vm.ToValue(result)
		}

		panic(vm.ToValue("First argument must be an object"))
	}
}

func (o *ObjectExtensions) has(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("has requires 2 arguments"))
		}

		obj := call.Arguments[0].Export()
		key := call.Arguments[1].String()

		if objMap, ok := obj.(map[string]interface{}); ok {
			_, exists := objMap[key]
			return vm.ToValue(exists)
		}

		return vm.ToValue(false)
	}
}

func (o *ObjectExtensions) GetFunctionHelp(name string) *FunctionHelp {
	return &FunctionHelp{
		Name:        name,
		Description: "Object function",
		ReturnType:  "any",
	}
}

// UtilityExtensions provides utility functions
type UtilityExtensions struct{}

func (u *UtilityExtensions) GetCategory() string {
	return "utility"
}

func (u *UtilityExtensions) GetFunctionNames() []string {
	return []string{
		"toJson", "fromJson", "uuid", "sleep", "typeOf", "randomString",
	}
}

func (u *UtilityExtensions) RegisterFunctions(vm *goja.Runtime) error {
	functions := map[string]func(call goja.FunctionCall) goja.Value{
		"toJson":       u.toJson(vm),
		"fromJson":     u.fromJson(vm),
		"uuid":         u.uuid(vm),
		"typeOf":       u.typeOf(vm),
		"randomString": u.randomString(vm),
	}

	for name, fn := range functions {
		vm.Set(name, fn)
	}

	return nil
}

func (u *UtilityExtensions) toJson(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("toJson requires 1 argument"))
		}

		// Use JSON.stringify equivalent
		jsonStr := vm.Get("JSON").(*goja.Object).Get("stringify")
		stringifyFunc, ok := goja.AssertFunction(jsonStr)
		if !ok {
			panic(vm.ToValue("Failed to stringify object"))
		}

		result, err := stringifyFunc(goja.Undefined(), call.Arguments[0])
		if err != nil {
			panic(vm.ToValue("Failed to stringify object"))
		}

		return result
	}
}

func (u *UtilityExtensions) fromJson(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("fromJson requires 1 argument"))
		}

		jsonStr := call.Arguments[0].String()

		// Use JSON.parse equivalent
		jsonParse := vm.Get("JSON").(*goja.Object).Get("parse")
		parseFunc, ok := goja.AssertFunction(jsonParse)
		if !ok {
			panic(vm.ToValue("Failed to parse JSON string"))
		}

		result, err := parseFunc(goja.Undefined(), vm.ToValue(jsonStr))
		if err != nil {
			panic(vm.ToValue("Failed to parse JSON string"))
		}

		return result
	}
}

func (u *UtilityExtensions) uuid(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		// Simple UUID v4 generation
		return vm.ToValue(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			rand.Uint32(),
			rand.Uint32()&0xffff,
			(rand.Uint32()&0x0fff)|0x4000,
			(rand.Uint32()&0x3fff)|0x8000,
			rand.Uint64()&0xffffffffffff,
		))
	}
}

func (u *UtilityExtensions) typeOf(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			return vm.ToValue("undefined")
		}

		val := call.Arguments[0].Export()

		switch val.(type) {
		case nil:
			return vm.ToValue("null")
		case bool:
			return vm.ToValue("boolean")
		case int, int32, int64, float32, float64:
			return vm.ToValue("number")
		case string:
			return vm.ToValue("string")
		case []interface{}:
			return vm.ToValue("array")
		case map[string]interface{}:
			return vm.ToValue("object")
		default:
			return vm.ToValue("object")
		}
	}
}

func (u *UtilityExtensions) randomString(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		length := 8
		if len(call.Arguments) > 0 {
			length = int(call.Arguments[0].ToInteger())
		}
		if length <= 0 {
			length = 8
		}

		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		result := make([]byte, length)
		for i := range result {
			result[i] = charset[rand.Intn(len(charset))]
		}
		return vm.ToValue(string(result))
	}
}

func (u *UtilityExtensions) GetFunctionHelp(name string) *FunctionHelp {
	return &FunctionHelp{
		Name:        name,
		Description: "Utility function",
		ReturnType:  "any",
	}
}
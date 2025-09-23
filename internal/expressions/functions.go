package expressions

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"math"
	"net/url"
	"regexp"
	"strings"

	"github.com/dop251/goja"
)

// FunctionRegistry manages all built-in functions for n8n expressions
type FunctionRegistry struct {
	extensions map[string]ExtensionProvider
}

// ExtensionProvider interface for function extension categories
type ExtensionProvider interface {
	GetCategory() string
	GetFunctionNames() []string
	RegisterFunctions(vm *goja.Runtime) error
	GetFunctionHelp(name string) *FunctionHelp
}

// FunctionHelp provides help information for functions
type FunctionHelp struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []ParameterInfo `json:"parameters"`
	ReturnType  string          `json:"returnType"`
	Examples    []string        `json:"examples"`
}

// ParameterInfo describes a function parameter
type ParameterInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// NewFunctionRegistry creates a new function registry with all n8n built-in functions
func NewFunctionRegistry() *FunctionRegistry {
	registry := &FunctionRegistry{
		extensions: make(map[string]ExtensionProvider),
	}

	// Register all extension categories
	registry.extensions["string"] = &StringExtensions{}
	registry.extensions["math"] = &MathExtensions{}
	registry.extensions["array"] = &ArrayExtensions{}
	registry.extensions["object"] = &ObjectExtensions{}
	registry.extensions["date"] = &DateExtensions{}
	registry.extensions["logic"] = &LogicExtensions{}
	registry.extensions["utility"] = &UtilityExtensions{}

	return registry
}

// RegisterAllExtensions registers all functions with the Goja runtime
func (r *FunctionRegistry) RegisterAllExtensions(vm *goja.Runtime) error {
	for _, ext := range r.extensions {
		if err := ext.RegisterFunctions(vm); err != nil {
			return fmt.Errorf("failed to register %s functions: %w", ext.GetCategory(), err)
		}
	}
	return nil
}

// GetAllFunctionHelp returns help for all registered functions
func (r *FunctionRegistry) GetAllFunctionHelp() map[string]*FunctionHelp {
	help := make(map[string]*FunctionHelp)

	for _, ext := range r.extensions {
		for _, funcName := range ext.GetFunctionNames() {
			help[funcName] = ext.GetFunctionHelp(funcName)
		}
	}

	return help
}

// StringExtensions provides string manipulation functions
type StringExtensions struct{}

func (s *StringExtensions) GetCategory() string {
	return "string"
}

func (s *StringExtensions) GetFunctionNames() []string {
	return []string{
		"split", "join", "substring", "substr", "replace", "replaceAll",
		"trim", "trimStart", "trimEnd", "toLowerCase", "toUpperCase",
		"charAt", "charCodeAt", "indexOf", "lastIndexOf", "includes",
		"startsWith", "endsWith", "padStart", "padEnd", "repeat", "slice",
		"base64Encode", "base64Decode", "urlEncode", "urlDecode",
		"htmlEncode", "htmlDecode", "md5", "sha1", "sha256", "sha512",
		"isEmail", "isUrl", "isDomain", "stripTags", "extractDomain",
		"extractUrl", "toTitleCase", "toCamelCase", "toSnakeCase", "toKebabCase",
	}
}

func (s *StringExtensions) RegisterFunctions(vm *goja.Runtime) error {
	functions := map[string]func(call goja.FunctionCall) goja.Value{
		// Core string functions
		"split":           s.split(vm),
		"join":            s.join(vm),
		"substring":       s.substring(vm),
		"substr":          s.substr(vm),
		"replace":         s.replace(vm),
		"replaceAll":      s.replaceAll(vm),
		"trim":            s.trim(vm),
		"trimStart":       s.trimStart(vm),
		"trimEnd":         s.trimEnd(vm),
		"toLowerCase":     s.toLowerCase(vm),
		"toUpperCase":     s.toUpperCase(vm),
		// n8n aliases
		"lowercase":       s.toLowerCase(vm),
		"uppercase":       s.toUpperCase(vm),
		"lower":           s.toLowerCase(vm),
		"upper":           s.toUpperCase(vm),
		"charAt":          s.charAt(vm),
		"charCodeAt":      s.charCodeAt(vm),
		"indexOf":         s.indexOf(vm),
		"lastIndexOf":     s.lastIndexOf(vm),
		"includes":        s.includes(vm),
		"startsWith":      s.startsWith(vm),
		"endsWith":        s.endsWith(vm),
		"padStart":        s.padStart(vm),
		"padEnd":          s.padEnd(vm),
		"repeat":          s.repeat(vm),
		"slice":           s.slice(vm),

		// Encoding functions
		"base64Encode":    s.base64Encode(vm),
		"base64Decode":    s.base64Decode(vm),
		"urlEncode":       s.urlEncode(vm),
		"urlDecode":       s.urlDecode(vm),
		"htmlEncode":      s.htmlEncode(vm),
		"htmlDecode":      s.htmlDecode(vm),

		// Hashing functions
		"md5":             s.md5(vm),
		"sha1":            s.sha1(vm),
		"sha256":          s.sha256(vm),
		"sha512":          s.sha512(vm),

		// Validation functions
		"isEmail":         s.isEmail(vm),
		"isUrl":           s.isUrl(vm),
		"isDomain":        s.isDomain(vm),

		// Utility functions
		"stripTags":       s.stripTags(vm),
		"extractDomain":   s.extractDomain(vm),
		"extractUrl":      s.extractUrl(vm),

		// Case conversion
		"toTitleCase":     s.toTitleCase(vm),
		"toCamelCase":     s.toCamelCase(vm),
		"toSnakeCase":     s.toSnakeCase(vm),
		"toKebabCase":     s.toKebabCase(vm),
	}

	for name, fn := range functions {
		vm.Set(name, fn)
	}

	return nil
}

// Core string functions implementation
func (s *StringExtensions) split(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("split requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		separator := call.Arguments[1].String()

		var limit int = -1
		if len(call.Arguments) > 2 {
			limit = int(call.Arguments[2].ToInteger())
		}

		var parts []string
		if separator == "" {
			// Split each character
			parts = strings.Split(str, "")
		} else {
			parts = strings.Split(str, separator)
		}

		if limit > 0 && len(parts) > limit {
			parts = parts[:limit]
		}

		// Convert to []interface{} for n8n compatibility
		result := make([]interface{}, len(parts))
		for i, part := range parts {
			result[i] = part
		}

		return vm.ToValue(result)
	}
}

func (s *StringExtensions) join(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("join requires 2 arguments"))
		}

		arrVal := call.Arguments[0]
		separator := call.Arguments[1].String()

		// Handle array input
		if arrVal.ExportType().Kind().String() == "slice" {
			arr := arrVal.Export().([]interface{})
			strArr := make([]string, len(arr))
			for i, v := range arr {
				strArr[i] = fmt.Sprintf("%v", v)
			}
			return vm.ToValue(strings.Join(strArr, separator))
		}

		panic(vm.NewTypeError("first argument must be an array"))
	}
}

func (s *StringExtensions) substring(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("substring requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		start := int(call.Arguments[1].ToInteger())

		var end int = len(str)
		if len(call.Arguments) > 2 {
			end = int(call.Arguments[2].ToInteger())
		}

		// Ensure bounds
		if start < 0 {
			start = 0
		}
		if end > len(str) {
			end = len(str)
		}
		if start > end {
			start, end = end, start
		}

		return vm.ToValue(str[start:end])
	}
}

func (s *StringExtensions) replace(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			panic(vm.NewTypeError("replace requires 3 arguments"))
		}

		str := call.Arguments[0].String()
		search := call.Arguments[1].String()
		replacement := call.Arguments[2].String()

		// Replace first occurrence only
		return vm.ToValue(strings.Replace(str, search, replacement, 1))
	}
}

func (s *StringExtensions) replaceAll(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			panic(vm.NewTypeError("replaceAll requires 3 arguments"))
		}

		str := call.Arguments[0].String()
		search := call.Arguments[1].String()
		replacement := call.Arguments[2].String()

		// Replace all occurrences
		return vm.ToValue(strings.ReplaceAll(str, search, replacement))
	}
}

func (s *StringExtensions) trim(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("trim requires 1 argument"))
		}

		str := call.Arguments[0].String()
		return vm.ToValue(strings.TrimSpace(str))
	}
}

func (s *StringExtensions) toLowerCase(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("toLowerCase requires 1 argument"))
		}

		str := call.Arguments[0].String()
		return vm.ToValue(strings.ToLower(str))
	}
}

func (s *StringExtensions) toUpperCase(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("toUpperCase requires 1 argument"))
		}

		str := call.Arguments[0].String()
		return vm.ToValue(strings.ToUpper(str))
	}
}

// Encoding functions
func (s *StringExtensions) base64Encode(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("base64Encode requires 1 argument"))
		}

		str := call.Arguments[0].String()
		encoded := base64.StdEncoding.EncodeToString([]byte(str))
		return vm.ToValue(encoded)
	}
}

func (s *StringExtensions) base64Decode(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("base64Decode requires 1 argument"))
		}

		str := call.Arguments[0].String()
		decoded, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			panic(vm.ToValue("DecodeError: Invalid base64 string"))
		}
		return vm.ToValue(string(decoded))
	}
}

func (s *StringExtensions) urlEncode(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("urlEncode requires 1 argument"))
		}

		str := call.Arguments[0].String()
		encoded := url.QueryEscape(str)
		return vm.ToValue(encoded)
	}
}

func (s *StringExtensions) urlDecode(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("urlDecode requires 1 argument"))
		}

		str := call.Arguments[0].String()
		decoded, err := url.QueryUnescape(str)
		if err != nil {
			panic(vm.ToValue("DecodeError: Invalid URL encoded string"))
		}
		return vm.ToValue(decoded)
	}
}

func (s *StringExtensions) htmlEncode(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("htmlEncode requires 1 argument"))
		}

		str := call.Arguments[0].String()
		encoded := html.EscapeString(str)
		return vm.ToValue(encoded)
	}
}

func (s *StringExtensions) htmlDecode(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("htmlDecode requires 1 argument"))
		}

		str := call.Arguments[0].String()
		decoded := html.UnescapeString(str)
		return vm.ToValue(decoded)
	}
}

// Hashing functions
func (s *StringExtensions) md5(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("md5 requires 1 argument"))
		}

		str := call.Arguments[0].String()
		hash := md5.Sum([]byte(str))
		return vm.ToValue(hex.EncodeToString(hash[:]))
	}
}

func (s *StringExtensions) sha1(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("sha1 requires 1 argument"))
		}

		str := call.Arguments[0].String()
		hash := sha1.Sum([]byte(str))
		return vm.ToValue(hex.EncodeToString(hash[:]))
	}
}

func (s *StringExtensions) sha256(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("sha256 requires 1 argument"))
		}

		str := call.Arguments[0].String()
		hash := sha256.Sum256([]byte(str))
		return vm.ToValue(hex.EncodeToString(hash[:]))
	}
}

func (s *StringExtensions) sha512(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("sha512 requires 1 argument"))
		}

		str := call.Arguments[0].String()
		hash := sha512.Sum512([]byte(str))
		return vm.ToValue(hex.EncodeToString(hash[:]))
	}
}

// Validation functions
func (s *StringExtensions) isEmail(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("isEmail requires 1 argument"))
		}

		str := call.Arguments[0].String()
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		return vm.ToValue(emailRegex.MatchString(str))
	}
}

func (s *StringExtensions) isUrl(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("isUrl requires 1 argument"))
		}

		str := call.Arguments[0].String()
		_, err := url.ParseRequestURI(str)
		return vm.ToValue(err == nil)
	}
}

func (s *StringExtensions) isDomain(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("isDomain requires 1 argument"))
		}

		str := call.Arguments[0].String()
		domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
		return vm.ToValue(domainRegex.MatchString(str))
	}
}

// Utility functions
func (s *StringExtensions) stripTags(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("stripTags requires 1 argument"))
		}

		str := call.Arguments[0].String()
		tagRegex := regexp.MustCompile(`<[^>]*>`)
		return vm.ToValue(tagRegex.ReplaceAllString(str, ""))
	}
}

func (s *StringExtensions) extractDomain(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("extractDomain requires 1 argument"))
		}

		str := call.Arguments[0].String()
		u, err := url.Parse(str)
		if err != nil {
			return vm.ToValue("")
		}
		return vm.ToValue(u.Hostname())
	}
}

func (s *StringExtensions) extractUrl(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("extractUrl requires 1 argument"))
		}

		str := call.Arguments[0].String()
		urlRegex := regexp.MustCompile(`https?://[^\s]+`)
		matches := urlRegex.FindAllString(str, -1)
		return vm.ToValue(matches)
	}
}

// Case conversion functions
func (s *StringExtensions) toTitleCase(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("toTitleCase requires 1 argument"))
		}

		str := call.Arguments[0].String()
		return vm.ToValue(strings.Title(strings.ToLower(str)))
	}
}

func (s *StringExtensions) toCamelCase(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("toCamelCase requires 1 argument"))
		}

		str := call.Arguments[0].String()
		words := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(str, "_", " "), "-", " "))
		if len(words) == 0 {
			return vm.ToValue("")
		}

		result := strings.ToLower(words[0])
		for i := 1; i < len(words); i++ {
			result += strings.Title(strings.ToLower(words[i]))
		}
		return vm.ToValue(result)
	}
}

func (s *StringExtensions) toSnakeCase(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("toSnakeCase requires 1 argument"))
		}

		str := call.Arguments[0].String()
		words := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(str, "_", " "), "-", " "))
		result := strings.ToLower(strings.Join(words, "_"))
		return vm.ToValue(result)
	}
}

func (s *StringExtensions) toKebabCase(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("toKebabCase requires 1 argument"))
		}

		str := call.Arguments[0].String()
		words := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(str, "_", " "), "-", " "))
		result := strings.ToLower(strings.Join(words, "-"))
		return vm.ToValue(result)
	}
}

// Additional helper functions for substring, charAt, etc.
func (s *StringExtensions) substr(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("substr requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		start := int(call.Arguments[1].ToInteger())

		var length int = len(str) - start
		if len(call.Arguments) > 2 {
			length = int(call.Arguments[2].ToInteger())
		}

		if start < 0 {
			start = 0
		}
		if start >= len(str) {
			return vm.ToValue("")
		}

		end := start + length
		if end > len(str) {
			end = len(str)
		}

		return vm.ToValue(str[start:end])
	}
}

func (s *StringExtensions) charAt(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("charAt requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		index := int(call.Arguments[1].ToInteger())

		if index < 0 || index >= len(str) {
			return vm.ToValue("")
		}

		return vm.ToValue(string(str[index]))
	}
}

func (s *StringExtensions) charCodeAt(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("charCodeAt requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		index := int(call.Arguments[1].ToInteger())

		if index < 0 || index >= len(str) {
			return vm.ToValue(math.NaN())
		}

		return vm.ToValue(int(str[index]))
	}
}

func (s *StringExtensions) indexOf(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("indexOf requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		search := call.Arguments[1].String()

		var start int = 0
		if len(call.Arguments) > 2 {
			start = int(call.Arguments[2].ToInteger())
		}

		if start < 0 {
			start = 0
		}
		if start >= len(str) {
			return vm.ToValue(-1)
		}

		index := strings.Index(str[start:], search)
		if index == -1 {
			return vm.ToValue(-1)
		}

		return vm.ToValue(start + index)
	}
}

func (s *StringExtensions) lastIndexOf(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("lastIndexOf requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		search := call.Arguments[1].String()

		return vm.ToValue(strings.LastIndex(str, search))
	}
}

func (s *StringExtensions) includes(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("includes requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		search := call.Arguments[1].String()

		return vm.ToValue(strings.Contains(str, search))
	}
}

func (s *StringExtensions) startsWith(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("startsWith requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		prefix := call.Arguments[1].String()

		return vm.ToValue(strings.HasPrefix(str, prefix))
	}
}

func (s *StringExtensions) endsWith(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("endsWith requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		suffix := call.Arguments[1].String()

		return vm.ToValue(strings.HasSuffix(str, suffix))
	}
}

func (s *StringExtensions) padStart(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("padStart requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		targetLength := int(call.Arguments[1].ToInteger())

		padString := " "
		if len(call.Arguments) > 2 {
			padString = call.Arguments[2].String()
		}

		if len(str) >= targetLength {
			return vm.ToValue(str)
		}

		padLength := targetLength - len(str)
		pad := strings.Repeat(padString, (padLength/len(padString))+1)[:padLength]

		return vm.ToValue(pad + str)
	}
}

func (s *StringExtensions) padEnd(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("padEnd requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		targetLength := int(call.Arguments[1].ToInteger())

		padString := " "
		if len(call.Arguments) > 2 {
			padString = call.Arguments[2].String()
		}

		if len(str) >= targetLength {
			return vm.ToValue(str)
		}

		padLength := targetLength - len(str)
		pad := strings.Repeat(padString, (padLength/len(padString))+1)[:padLength]

		return vm.ToValue(str + pad)
	}
}

func (s *StringExtensions) repeat(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("repeat requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		count := int(call.Arguments[1].ToInteger())

		if count < 0 {
			panic(vm.ToValue("RangeError: repeat count must be non-negative"))
		}

		return vm.ToValue(strings.Repeat(str, count))
	}
}

func (s *StringExtensions) slice(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("slice requires 2 arguments"))
		}

		str := call.Arguments[0].String()
		start := int(call.Arguments[1].ToInteger())

		var end int = len(str)
		if len(call.Arguments) > 2 {
			end = int(call.Arguments[2].ToInteger())
		}

		// Handle negative indices
		if start < 0 {
			start = len(str) + start
		}
		if end < 0 {
			end = len(str) + end
		}

		// Ensure bounds
		if start < 0 {
			start = 0
		}
		if end > len(str) {
			end = len(str)
		}
		if start > end {
			return vm.ToValue("")
		}

		return vm.ToValue(str[start:end])
	}
}

func (s *StringExtensions) trimStart(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("trimStart requires 1 argument"))
		}

		str := call.Arguments[0].String()
		return vm.ToValue(strings.TrimLeftFunc(str, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r'
		}))
	}
}

func (s *StringExtensions) trimEnd(vm *goja.Runtime) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewTypeError("trimEnd requires 1 argument"))
		}

		str := call.Arguments[0].String()
		return vm.ToValue(strings.TrimRightFunc(str, func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r'
		}))
	}
}

func (s *StringExtensions) GetFunctionHelp(name string) *FunctionHelp {
	helpMap := map[string]*FunctionHelp{
		"split": {
			Name:        "split",
			Description: "Splits a string into an array of substrings",
			Parameters: []ParameterInfo{
				{Name: "string", Type: "string", Required: true, Description: "The string to split"},
				{Name: "separator", Type: "string", Required: true, Description: "The separator to split on"},
				{Name: "limit", Type: "number", Required: false, Description: "Maximum number of parts"},
			},
			ReturnType: "array",
			Examples:   []string{`split("hello world", " ")`, `split("a,b,c", ",", 2)`},
		},
		// Add more help entries for other functions...
	}

	if help, exists := helpMap[name]; exists {
		return help
	}

	return &FunctionHelp{
		Name:        name,
		Description: "String function",
		ReturnType:  "string",
	}
}
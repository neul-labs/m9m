package runtime

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/n8n-go/n8n-go/internal/expressions"
	"github.com/n8n-go/n8n-go/pkg/model"
)

type JavaScriptRuntime struct {
	mu              sync.RWMutex
	vm              *goja.Runtime
	registry        *require.Registry
	npmCache        map[string]*NpmPackage
	nodeModulesPath string
	globalModules   map[string]interface{}
	n8nHelpers      *N8nHelpers
	executionContext *ExecutionContext
}

type NpmPackage struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Main         string                 `json:"main"`
	Dependencies map[string]string      `json:"dependencies"`
	CachePath    string                 `json:"cache_path"`
	LoadedAt     time.Time              `json:"loaded_at"`
	Module       interface{}            `json:"-"`
}

type ExecutionContext struct {
	WorkflowID   string                 `json:"workflow_id"`
	ExecutionID  string                 `json:"execution_id"`
	NodeID       string                 `json:"node_id"`
	ItemIndex    int                    `json:"item_index"`
	RunIndex     int                    `json:"run_index"`
	Mode         string                 `json:"mode"` // "manual", "trigger", "webhook", etc.
	Timezone     string                 `json:"timezone"`
	Variables    map[string]interface{} `json:"variables"`
	Credentials  map[string]interface{} `json:"credentials"`
}

type N8nHelpers struct {
	runtime *JavaScriptRuntime
}

type NodeJSGlobals struct {
	Process   *ProcessObject   `json:"process"`
	Buffer    *BufferObject    `json:"Buffer"`
	Global    interface{}      `json:"global"`
	Console   interface{}      `json:"console"`
	SetTimeout func(goja.FunctionCall) goja.Value `json:"setTimeout"`
	ClearTimeout func(goja.FunctionCall) goja.Value `json:"clearTimeout"`
	SetInterval func(goja.FunctionCall) goja.Value `json:"setInterval"`
	ClearInterval func(goja.FunctionCall) goja.Value `json:"clearInterval"`
}

type ProcessObject struct {
	Env     map[string]string `json:"env"`
	Version string           `json:"version"`
	Platform string          `json:"platform"`
	Arch    string           `json:"arch"`
	Argv    []string         `json:"argv"`
	Pid     int              `json:"pid"`
}

type BufferObject struct {
	runtime *JavaScriptRuntime
}

func NewJavaScriptRuntime(nodeModulesPath string) *JavaScriptRuntime {
	vm := goja.New()
	registry := new(require.Registry)

	js := &JavaScriptRuntime{
		vm:              vm,
		registry:        registry,
		npmCache:        make(map[string]*NpmPackage),
		nodeModulesPath: nodeModulesPath,
		globalModules:   make(map[string]interface{}),
		executionContext: &ExecutionContext{
			Variables:   make(map[string]interface{}),
			Credentials: make(map[string]interface{}),
		},
	}

	js.n8nHelpers = &N8nHelpers{runtime: js}

	// Initialize the runtime
	js.initializeNodeJSGlobals()
	js.initializeBuiltinModules()
	js.initializeN8nGlobals()

	return js
}

func (js *JavaScriptRuntime) initializeNodeJSGlobals() {
	// Add console support
	registry := new(require.Registry)
	registry.Enable(js.vm)
	console.Enable(js.vm)

	// Create process object
	process := &ProcessObject{
		Env:      make(map[string]string),
		Version:  "v16.0.0", // Simulate Node.js v16
		Platform: "linux",
		Arch:     "x64",
		Argv:     []string{"node"},
		Pid:      1234,
	}

	// Copy environment variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			process.Env[parts[0]] = parts[1]
		}
	}

	js.vm.Set("process", process)

	// Add Buffer support
	buffer := &BufferObject{runtime: js}
	js.vm.Set("Buffer", buffer)

	// Add global object
	js.vm.Set("global", js.vm.GlobalObject())

	// Add timer functions
	js.vm.Set("setTimeout", js.setTimeout)
	js.vm.Set("clearTimeout", js.clearTimeout)
	js.vm.Set("setInterval", js.setInterval)
	js.vm.Set("clearInterval", js.clearInterval)

	// Add require function
	js.vm.Set("require", js.requireFunction)
}

func (js *JavaScriptRuntime) initializeBuiltinModules() {
	// Core Node.js modules simulation
	modules := map[string]interface{}{
		"util":   js.createUtilModule(),
		"crypto": js.createCryptoModule(),
		"path":   js.createPathModule(),
		"fs":     js.createFsModule(),
		"http":   js.createHttpModule(),
		"https":  js.createHttpsModule(),
		"url":    js.createUrlModule(),
		"querystring": js.createQueryStringModule(),
		"os":     js.createOsModule(),
		"events": js.createEventsModule(),
	}

	for name, module := range modules {
		js.globalModules[name] = module
	}
}

func (js *JavaScriptRuntime) initializeN8nGlobals() {
	// Add n8n-specific global functions and objects
	js.vm.Set("$json", js.createJsonHelper())
	js.vm.Set("$node", js.createNodeHelper())
	js.vm.Set("$parameter", js.createParameterHelper())
	js.vm.Set("$workflow", js.createWorkflowHelper())
	js.vm.Set("$execution", js.createExecutionHelper())
	js.vm.Set("$items", js.createItemsHelper())
	js.vm.Set("$input", js.createInputHelper())
	js.vm.Set("$", js.createDollarHelper())

	// Add n8n helper functions
	js.vm.Set("helpers", js.n8nHelpers)

	// Add moment.js compatibility
	js.vm.Set("moment", js.createMomentHelper())

	// Add lodash compatibility
	js.vm.Set("_", js.createLodashHelper())
}

func (js *JavaScriptRuntime) requireFunction(call goja.FunctionCall) goja.Value {
	moduleName := call.Argument(0).String()

	// Check if it's a built-in module
	if module, exists := js.globalModules[moduleName]; exists {
		return js.vm.ToValue(module)
	}

	// Try to load npm package
	pkg, err := js.LoadNpmPackage(moduleName, "latest")
	if err != nil {
		panic(js.vm.NewGoError(fmt.Errorf("cannot find module '%s': %w", moduleName, err)))
	}

	return js.vm.ToValue(pkg.Module)
}

func (js *JavaScriptRuntime) LoadNpmPackage(name, version string) (*NpmPackage, error) {
	js.mu.Lock()
	defer js.mu.Unlock()

	cacheKey := fmt.Sprintf("%s@%s", name, version)

	// Check cache first
	if pkg, exists := js.npmCache[cacheKey]; exists {
		return pkg, nil
	}

	// Download and install package
	pkg, err := js.downloadNpmPackage(name, version)
	if err != nil {
		return nil, err
	}

	// Load the main module
	mainFile := filepath.Join(pkg.CachePath, pkg.Main)
	moduleCode, err := ioutil.ReadFile(mainFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read main file %s: %w", mainFile, err)
	}

	// Execute module in isolated context
	moduleValue, err := js.executeModuleCode(string(moduleCode), pkg)
	if err != nil {
		return nil, fmt.Errorf("failed to execute module %s: %w", name, err)
	}

	pkg.Module = moduleValue
	pkg.LoadedAt = time.Now()

	// Cache the package
	js.npmCache[cacheKey] = pkg

	return pkg, nil
}

func (js *JavaScriptRuntime) downloadNpmPackage(name, version string) (*NpmPackage, error) {
	// Create cache directory
	cacheDir := filepath.Join(js.nodeModulesPath, name)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	// Download package.json
	packageJsonUrl := fmt.Sprintf("https://registry.npmjs.org/%s/%s", name, version)
	resp, err := http.Get(packageJsonUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var packageInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&packageInfo); err != nil {
		return nil, err
	}

	// Extract package information
	pkg := &NpmPackage{
		Name:         name,
		Version:      version,
		Main:         "index.js",
		Dependencies: make(map[string]string),
		CachePath:    cacheDir,
	}

	if main, ok := packageInfo["main"].(string); ok {
		pkg.Main = main
	}

	if deps, ok := packageInfo["dependencies"].(map[string]interface{}); ok {
		for depName, depVersion := range deps {
			pkg.Dependencies[depName] = depVersion.(string)
		}
	}

	// For now, we'll create a simple mock of the package
	// In a full implementation, you would download the actual tarball
	js.createMockPackage(pkg)

	return pkg, nil
}

func (js *JavaScriptRuntime) createMockPackage(pkg *NpmPackage) {
	var mockCode string

	switch pkg.Name {
	case "axios":
		mockCode = js.createAxiosMock()
	case "lodash":
		mockCode = js.createLodashMock()
	case "moment":
		mockCode = js.createMomentMock()
	case "uuid":
		mockCode = js.createUuidMock()
	case "crypto-js":
		mockCode = js.createCryptoJsMock()
	default:
		// Generic mock
		mockCode = fmt.Sprintf(`
			module.exports = {
				name: '%s',
				version: '%s',
				mock: true
			};
		`, pkg.Name, pkg.Version)
	}

	mainFile := filepath.Join(pkg.CachePath, pkg.Main)
	ioutil.WriteFile(mainFile, []byte(mockCode), 0644)
}

func (js *JavaScriptRuntime) executeModuleCode(code string, pkg *NpmPackage) (interface{}, error) {
	// Create module context
	moduleCode := fmt.Sprintf(`
		(function(exports, require, module, __filename, __dirname) {
			%s
		})
	`, code)

	// Compile the module
	program, err := goja.Compile(pkg.Name, moduleCode, false)
	if err != nil {
		return nil, err
	}

	// Create module object
	moduleObj := js.vm.NewObject()
	exportsObj := js.vm.NewObject()
	moduleObj.Set("exports", exportsObj)

	// Execute the module
	moduleFunc, err := js.vm.RunProgram(program)
	if err != nil {
		return nil, err
	}

	// Call the module function
	if callable, ok := goja.AssertFunction(moduleFunc); ok {
		_, err = callable(goja.Undefined(),
			js.vm.ToValue(exportsObj),
			js.vm.ToValue(js.requireFunction),
			js.vm.ToValue(moduleObj),
			js.vm.ToValue(filepath.Join(pkg.CachePath, pkg.Main)),
			js.vm.ToValue(pkg.CachePath),
		)
		if err != nil {
			return nil, err
		}
	}

	// Return module.exports
	exports := moduleObj.Get("exports")
	return exports.Export(), nil
}

func (js *JavaScriptRuntime) Execute(code string, context *ExecutionContext, items []model.DataItem) (interface{}, error) {
	js.mu.Lock()
	defer js.mu.Unlock()

	// Set execution context
	js.executionContext = context

	// Set current items
	js.vm.Set("$items", js.createItemsHelperWithData(items))
	js.vm.Set("items", items)

	// Execute the code
	result, err := js.vm.RunString(code)
	if err != nil {
		return nil, err
	}

	return result.Export(), nil
}

func (js *JavaScriptRuntime) ExecuteExpression(expression string, context *expressions.ExpressionContext) (interface{}, error) {
	js.mu.Lock()
	defer js.mu.Unlock()

	// Convert expression context to JavaScript variables
	for key, value := range context.GetVariables() {
		js.vm.Set(key, value)
	}

	// Execute the expression
	result, err := js.vm.RunString(expression)
	if err != nil {
		return nil, err
	}

	return result.Export(), nil
}

// Timer functions
func (js *JavaScriptRuntime) setTimeout(call goja.FunctionCall) goja.Value {
	callback := call.Argument(0)
	delay := call.Argument(1).ToInteger()

	go func() {
		time.Sleep(time.Duration(delay) * time.Millisecond)
		if callable, ok := goja.AssertFunction(callback); ok {
			callable(goja.Undefined())
		}
	}()

	return js.vm.ToValue(1) // Return timer ID
}

func (js *JavaScriptRuntime) clearTimeout(call goja.FunctionCall) goja.Value {
	// Timer ID from argument (not implemented for simplicity)
	return goja.Undefined()
}

func (js *JavaScriptRuntime) setInterval(call goja.FunctionCall) goja.Value {
	callback := call.Argument(0)
	interval := call.Argument(1).ToInteger()

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			if callable, ok := goja.AssertFunction(callback); ok {
				callable(goja.Undefined())
			}
		}
	}()

	return js.vm.ToValue(1) // Return timer ID
}

func (js *JavaScriptRuntime) clearInterval(call goja.FunctionCall) goja.Value {
	// Timer ID from argument (not implemented for simplicity)
	return goja.Undefined()
}

// Built-in module creators
func (js *JavaScriptRuntime) createUtilModule() interface{} {
	return map[string]interface{}{
		"inspect": func(obj interface{}) string {
			data, _ := json.MarshalIndent(obj, "", "  ")
			return string(data)
		},
		"format": func(f string, args ...interface{}) string {
			return fmt.Sprintf(f, args...)
		},
		"isArray": func(obj interface{}) bool {
			_, ok := obj.([]interface{})
			return ok
		},
		"isObject": func(obj interface{}) bool {
			_, ok := obj.(map[string]interface{})
			return ok
		},
	}
}

func (js *JavaScriptRuntime) createCryptoModule() interface{} {
	return map[string]interface{}{
		"createHash": func(algorithm string) map[string]interface{} {
			return map[string]interface{}{
				"update": func(data string) map[string]interface{} {
					return map[string]interface{}{
						"digest": func(encoding string) string {
							hash := md5.Sum([]byte(data))
							return fmt.Sprintf("%x", hash)
						},
					}
				},
			}
		},
		"randomBytes": func(size int) []byte {
			bytes := make([]byte, size)
			// Simple mock - in real implementation use crypto/rand
			for i := range bytes {
				bytes[i] = byte(i % 256)
			}
			return bytes
		},
	}
}

func (js *JavaScriptRuntime) createPathModule() interface{} {
	return map[string]interface{}{
		"join": func(parts ...string) string {
			return filepath.Join(parts...)
		},
		"basename": func(path string) string {
			return filepath.Base(path)
		},
		"dirname": func(path string) string {
			return filepath.Dir(path)
		},
		"extname": func(path string) string {
			return filepath.Ext(path)
		},
		"resolve": func(path string) string {
			abs, _ := filepath.Abs(path)
			return abs
		},
	}
}

func (js *JavaScriptRuntime) createFsModule() interface{} {
	return map[string]interface{}{
		"readFileSync": func(filename string, encoding string) string {
			data, err := ioutil.ReadFile(filename)
			if err != nil {
				panic(js.vm.NewGoError(err))
			}
			return string(data)
		},
		"writeFileSync": func(filename string, data string) {
			err := ioutil.WriteFile(filename, []byte(data), 0644)
			if err != nil {
				panic(js.vm.NewGoError(err))
			}
		},
		"existsSync": func(filename string) bool {
			_, err := os.Stat(filename)
			return err == nil
		},
	}
}

func (js *JavaScriptRuntime) createHttpModule() interface{} {
	return map[string]interface{}{
		"request": func(options interface{}, callback goja.Value) {
			// Mock HTTP request
			go func() {
				time.Sleep(100 * time.Millisecond)
				if callable, ok := goja.AssertFunction(callback); ok {
					response := map[string]interface{}{
						"statusCode": 200,
						"headers":    map[string]string{"content-type": "application/json"},
					}
					callable(goja.Undefined(), js.vm.ToValue(response), js.vm.ToValue(`{"success": true}`))
				}
			}()
		},
	}
}

func (js *JavaScriptRuntime) createHttpsModule() interface{} {
	return js.createHttpModule() // Same as HTTP for now
}

func (js *JavaScriptRuntime) createUrlModule() interface{} {
	return map[string]interface{}{
		"parse": func(urlStr string) map[string]interface{} {
			// Simple URL parsing mock
			return map[string]interface{}{
				"protocol": "https:",
				"host":     "example.com",
				"pathname": "/path",
				"search":   "?query=value",
			}
		},
		"format": func(urlObj map[string]interface{}) string {
			return "https://example.com/path?query=value"
		},
	}
}

func (js *JavaScriptRuntime) createQueryStringModule() interface{} {
	return map[string]interface{}{
		"parse": func(str string) map[string]interface{} {
			result := make(map[string]interface{})
			pairs := strings.Split(str, "&")
			for _, pair := range pairs {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) == 2 {
					result[parts[0]] = parts[1]
				}
			}
			return result
		},
		"stringify": func(obj map[string]interface{}) string {
			var pairs []string
			for key, value := range obj {
				pairs = append(pairs, fmt.Sprintf("%s=%v", key, value))
			}
			return strings.Join(pairs, "&")
		},
	}
}

func (js *JavaScriptRuntime) createOsModule() interface{} {
	return map[string]interface{}{
		"platform": func() string { return "linux" },
		"arch":     func() string { return "x64" },
		"tmpdir":   func() string { return "/tmp" },
		"homedir":  func() string { return "/home/user" },
	}
}

func (js *JavaScriptRuntime) createEventsModule() interface{} {
	return map[string]interface{}{
		"EventEmitter": func() map[string]interface{} {
			return map[string]interface{}{
				"on":   func(event string, listener goja.Value) {},
				"emit": func(event string, args ...interface{}) {},
			}
		},
	}
}

// Mock implementations for common npm packages
func (js *JavaScriptRuntime) createAxiosMock() string {
	return `
		const axios = {
			get: async function(url, config = {}) {
				return {
					status: 200,
					statusText: 'OK',
					data: { success: true, url: url },
					headers: { 'content-type': 'application/json' }
				};
			},
			post: async function(url, data, config = {}) {
				return {
					status: 200,
					statusText: 'OK',
					data: { success: true, url: url, posted: data },
					headers: { 'content-type': 'application/json' }
				};
			},
			put: async function(url, data, config = {}) {
				return {
					status: 200,
					statusText: 'OK',
					data: { success: true, url: url, put: data },
					headers: { 'content-type': 'application/json' }
				};
			},
			delete: async function(url, config = {}) {
				return {
					status: 200,
					statusText: 'OK',
					data: { success: true, url: url, deleted: true },
					headers: { 'content-type': 'application/json' }
				};
			}
		};

		axios.create = function(config) {
			return axios;
		};

		module.exports = axios;
	`
}

func (js *JavaScriptRuntime) createLodashMock() string {
	return `
		const _ = {
			get: function(object, path, defaultValue) {
				const keys = path.split('.');
				let result = object;
				for (const key of keys) {
					if (result && typeof result === 'object' && key in result) {
						result = result[key];
					} else {
						return defaultValue;
					}
				}
				return result;
			},
			set: function(object, path, value) {
				const keys = path.split('.');
				let current = object;
				for (let i = 0; i < keys.length - 1; i++) {
					const key = keys[i];
					if (!(key in current) || typeof current[key] !== 'object') {
						current[key] = {};
					}
					current = current[key];
				}
				current[keys[keys.length - 1]] = value;
				return object;
			},
			clone: function(obj) {
				return JSON.parse(JSON.stringify(obj));
			},
			merge: function(target, ...sources) {
				return Object.assign(target, ...sources);
			},
			isEmpty: function(value) {
				return value == null || (typeof value === 'object' && Object.keys(value).length === 0);
			},
			isArray: Array.isArray,
			isObject: function(value) {
				return value != null && typeof value === 'object' && !Array.isArray(value);
			},
			map: function(collection, iteratee) {
				if (Array.isArray(collection)) {
					return collection.map(iteratee);
				}
				return Object.keys(collection).map(key => iteratee(collection[key], key));
			},
			filter: function(collection, predicate) {
				if (Array.isArray(collection)) {
					return collection.filter(predicate);
				}
				const result = {};
				Object.keys(collection).forEach(key => {
					if (predicate(collection[key], key)) {
						result[key] = collection[key];
					}
				});
				return result;
			}
		};

		module.exports = _;
	`
}

func (js *JavaScriptRuntime) createMomentMock() string {
	return `
		function moment(input, format) {
			const date = input ? new Date(input) : new Date();

			return {
				format: function(fmt = 'YYYY-MM-DD HH:mm:ss') {
					return date.toISOString().substring(0, 19).replace('T', ' ');
				},
				add: function(amount, unit) {
					const newDate = new Date(date);
					switch(unit) {
						case 'days': newDate.setDate(newDate.getDate() + amount); break;
						case 'hours': newDate.setHours(newDate.getHours() + amount); break;
						case 'minutes': newDate.setMinutes(newDate.getMinutes() + amount); break;
						case 'seconds': newDate.setSeconds(newDate.getSeconds() + amount); break;
					}
					return moment(newDate);
				},
				subtract: function(amount, unit) {
					return this.add(-amount, unit);
				},
				unix: function() {
					return Math.floor(date.getTime() / 1000);
				},
				valueOf: function() {
					return date.getTime();
				},
				toDate: function() {
					return new Date(date);
				},
				isValid: function() {
					return !isNaN(date.getTime());
				}
			};
		}

		moment.now = function() {
			return Date.now();
		};

		moment.utc = function(input) {
			return moment(input);
		};

		module.exports = moment;
	`
}

func (js *JavaScriptRuntime) createUuidMock() string {
	return `
		const uuid = {
			v4: function() {
				return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
					const r = Math.random() * 16 | 0;
					const v = c === 'x' ? r : (r & 0x3 | 0x8);
					return v.toString(16);
				});
			}
		};

		module.exports = uuid;
	`
}

func (js *JavaScriptRuntime) createCryptoJsMock() string {
	return `
		const CryptoJS = {
			MD5: function(message) {
				return { toString: function() { return 'mock-md5-hash'; } };
			},
			SHA1: function(message) {
				return { toString: function() { return 'mock-sha1-hash'; } };
			},
			SHA256: function(message) {
				return { toString: function() { return 'mock-sha256-hash'; } };
			},
			enc: {
				Base64: {
					stringify: function(wordArray) { return btoa('mock-data'); },
					parse: function(base64) { return { toString: function() { return 'mock-data'; } }; }
				},
				Utf8: {
					stringify: function(wordArray) { return 'mock-utf8-string'; },
					parse: function(utf8) { return { toString: function() { return 'mock-data'; } }; }
				}
			}
		};

		module.exports = CryptoJS;
	`
}

func (js *JavaScriptRuntime) Dispose() {
	js.mu.Lock()
	defer js.mu.Unlock()

	// Clear caches
	js.npmCache = make(map[string]*NpmPackage)
	js.globalModules = make(map[string]interface{})

	// Reset VM
	js.vm = goja.New()
}

func (js *JavaScriptRuntime) GetPackageInfo(name string) (*NpmPackage, bool) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	for key, pkg := range js.npmCache {
		if strings.HasPrefix(key, name+"@") {
			return pkg, true
		}
	}
	return nil, false
}

func (js *JavaScriptRuntime) SetExecutionContext(context *ExecutionContext) {
	js.mu.Lock()
	defer js.mu.Unlock()
	js.executionContext = context
}
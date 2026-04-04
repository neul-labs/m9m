package runtime

import (
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

type JavaScriptRuntime struct {
	mu               sync.RWMutex
	vm               *goja.Runtime
	registry         *require.Registry
	npmCache         map[string]*NpmPackage
	nodeModulesPath  string
	globalModules    map[string]interface{}
	n8nHelpers       *N8nHelpers
	executionContext *ExecutionContext
}

type NpmPackage struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Main         string            `json:"main"`
	Dependencies map[string]string `json:"dependencies"`
	CachePath    string            `json:"cache_path"`
	LoadedAt     time.Time         `json:"loaded_at"`
	Module       interface{}       `json:"-"`
}

type ExecutionContext struct {
	WorkflowID  string                 `json:"workflow_id"`
	ExecutionID string                 `json:"execution_id"`
	NodeID      string                 `json:"node_id"`
	ItemIndex   int                    `json:"item_index"`
	RunIndex    int                    `json:"run_index"`
	Mode        string                 `json:"mode"`
	Timezone    string                 `json:"timezone"`
	Variables   map[string]interface{} `json:"variables"`
	Credentials map[string]interface{} `json:"credentials"`
}

type N8nHelpers struct {
	runtime *JavaScriptRuntime
}

type NodeJSGlobals struct {
	Process       *ProcessObject                     `json:"process"`
	Buffer        *BufferObject                      `json:"Buffer"`
	Global        interface{}                        `json:"global"`
	Console       interface{}                        `json:"console"`
	SetTimeout    func(goja.FunctionCall) goja.Value `json:"setTimeout"`
	ClearTimeout  func(goja.FunctionCall) goja.Value `json:"clearTimeout"`
	SetInterval   func(goja.FunctionCall) goja.Value `json:"setInterval"`
	ClearInterval func(goja.FunctionCall) goja.Value `json:"clearInterval"`
}

type ProcessObject struct {
	Env      map[string]string `json:"env"`
	Version  string            `json:"version"`
	Platform string            `json:"platform"`
	Arch     string            `json:"arch"`
	Argv     []string          `json:"argv"`
	Pid      int               `json:"pid"`
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

	js.initializeNodeJSGlobals()
	js.initializeBuiltinModules()
	js.initializeN8nGlobals()

	return js
}

func (js *JavaScriptRuntime) Dispose() {
	js.mu.Lock()
	defer js.mu.Unlock()

	js.npmCache = make(map[string]*NpmPackage)
	js.globalModules = make(map[string]interface{})
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

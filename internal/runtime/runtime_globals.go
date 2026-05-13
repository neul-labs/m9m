package runtime

import (
	"fmt"
	"os"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
)

func (js *JavaScriptRuntime) initializeNodeJSGlobals() {
	registry := new(require.Registry)
	registry.Enable(js.vm)
	console.Enable(js.vm)

	process := &ProcessObject{
		Env:      make(map[string]string),
		Version:  "v16.0.0",
		Platform: "linux",
		Arch:     "x64",
		Argv:     []string{"node"},
		Pid:      1234,
	}

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 && isEnvVarSafe(parts[0]) {
			process.Env[parts[0]] = parts[1]
		}
	}

	_ = js.vm.Set("process", process)
	_ = js.vm.Set("Buffer", &BufferObject{runtime: js})
	_ = js.vm.Set("global", js.vm.GlobalObject())
	_ = js.vm.Set("setTimeout", js.setTimeout)
	_ = js.vm.Set("clearTimeout", js.clearTimeout)
	_ = js.vm.Set("setInterval", js.setInterval)
	_ = js.vm.Set("clearInterval", js.clearInterval)
	_ = js.vm.Set("require", js.requireFunction)
}

func (js *JavaScriptRuntime) initializeN8nGlobals() {
	_ = js.vm.Set("$json", js.createJsonHelper())
	_ = js.vm.Set("$node", js.createNodeHelper())
	_ = js.vm.Set("$parameter", js.createParameterHelper())
	_ = js.vm.Set("$workflow", js.createWorkflowHelper())
	_ = js.vm.Set("$execution", js.createExecutionHelper())
	_ = js.vm.Set("$items", js.createItemsHelper())
	_ = js.vm.Set("$input", js.createInputHelper())
	_ = js.vm.Set("$", js.createDollarHelper())
	_ = js.vm.Set("helpers", js.n8nHelpers)
	_ = js.vm.Set("moment", js.createMomentHelper())
	_ = js.vm.Set("_", js.createLodashHelper())
}

func (js *JavaScriptRuntime) requireFunction(call goja.FunctionCall) goja.Value {
	moduleName := call.Argument(0).String()

	if module, exists := js.globalModules[moduleName]; exists {
		return js.vm.ToValue(module)
	}

	pkg, err := js.LoadNpmPackage(moduleName, "latest")
	if err != nil {
		panic(js.vm.NewGoError(fmt.Errorf("cannot find module '%s': %w", moduleName, err)))
	}

	return js.vm.ToValue(pkg.Module)
}

package runtime

import (
	"time"

	"github.com/dop251/goja"
)

func (js *JavaScriptRuntime) setTimeout(call goja.FunctionCall) goja.Value {
	callback := call.Argument(0)
	delay := call.Argument(1).ToInteger()

	go func() {
		time.Sleep(time.Duration(delay) * time.Millisecond)
		if callable, ok := goja.AssertFunction(callback); ok {
			callable(goja.Undefined())
		}
	}()

	return js.vm.ToValue(1)
}

func (js *JavaScriptRuntime) clearTimeout(call goja.FunctionCall) goja.Value {
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

	return js.vm.ToValue(1)
}

func (js *JavaScriptRuntime) clearInterval(call goja.FunctionCall) goja.Value {
	return goja.Undefined()
}

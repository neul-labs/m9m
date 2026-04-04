package runtime

import (
	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
)

func (js *JavaScriptRuntime) Execute(code string, context *ExecutionContext, items []model.DataItem) (interface{}, error) {
	js.mu.Lock()
	defer js.mu.Unlock()

	js.executionContext = context
	js.vm.Set("$items", js.createItemsHelperWithData(items))
	js.vm.Set("items", items)

	result, err := js.vm.RunString(code)
	if err != nil {
		return nil, err
	}

	return result.Export(), nil
}

func (js *JavaScriptRuntime) ExecuteExpression(expression string, context *expressions.ExpressionContext) (interface{}, error) {
	js.mu.Lock()
	defer js.mu.Unlock()

	for key, value := range context.GetVariables() {
		js.vm.Set(key, value)
	}

	result, err := js.vm.RunString(expression)
	if err != nil {
		return nil, err
	}

	return result.Export(), nil
}

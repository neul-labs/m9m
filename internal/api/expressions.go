package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/neul-labs/m9m/internal/expressions"
	"github.com/neul-labs/m9m/internal/model"
)

const maxExpressionLength = 10000

var dangerousPatterns = []string{
	"require(", "import(", "eval(", "Function(",
	"process.", "global.", "__proto__", "constructor.",
	"Reflect.", "Proxy(", "Object.defineProperty",
}

func (s *APIServer) EvaluateExpression(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Expression string                 `json:"expression"`
		Context    map[string]interface{} `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.Expression == "" {
		s.sendError(w, http.StatusBadRequest, "Expression is required", nil)
		return
	}
	if len(req.Expression) > maxExpressionLength {
		s.sendError(w, http.StatusBadRequest, "Expression too long", nil)
		return
	}
	if containsDangerousPattern(req.Expression) {
		s.sendError(w, http.StatusBadRequest, "Expression contains disallowed constructs", nil)
		return
	}

	evaluator := expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig())
	ctx := expressions.NewExpressionContext()
	if req.Context != nil {
		ctx.ConnectionInputData = []model.DataItem{{JSON: req.Context}}
	}

	result, err := evaluator.EvaluateExpression(req.Expression, ctx)
	if err != nil {
		s.sendJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"result":  result,
	})
}

func containsDangerousPattern(expr string) bool {
	lowerExpr := strings.ToLower(expr)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerExpr, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

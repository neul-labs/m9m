package templates

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/neul-labs/m9m/internal/expressions"
)

func NewSecurityScanner() *SecurityScanner {
	scanner := &SecurityScanner{
		rules:     make(map[string]*SecurityRule),
		whitelist: make(map[string]bool),
		blacklist: make(map[string]bool),
		evaluator: expressions.NewGojaExpressionEvaluator(expressions.DefaultEvaluatorConfig()),
	}

	scanner.initializeSecurityRules()
	return scanner
}

func (scanner *SecurityScanner) ScanTemplate(template *MarketplaceTemplate) *SecurityReport {
	report := &SecurityReport{
		ScanDate:        time.Now(),
		SecurityLevel:   "safe",
		Vulnerabilities: []*Vulnerability{},
		CodeQuality:     &CodeQuality{Score: 100.0},
		TrustScore:      1.0,
	}

	templateJSON, _ := json.Marshal(template.WorkflowTemplate)
	content := string(templateJSON)

	for _, rule := range scanner.rules {
		if scanner.matchesPattern(content, rule.Pattern) {
			vulnerability := &Vulnerability{
				ID:          rule.ID,
				Type:        rule.Name,
				Severity:    rule.Severity,
				Description: rule.Description,
				Mitigation:  rule.Remediation,
			}
			report.Vulnerabilities = append(report.Vulnerabilities, vulnerability)

			if rule.Severity == "high" || rule.Severity == "critical" {
				report.SecurityLevel = "danger"
				report.TrustScore *= 0.5
			} else if rule.Severity == "medium" {
				if report.SecurityLevel == "safe" {
					report.SecurityLevel = "warning"
				}
				report.TrustScore *= 0.8
			}
		}
	}

	return report
}

func (scanner *SecurityScanner) matchesPattern(content, pattern string) bool {
	return strings.Contains(strings.ToLower(content), strings.ToLower(pattern))
}

func (scanner *SecurityScanner) initializeSecurityRules() {
	scanner.rules["hardcoded_credentials"] = &SecurityRule{
		ID:          "hardcoded_credentials",
		Name:        "Hardcoded Credentials",
		Pattern:     "password|secret|key|token",
		Severity:    "high",
		Description: "Template may contain hardcoded credentials",
		Remediation: "Use parameter substitution for sensitive values",
	}

	scanner.rules["external_scripts"] = &SecurityRule{
		ID:          "external_scripts",
		Name:        "External Scripts",
		Pattern:     "eval|function|script",
		Severity:    "medium",
		Description: "Template contains executable code",
		Remediation: "Review code execution for security risks",
	}
}

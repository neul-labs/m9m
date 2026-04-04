package runtime

import (
	"fmt"
	"strings"
)

// allowedEnvVars is a whitelist of environment variables safe to expose to JavaScript code
// SECURITY: Only expose non-sensitive environment variables
var allowedEnvVars = map[string]bool{
	"NODE_ENV": true,
	"TZ":       true,
	"LANG":     true,
	"LC_ALL":   true,
	"LC_CTYPE": true,
	"HOME":     true,
	"USER":     true,
	"SHELL":    true,
	"PATH":     false,
	"PWD":      true,
	"TERM":     true,
	"HOSTNAME": true,
	"LOGNAME":  true,
	"TMPDIR":   true,
	"TEMP":     true,
	"TMP":      true,
}

// sensitiveEnvPatterns contains patterns of environment variable names that should never be exposed
var sensitiveEnvPatterns = []string{
	"KEY",
	"SECRET",
	"TOKEN",
	"PASSWORD",
	"PASSWD",
	"CREDENTIAL",
	"AUTH",
	"API_KEY",
	"APIKEY",
	"PRIVATE",
	"AWS_",
	"AZURE_",
	"GCP_",
	"GOOGLE_",
	"DATABASE",
	"DB_",
	"REDIS_",
	"MONGO",
	"POSTGRES",
	"MYSQL",
	"N8N_",
	"M9M_",
	"ENCRYPTION",
	"CERT",
	"SSL_",
	"TLS_",
	"JWT",
	"OAUTH",
	"SMTP",
	"MAIL",
}

func isEnvVarSafe(name string) bool {
	if allowed, exists := allowedEnvVars[name]; exists {
		return allowed
	}

	upperName := strings.ToUpper(name)
	for _, pattern := range sensitiveEnvPatterns {
		if strings.Contains(upperName, pattern) {
			return false
		}
	}

	return false
}

// escapeJSString escapes a string for safe embedding in JavaScript code
// SECURITY: Prevents code injection through string interpolation
func escapeJSString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\u2028", `\u2028`)
	s = strings.ReplaceAll(s, "\u2029", `\u2029`)
	return s
}

// validatePackageName validates that a package name is safe
// SECURITY: Prevents malicious package names from causing issues
func validatePackageName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("package name cannot be empty")
	}
	if len(name) > 214 {
		return fmt.Errorf("package name too long")
	}
	if name[0] == '.' || name[0] == '_' {
		return fmt.Errorf("package name cannot start with . or _")
	}

	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '@' || c == '/') {
			return fmt.Errorf("package name contains invalid character: %c", c)
		}
	}

	return nil
}

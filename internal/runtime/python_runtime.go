package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PythonRuntime provides Python code execution capabilities compatible with n8n
type PythonRuntime struct {
	mu              sync.RWMutex
	pythonPath      string
	venvPath        string
	tempDir         string
	maxExecutionTime time.Duration
	enabledPackages map[string]bool
	sandboxMode     bool
	initialized     bool
}

// PythonExecutionResult contains the result of Python code execution
type PythonExecutionResult struct {
	Output interface{} `json:"output"`
	Error  string      `json:"error,omitempty"`
	Logs   []string    `json:"logs,omitempty"`
	Type   string      `json:"type"` // "success" or "error"
}

// NewPythonRuntime creates a new Python runtime instance
func NewPythonRuntime() (*PythonRuntime, error) {
	tempDir, err := os.MkdirTemp("", "n8n-python-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	pr := &PythonRuntime{
		pythonPath:      "python3",
		tempDir:         tempDir,
		maxExecutionTime: 30 * time.Second,
		enabledPackages: make(map[string]bool),
		sandboxMode:     true,
	}

	// Initialize default allowed packages
	pr.enableDefaultPackages()

	return pr, nil
}

// Initialize sets up the Python environment
func (pr *PythonRuntime) Initialize(ctx context.Context) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.initialized {
		return nil
	}

	// Check Python availability
	if err := pr.checkPythonAvailable(); err != nil {
		return err
	}

	// Create virtual environment
	if err := pr.createVirtualEnvironment(); err != nil {
		return err
	}

	// Install base packages
	if err := pr.installBasePackages(); err != nil {
		return err
	}

	pr.initialized = true
	return nil
}

// Execute runs Python code with n8n-compatible context
func (pr *PythonRuntime) Execute(code string, inputData interface{}, options map[string]interface{}) (*PythonExecutionResult, error) {
	pr.mu.RLock()
	if !pr.initialized {
		pr.mu.RUnlock()
		if err := pr.Initialize(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to initialize Python runtime: %w", err)
		}
		pr.mu.RLock()
	}
	pr.mu.RUnlock()

	// Prepare execution context
	scriptPath, err := pr.prepareScript(code, inputData, options)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare script: %w", err)
	}
	defer os.Remove(scriptPath)

	// Execute Python script
	return pr.executeScript(scriptPath)
}

// ExecuteWithPackages runs Python code with specific package requirements
func (pr *PythonRuntime) ExecuteWithPackages(code string, packages []string, inputData interface{}) (*PythonExecutionResult, error) {
	// Install required packages
	for _, pkg := range packages {
		if err := pr.InstallPackage(pkg); err != nil {
			return nil, fmt.Errorf("failed to install package %s: %w", pkg, err)
		}
	}

	return pr.Execute(code, inputData, nil)
}

// InstallPackage installs a Python package
func (pr *PythonRuntime) InstallPackage(packageName string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if !pr.isPackageAllowed(packageName) {
		return fmt.Errorf("package %s is not allowed in sandbox mode", packageName)
	}

	cmd := exec.Command(pr.getPipPath(), "install", packageName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	return nil
}

// prepareScript creates a Python script file with context
func (pr *PythonRuntime) prepareScript(code string, inputData interface{}, options map[string]interface{}) (string, error) {
	// Create wrapper script with n8n context
	wrapper := pr.generateWrapper(code, inputData, options)

	// Write to temp file
	scriptPath := filepath.Join(pr.tempDir, fmt.Sprintf("script_%d.py", time.Now().UnixNano()))
	if err := os.WriteFile(scriptPath, []byte(wrapper), 0600); err != nil {
		return "", err
	}

	return scriptPath, nil
}

// generateWrapper creates the Python wrapper with n8n context
func (pr *PythonRuntime) generateWrapper(code string, inputData interface{}, options map[string]interface{}) string {
	inputJSON, _ := json.Marshal(inputData)
	optionsJSON, _ := json.Marshal(options)

	wrapper := fmt.Sprintf(`
import json
import sys
import traceback
from typing import Any, Dict, List, Optional

# n8n compatibility helpers
class N8nHelpers:
    def __init__(self, input_data, options):
        self.input_data = input_data
        self.options = options or {}
        self.logs = []
    
    def log(self, *args):
        """Log messages for debugging"""
        message = " ".join(str(arg) for arg in args)
        self.logs.append(message)
        print(f"[LOG] {message}", file=sys.stderr)
    
    def get_input(self, path: str = None):
        """Get input data, optionally at a specific path"""
        if not path:
            return self.input_data
        
        parts = path.split('.')
        data = self.input_data
        for part in parts:
            if isinstance(data, dict):
                data = data.get(part)
            elif isinstance(data, list) and part.isdigit():
                idx = int(part)
                data = data[idx] if idx < len(data) else None
            else:
                return None
        return data
    
    def set_output(self, data: Any) -> Dict:
        """Format output for n8n"""
        return {
            "output": data,
            "logs": self.logs,
            "type": "success"
        }

# Initialize helpers
$input = json.loads('%s')
$options = json.loads('%s')
$helpers = N8nHelpers($input, $options)

# Make helpers available globally
$json = $input
$node = {"name": "PythonCode", "type": "python"}
$workflow = {"name": "CurrentWorkflow", "active": True}

# User code execution
try:
    # Execute user code
    exec_globals = {
        '__builtins__': __builtins__,
        '$input': $input,
        '$json': $json,
        '$node': $node,
        '$workflow': $workflow,
        '$helpers': $helpers,
        '$options': $options,
        'json': json,
    }
    
    # Add safe imports
    import datetime
    import re
    import math
    import random
    import hashlib
    import base64
    import urllib.parse
    
    exec_globals.update({
        'datetime': datetime,
        're': re,
        'math': math,
        'random': random,
        'hashlib': hashlib,
        'base64': base64,
        'urllib': urllib,
    })
    
    # Execute user code
    exec('''%s''', exec_globals)
    
    # Get return value if specified
    if 'return_value' in exec_globals:
        result = $helpers.set_output(exec_globals['return_value'])
    elif 'output' in exec_globals:
        result = $helpers.set_output(exec_globals['output'])
    else:
        # Try to find the last expression value
        result = $helpers.set_output($input)
    
    print(json.dumps(result))
    
except Exception as e:
    error_result = {
        "error": str(e),
        "logs": $helpers.logs,
        "type": "error",
        "traceback": traceback.format_exc()
    }
    print(json.dumps(error_result))
    sys.exit(1)
`, string(inputJSON), string(optionsJSON), code)

	return wrapper
}

// executeScript runs the Python script and captures output
func (pr *PythonRuntime) executeScript(scriptPath string) (*PythonExecutionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pr.maxExecutionTime)
	defer cancel()

	cmd := exec.CommandContext(ctx, pr.getPythonPath(), scriptPath)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	
	// Parse output
	var result PythonExecutionResult
	if jsonErr := json.Unmarshal(stdout.Bytes(), &result); jsonErr != nil {
		// If JSON parsing fails, treat as error
		if err != nil {
			return &PythonExecutionResult{
				Error: fmt.Sprintf("execution error: %v\nstderr: %s", err, stderr.String()),
				Type:  "error",
			}, nil
		}
		// Return raw output if execution succeeded but JSON parsing failed
		return &PythonExecutionResult{
			Output: stdout.String(),
			Type:   "success",
		}, nil
	}

	return &result, nil
}

// checkPythonAvailable verifies Python is installed
func (pr *PythonRuntime) checkPythonAvailable() error {
	cmd := exec.Command(pr.pythonPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Python not found: %w", err)
	}
	return nil
}

// createVirtualEnvironment creates an isolated Python environment
func (pr *PythonRuntime) createVirtualEnvironment() error {
	pr.venvPath = filepath.Join(pr.tempDir, "venv")
	cmd := exec.Command(pr.pythonPath, "-m", "venv", pr.venvPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create virtual environment: %w", err)
	}
	return nil
}

// installBasePackages installs essential Python packages
func (pr *PythonRuntime) installBasePackages() error {
	basePackages := []string{
		"numpy",
		"pandas",
		"requests",
	}

	for _, pkg := range basePackages {
		if pr.isPackageAllowed(pkg) {
			cmd := exec.Command(pr.getPipPath(), "install", "--quiet", pkg)
			cmd.Run() // Ignore errors for optional packages
		}
	}

	return nil
}

// enableDefaultPackages sets up the default allowed packages
func (pr *PythonRuntime) enableDefaultPackages() {
	allowedPackages := []string{
		"numpy", "pandas", "requests", "urllib3", "beautifulsoup4",
		"matplotlib", "scipy", "scikit-learn", "pillow", "openpyxl",
		"xlrd", "xlwt", "pyyaml", "jsonschema", "python-dateutil",
		"pytz", "regex", "tqdm", "colorama", "tabulate",
	}

	for _, pkg := range allowedPackages {
		pr.enabledPackages[pkg] = true
	}
}

// isPackageAllowed checks if a package is allowed in sandbox mode
func (pr *PythonRuntime) isPackageAllowed(packageName string) bool {
	if !pr.sandboxMode {
		return true
	}
	
	// Check against allowed list
	baseName := strings.Split(packageName, "==")[0] // Remove version specifier
	return pr.enabledPackages[baseName]
}

// getPythonPath returns the path to Python executable
func (pr *PythonRuntime) getPythonPath() string {
	if pr.venvPath != "" {
		return filepath.Join(pr.venvPath, "bin", "python")
	}
	return pr.pythonPath
}

// getPipPath returns the path to pip executable
func (pr *PythonRuntime) getPipPath() string {
	if pr.venvPath != "" {
		return filepath.Join(pr.venvPath, "bin", "pip")
	}
	return "pip3"
}

// Cleanup cleans up temporary files and virtual environment
func (pr *PythonRuntime) Cleanup() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.tempDir != "" {
		return os.RemoveAll(pr.tempDir)
	}
	return nil
}

// SetMaxExecutionTime sets the maximum execution time for Python scripts
func (pr *PythonRuntime) SetMaxExecutionTime(duration time.Duration) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.maxExecutionTime = duration
}

// SetSandboxMode enables or disables sandbox mode
func (pr *PythonRuntime) SetSandboxMode(enabled bool) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.sandboxMode = enabled
}

// EnablePackage adds a package to the allowed list
func (pr *PythonRuntime) EnablePackage(packageName string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.enabledPackages[packageName] = true
}
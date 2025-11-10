package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
)

// EmbeddedPythonRuntime provides Python execution using a JavaScript-based Python interpreter
// This allows us to run Python code without requiring Python to be installed
type EmbeddedPythonRuntime struct {
	mu           sync.RWMutex
	vm           *goja.Runtime
	pyEngine     *goja.Object
	initialized  bool
	maxExecTime  time.Duration
}

// NewEmbeddedPythonRuntime creates a new embedded Python runtime
func NewEmbeddedPythonRuntime() (*EmbeddedPythonRuntime, error) {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	// Enable console for debugging
	new(require.Registry).Enable(vm)
	console.Enable(vm)

	epr := &EmbeddedPythonRuntime{
		vm:          vm,
		maxExecTime: 30 * time.Second,
	}

	if err := epr.initialize(); err != nil {
		return nil, err
	}

	return epr, nil
}

// initialize sets up the Python interpreter in JavaScript
func (epr *EmbeddedPythonRuntime) initialize() error {
	epr.mu.Lock()
	defer epr.mu.Unlock()

	if epr.initialized {
		return nil
	}

	// Load Skulpt (JavaScript Python interpreter) or create a minimal Python-like interpreter
	// For production, we'd load Skulpt or Pyodide here
	// For now, we'll create a Python-compatible execution environment
	
	_, err := epr.vm.RunString(`
		// Python-like execution environment
		const PythonRuntime = {
			// Core Python built-ins
			builtins: {
				print: function(...args) {
					const output = args.map(arg => {
						if (typeof arg === 'object') {
							return JSON.stringify(arg);
						}
						return String(arg);
					}).join(' ');
					console.log(output);
					return output;
				},
				len: function(obj) {
					if (Array.isArray(obj) || typeof obj === 'string') {
						return obj.length;
					}
					if (typeof obj === 'object') {
						return Object.keys(obj).length;
					}
					return 0;
				},
				range: function(start, stop, step) {
					if (stop === undefined) {
						stop = start;
						start = 0;
					}
					if (step === undefined) {
						step = 1;
					}
					const result = [];
					for (let i = start; i < stop; i += step) {
						result.push(i);
					}
					return result;
				},
				str: function(obj) {
					return String(obj);
				},
				int: function(obj) {
					return parseInt(obj, 10);
				},
				float: function(obj) {
					return parseFloat(obj);
				},
				bool: function(obj) {
					return !!obj;
				},
				list: function(obj) {
					if (Array.isArray(obj)) return obj;
					if (typeof obj === 'string') return obj.split('');
					return Array.from(obj || []);
				},
				dict: function(obj) {
					return Object(obj || {});
				},
				sum: function(arr) {
					return arr.reduce((a, b) => a + b, 0);
				},
				map: function(func, iterable) {
					return iterable.map(func);
				},
				filter: function(func, iterable) {
					return iterable.filter(func);
				},
				zip: function(...arrays) {
					const length = Math.min(...arrays.map(arr => arr.length));
					const result = [];
					for (let i = 0; i < length; i++) {
						result.push(arrays.map(arr => arr[i]));
					}
					return result;
				},
				enumerate: function(iterable) {
					return iterable.map((item, index) => [index, item]);
				},
				type: function(obj) {
					if (obj === null) return 'NoneType';
					if (Array.isArray(obj)) return 'list';
					if (typeof obj === 'object') return 'dict';
					if (typeof obj === 'string') return 'str';
					if (typeof obj === 'number') {
						return Number.isInteger(obj) ? 'int' : 'float';
					}
					if (typeof obj === 'boolean') return 'bool';
					if (typeof obj === 'function') return 'function';
					return typeof obj;
				}
			},

			// Python standard library modules (subset)
			modules: {
				json: {
					dumps: function(obj, indent) {
						return JSON.stringify(obj, null, indent);
					},
					loads: function(str) {
						return JSON.parse(str);
					}
				},
				math: {
					pi: Math.PI,
					e: Math.E,
					abs: Math.abs,
					ceil: Math.ceil,
					floor: Math.floor,
					round: Math.round,
					sqrt: Math.sqrt,
					pow: Math.pow,
					sin: Math.sin,
					cos: Math.cos,
					tan: Math.tan,
					log: Math.log,
					log10: Math.log10,
					exp: Math.exp,
					min: Math.min,
					max: Math.max
				},
				random: {
					random: Math.random,
					randint: function(a, b) {
						return Math.floor(Math.random() * (b - a + 1)) + a;
					},
					choice: function(arr) {
						return arr[Math.floor(Math.random() * arr.length)];
					},
					shuffle: function(arr) {
						const result = [...arr];
						for (let i = result.length - 1; i > 0; i--) {
							const j = Math.floor(Math.random() * (i + 1));
							[result[i], result[j]] = [result[j], result[i]];
						}
						return result;
					},
					uniform: function(a, b) {
						return Math.random() * (b - a) + a;
					}
				},
				datetime: {
					datetime: {
						now: function() {
							return new Date();
						},
						fromisoformat: function(str) {
							return new Date(str);
						}
					},
					date: {
						today: function() {
							const d = new Date();
							d.setHours(0, 0, 0, 0);
							return d;
						}
					},
					timedelta: function(days = 0, seconds = 0, microseconds = 0, milliseconds = 0, minutes = 0, hours = 0, weeks = 0) {
						return {
							totalMilliseconds: milliseconds + (seconds * 1000) + (minutes * 60000) + (hours * 3600000) + (days * 86400000) + (weeks * 604800000),
							add: function(date) {
								return new Date(date.getTime() + this.totalMilliseconds);
							}
						};
					}
				},
				re: {
					search: function(pattern, string) {
						const match = string.match(new RegExp(pattern));
						return match ? { group: () => match[0] } : null;
					},
					findall: function(pattern, string) {
						return string.match(new RegExp(pattern, 'g')) || [];
					},
					sub: function(pattern, replacement, string) {
						return string.replace(new RegExp(pattern, 'g'), replacement);
					},
					match: function(pattern, string) {
						const regex = new RegExp('^' + pattern);
						const match = string.match(regex);
						return match ? { group: () => match[0] } : null;
					}
				},
				string: {
					ascii_lowercase: 'abcdefghijklmnopqrstuvwxyz',
					ascii_uppercase: 'ABCDEFGHIJKLMNOPQRSTUVWXYZ',
					digits: '0123456789',
					punctuation: '!"#$%&\'()*+,-./:;<=>?@[\\]^_`{|}~'
				},
				collections: {
					Counter: function(iterable) {
						const counter = {};
						for (const item of iterable || []) {
							counter[item] = (counter[item] || 0) + 1;
						}
						return counter;
					},
					defaultdict: function(defaultFactory) {
						return new Proxy({}, {
							get(target, prop) {
								if (!(prop in target)) {
									target[prop] = defaultFactory();
								}
								return target[prop];
							}
						});
					}
				}
			},

			// Execute Python-like code
			execute: function(code, context) {
				// Create execution context with Python builtins and modules
				const execContext = {
					...this.builtins,
					...context,
					// Import simulation
					import: (moduleName) => {
						if (this.modules[moduleName]) {
							return this.modules[moduleName];
						}
						throw new Error(`Module '${moduleName}' not found`);
					}
				};

				// Transform Python-like syntax to JavaScript
				let jsCode = this.transformPythonToJS(code);
				
				try {
					// Create function with context
					const func = new Function(...Object.keys(execContext), jsCode);
					const result = func(...Object.values(execContext));
					return {
						success: true,
						output: result,
						error: null
					};
				} catch (error) {
					return {
						success: false,
						output: null,
						error: error.message
					};
				}
			},

			// Basic Python to JavaScript syntax transformation
			transformPythonToJS: function(code) {
				let jsCode = code;

				// Transform Python-specific syntax
				jsCode = jsCode.replace(/^import\s+(\w+)/gm, 'const $1 = import("$1")');
				jsCode = jsCode.replace(/^from\s+(\w+)\s+import\s+(\w+)/gm, 'const $2 = import("$1").$2');
				jsCode = jsCode.replace(/\bTrue\b/g, 'true');
				jsCode = jsCode.replace(/\bFalse\b/g, 'false');
				jsCode = jsCode.replace(/\bNone\b/g, 'null');
				jsCode = jsCode.replace(/\band\b/g, '&&');
				jsCode = jsCode.replace(/\bor\b/g, '||');
				jsCode = jsCode.replace(/\bnot\b/g, '!');
				jsCode = jsCode.replace(/\belif\b/g, 'else if');
				jsCode = jsCode.replace(/def\s+(\w+)\s*\(([^)]*)\)\s*:/g, 'function $1($2) {');
				jsCode = jsCode.replace(/if\s+(.+)\s*:/g, 'if ($1) {');
				jsCode = jsCode.replace(/else\s*:/g, 'else {');
				jsCode = jsCode.replace(/for\s+(\w+)\s+in\s+(.+)\s*:/g, 'for (const $1 of $2) {');
				jsCode = jsCode.replace(/while\s+(.+)\s*:/g, 'while ($1) {');
				
				// Handle indentation-based blocks (simplified)
				const lines = jsCode.split('\n');
				const processedLines = [];
				const indentStack = [0];
				
				for (let i = 0; i < lines.length; i++) {
					const line = lines[i];
					const indent = line.search(/\S/);
					
					if (indent === -1) {
						processedLines.push(line);
						continue;
					}
					
					while (indentStack.length > 1 && indent < indentStack[indentStack.length - 1]) {
						indentStack.pop();
						processedLines.push(' '.repeat(indentStack[indentStack.length - 1]) + '}');
					}
					
					processedLines.push(line);
					
					if (line.includes('{')) {
						indentStack.push(indent + 2);
					}
				}
				
				// Close remaining blocks
				while (indentStack.length > 1) {
					indentStack.pop();
					processedLines.push('}');
				}
				
				return processedLines.join('\n');
			}
		};

		// Export the runtime
		globalThis.PythonRuntime = PythonRuntime;
	`)

	if err != nil {
		return fmt.Errorf("failed to initialize Python runtime: %w", err)
	}

	// Get reference to Python runtime
	pyEngine := epr.vm.Get("PythonRuntime")
	if pyEngine == nil {
		return fmt.Errorf("failed to get Python runtime")
	}

	obj, ok := pyEngine.(*goja.Object)
	if !ok {
		return fmt.Errorf("Python runtime is not an object")
	}

	epr.pyEngine = obj
	epr.initialized = true
	return nil
}

// Execute runs Python-like code in the embedded interpreter
func (epr *EmbeddedPythonRuntime) Execute(code string, inputData interface{}, options map[string]interface{}) (*PythonExecutionResult, error) {
	epr.mu.RLock()
	if !epr.initialized {
		epr.mu.RUnlock()
		return nil, fmt.Errorf("Python runtime not initialized")
	}
	epr.mu.RUnlock()

	// Prepare context
	context := map[string]interface{}{
		"$input": inputData,
		"$json":  inputData,
		"$options": options,
	}

	// Set timeout
	ctx, cancel := context.WithTimeout(context.Background(), epr.maxExecTime)
	defer cancel()

	// Create a channel for result
	resultChan := make(chan *PythonExecutionResult, 1)
	errChan := make(chan error, 1)

	go func() {
		epr.mu.Lock()
		defer epr.mu.Unlock()

		// Call execute method
		executeFunc := epr.pyEngine.Get("execute")
		if executeFunc == nil {
			errChan <- fmt.Errorf("execute function not found")
			return
		}

		// Execute the code
		result, err := goja.AssertFunction(executeFunc)(goja.Undefined(), epr.vm.ToValue(code), epr.vm.ToValue(context))
		if err != nil {
			errChan <- fmt.Errorf("execution error: %w", err)
			return
		}

		// Parse result
		resultObj := result.ToObject(epr.vm)
		success := resultObj.Get("success")
		output := resultObj.Get("output")
		errorMsg := resultObj.Get("error")

		if success != nil && success.ToBoolean() {
			resultChan <- &PythonExecutionResult{
				Output: output.Export(),
				Type:   "success",
			}
		} else {
			errorStr := ""
			if errorMsg != nil {
				errorStr = errorMsg.String()
			}
			resultChan <- &PythonExecutionResult{
				Error: errorStr,
				Type:  "error",
			}
		}
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("execution timeout")
	}
}

// ExecuteWithDataFrame runs Python code with pandas-like DataFrame support
func (epr *EmbeddedPythonRuntime) ExecuteWithDataFrame(code string, data []map[string]interface{}) (*PythonExecutionResult, error) {
	// Add DataFrame simulation to context
	epr.mu.Lock()
	_, err := epr.vm.RunString(`
		// Simple DataFrame implementation
		class DataFrame {
			constructor(data) {
				this.data = data;
				this.columns = data.length > 0 ? Object.keys(data[0]) : [];
				this.shape = [data.length, this.columns.length];
			}

			head(n = 5) {
				return new DataFrame(this.data.slice(0, n));
			}

			tail(n = 5) {
				return new DataFrame(this.data.slice(-n));
			}

			filter(condition) {
				return new DataFrame(this.data.filter(condition));
			}

			select(...columns) {
				return new DataFrame(this.data.map(row => {
					const newRow = {};
					columns.forEach(col => {
						newRow[col] = row[col];
					});
					return newRow;
				}));
			}

			groupBy(column) {
				const groups = {};
				this.data.forEach(row => {
					const key = row[column];
					if (!groups[key]) groups[key] = [];
					groups[key].push(row);
				});
				return groups;
			}

			toJSON() {
				return this.data;
			}
		}

		// Add pandas-like module
		PythonRuntime.modules.pandas = {
			DataFrame: DataFrame,
			read_json: function(jsonStr) {
				return new DataFrame(JSON.parse(jsonStr));
			},
			concat: function(dfs) {
				const allData = [];
				dfs.forEach(df => allData.push(...df.data));
				return new DataFrame(allData);
			}
		};
	`)
	epr.mu.Unlock()

	if err != nil {
		return nil, fmt.Errorf("failed to add DataFrame support: %w", err)
	}

	// Create DataFrame from input data
	inputWithDF := map[string]interface{}{
		"df": data,
		"DataFrame": "pandas.DataFrame",
	}

	return epr.Execute(code, inputWithDF, nil)
}

// AddModule adds a custom module to the Python runtime
func (epr *EmbeddedPythonRuntime) AddModule(name string, module map[string]interface{}) error {
	epr.mu.Lock()
	defer epr.mu.Unlock()

	if !epr.initialized {
		return fmt.Errorf("Python runtime not initialized")
	}

	// Add module to Python runtime
	modules := epr.pyEngine.Get("modules")
	if modules == nil {
		return fmt.Errorf("modules object not found")
	}

	modulesObj := modules.ToObject(epr.vm)
	modulesObj.Set(name, module)

	return nil
}

// Cleanup cleans up the embedded Python runtime
func (epr *EmbeddedPythonRuntime) Cleanup() error {
	epr.mu.Lock()
	defer epr.mu.Unlock()

	epr.vm = nil
	epr.pyEngine = nil
	epr.initialized = false

	return nil
}
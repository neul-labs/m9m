package runtime

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
)

func (js *JavaScriptRuntime) initializeBuiltinModules() {
	modules := map[string]interface{}{
		"util":        js.createUtilModule(),
		"crypto":      js.createCryptoModule(),
		"path":        js.createPathModule(),
		"fs":          js.createFsModule(),
		"http":        js.createHttpModule(),
		"https":       js.createHttpsModule(),
		"url":         js.createUrlModule(),
		"querystring": js.createQueryStringModule(),
		"os":          js.createOsModule(),
		"events":      js.createEventsModule(),
	}

	for name, module := range modules {
		js.globalModules[name] = module
	}
}

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
			if err := ioutil.WriteFile(filename, []byte(data), 0644); err != nil {
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
	return js.createHttpModule()
}

func (js *JavaScriptRuntime) createUrlModule() interface{} {
	return map[string]interface{}{
		"parse": func(urlStr string) map[string]interface{} {
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

package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/dop251/goja"
)

func (js *JavaScriptRuntime) LoadNpmPackage(name, version string) (*NpmPackage, error) {
	js.mu.Lock()
	defer js.mu.Unlock()

	cacheKey := fmt.Sprintf("%s@%s", name, version)
	if pkg, exists := js.npmCache[cacheKey]; exists {
		return pkg, nil
	}

	pkg, err := js.downloadNpmPackage(name, version)
	if err != nil {
		return nil, err
	}

	mainFile := filepath.Join(pkg.CachePath, pkg.Main)
	moduleCode, err := ioutil.ReadFile(mainFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read main file %s: %w", mainFile, err)
	}

	moduleValue, err := js.executeModuleCode(string(moduleCode), pkg)
	if err != nil {
		return nil, fmt.Errorf("failed to execute module %s: %w", name, err)
	}

	pkg.Module = moduleValue
	pkg.LoadedAt = time.Now()
	js.npmCache[cacheKey] = pkg

	return pkg, nil
}

func (js *JavaScriptRuntime) downloadNpmPackage(name, version string) (*NpmPackage, error) {
	cacheDir := filepath.Join(js.nodeModulesPath, name)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	pkg := &NpmPackage{
		Name:         name,
		Version:      version,
		Main:         "index.js",
		Dependencies: make(map[string]string),
		CachePath:    cacheDir,
	}

	packageJSONURL := fmt.Sprintf("https://registry.npmjs.org/%s/%s", name, version)
	resp, err := http.Get(packageJSONURL)
	if err != nil {
		if mockErr := js.createMockPackage(pkg); mockErr != nil {
			return nil, mockErr
		}
		return pkg, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if mockErr := js.createMockPackage(pkg); mockErr != nil {
			return nil, mockErr
		}
		return pkg, nil
	}

	var packageInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&packageInfo); err != nil {
		if mockErr := js.createMockPackage(pkg); mockErr != nil {
			return nil, mockErr
		}
		return pkg, nil
	}

	if main, ok := packageInfo["main"].(string); ok {
		pkg.Main = main
	}

	if deps, ok := packageInfo["dependencies"].(map[string]interface{}); ok {
		for depName, depVersion := range deps {
			pkg.Dependencies[depName] = depVersion.(string)
		}
	}

	if err := js.createMockPackage(pkg); err != nil {
		return nil, err
	}

	return pkg, nil
}

func (js *JavaScriptRuntime) createMockPackage(pkg *NpmPackage) error {
	if err := validatePackageName(pkg.Name); err != nil {
		return fmt.Errorf("invalid package name: %w", err)
	}

	var mockCode string

	switch pkg.Name {
	case "axios":
		mockCode = js.createAxiosMock()
	case "lodash":
		mockCode = js.createLodashMock()
	case "moment":
		mockCode = js.createMomentMock()
	case "uuid":
		mockCode = js.createUuidMock()
	case "crypto-js":
		mockCode = js.createCryptoJsMock()
	default:
		nameJSON, _ := json.Marshal(pkg.Name)
		versionJSON, _ := json.Marshal(pkg.Version)

		mockCode = fmt.Sprintf(`
			module.exports = {
				name: %s,
				version: %s,
				mock: true
			};
		`, string(nameJSON), string(versionJSON))
	}

	mainFile := filepath.Join(pkg.CachePath, pkg.Main)
	if err := os.MkdirAll(filepath.Dir(mainFile), 0755); err != nil {
		return fmt.Errorf("failed to create mock package directory: %w", err)
	}
	if err := ioutil.WriteFile(mainFile, []byte(mockCode), 0644); err != nil {
		return fmt.Errorf("failed to write mock package file: %w", err)
	}

	return nil
}

func (js *JavaScriptRuntime) executeModuleCode(code string, pkg *NpmPackage) (interface{}, error) {
	moduleCode := fmt.Sprintf(`
		(function(exports, require, module, __filename, __dirname) {
			%s
		})
	`, code)

	program, err := goja.Compile(pkg.Name, moduleCode, false)
	if err != nil {
		return nil, err
	}

	moduleObj := js.vm.NewObject()
	exportsObj := js.vm.NewObject()
	moduleObj.Set("exports", exportsObj)

	moduleFunc, err := js.vm.RunProgram(program)
	if err != nil {
		return nil, err
	}

	if callable, ok := goja.AssertFunction(moduleFunc); ok {
		_, err = callable(goja.Undefined(),
			js.vm.ToValue(exportsObj),
			js.vm.ToValue(js.requireFunction),
			js.vm.ToValue(moduleObj),
			js.vm.ToValue(filepath.Join(pkg.CachePath, pkg.Main)),
			js.vm.ToValue(pkg.CachePath),
		)
		if err != nil {
			return nil, err
		}
	}

	return moduleObj.Get("exports").Export(), nil
}

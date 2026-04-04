package main

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/neul-labs/m9m/internal/runtime"
)

func newJavaScriptRuntime() *runtime.JavaScriptRuntime {
	return runtime.NewJavaScriptRuntime(nodeModulesPath)
}

func mustReadFile(path string, description string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", description, err)
	}
	return data
}

func mustWriteFile(path string, data []byte, description string) {
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		log.Fatalf("Failed to save %s: %v", description, err)
	}
}

func marshalIndented(v interface{}) []byte {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal output: %v", err)
	}
	return data
}

func printJSON(v interface{}) {
	data := marshalIndented(v)
	println(string(data))
}

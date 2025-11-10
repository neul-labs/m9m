package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

// PluginHandler handles HTTP requests for plugin management
type PluginHandler struct {
	registry *PluginRegistry
}

// NewPluginHandler creates a new plugin handler
func NewPluginHandler(registry *PluginRegistry) *PluginHandler {
	return &PluginHandler{
		registry: registry,
	}
}

// RegisterRoutes registers plugin management routes
func (h *PluginHandler) RegisterRoutes(router *mux.Router) {
	// Plugin management endpoints
	router.HandleFunc("/api/plugins", h.ListPlugins).Methods("GET")
	router.HandleFunc("/api/plugins/stats", h.GetStats).Methods("GET")
	router.HandleFunc("/api/plugins/reload", h.ReloadPlugins).Methods("POST")
	router.HandleFunc("/api/plugins/upload", h.UploadPlugin).Methods("POST")
	router.HandleFunc("/api/plugins/{name}", h.GetPlugin).Methods("GET")
	router.HandleFunc("/api/plugins/{name}/reload", h.ReloadSinglePlugin).Methods("POST")
	router.HandleFunc("/api/plugins/{name}", h.DeletePlugin).Methods("DELETE")
}

// ListPlugins returns a list of all loaded plugins
func (h *PluginHandler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	plugins := h.registry.ListPlugins()

	response := map[string]interface{}{
		"success": true,
		"count":   len(plugins),
		"plugins": plugins,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetStats returns statistics about loaded plugins
func (h *PluginHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.registry.GetStats()

	response := map[string]interface{}{
		"success": true,
		"stats":   stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetPlugin returns details about a specific plugin
func (h *PluginHandler) GetPlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	plugin, exists := h.registry.GetPlugin(name)
	if !exists {
		h.sendError(w, http.StatusNotFound, fmt.Sprintf("Plugin %s not found", name))
		return
	}

	description := plugin.GetDescription()
	response := map[string]interface{}{
		"success": true,
		"plugin": map[string]interface{}{
			"name":        description.Name,
			"description": description.Description,
			"category":    description.Category,
			"type":        plugin.GetType(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ReloadPlugins reloads all plugins
func (h *PluginHandler) ReloadPlugins(w http.ResponseWriter, r *http.Request) {
	log.Printf("Plugin reload requested")

	if err := h.registry.ReloadAllPlugins(); err != nil {
		log.Printf("Failed to reload plugins: %v", err)
		h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reload plugins: %v", err))
		return
	}

	stats := h.registry.GetStats()
	response := map[string]interface{}{
		"success": true,
		"message": "All plugins reloaded successfully",
		"stats":   stats,
	}

	log.Printf("✅ Plugins reloaded successfully: %d total", stats["total"])

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ReloadSinglePlugin reloads a specific plugin
func (h *PluginHandler) ReloadSinglePlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	log.Printf("Reload requested for plugin: %s", name)

	if err := h.registry.ReloadPlugin(name); err != nil {
		log.Printf("Failed to reload plugin %s: %v", name, err)
		h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to reload plugin: %v", err))
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Plugin %s reloaded successfully", name),
	}

	log.Printf("✅ Plugin %s reloaded successfully", name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UploadPlugin handles plugin file uploads (for cluster distribution)
func (h *PluginHandler) UploadPlugin(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		h.sendError(w, http.StatusBadRequest, "Failed to parse upload")
		return
	}

	// Get the file from form data
	file, header, err := r.FormFile("plugin")
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "No file provided")
		return
	}
	defer file.Close()

	// Validate file extension
	filename := header.Filename
	validExtensions := []string{".js", ".grpc.yaml", ".rest.yaml"}
	isValid := false
	for _, ext := range validExtensions {
		if strings.HasSuffix(filename, ext) {
			isValid = true
			break
		}
	}

	if !isValid {
		h.sendError(w, http.StatusBadRequest, "Invalid file type. Must be .js, .grpc.yaml, or .rest.yaml")
		return
	}

	// Get plugin directory
	pluginDir := h.registry.GetPluginDirectory()
	if pluginDir == "" {
		h.sendError(w, http.StatusInternalServerError, "No plugin directory configured")
		return
	}

	// Create destination file path
	destPath := filepath.Join(pluginDir, filename)

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create file: %v", err))
		return
	}
	defer destFile.Close()

	// Copy uploaded file to destination
	written, err := io.Copy(destFile, file)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to save file: %v", err))
		return
	}

	log.Printf("✅ Plugin uploaded: %s (%d bytes)", filename, written)

	response := map[string]interface{}{
		"success":  true,
		"message":  "Plugin uploaded successfully",
		"filename": filename,
		"size":     written,
		"path":     destPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeletePlugin removes a plugin file
func (h *PluginHandler) DeletePlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	plugin, exists := h.registry.GetPlugin(name)
	if !exists {
		h.sendError(w, http.StatusNotFound, fmt.Sprintf("Plugin %s not found", name))
		return
	}

	// Get the file path
	var filePath string
	switch p := plugin.(type) {
	case *JavaScriptNodePlugin:
		filePath = p.FilePath
	case *GRPCNodePlugin:
		filePath = p.ConfigPath
	case *RESTNodePlugin:
		filePath = p.ConfigPath
	}

	// Delete the file
	if err := os.Remove(filePath); err != nil {
		h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete file: %v", err))
		return
	}

	log.Printf("✅ Plugin file deleted: %s", filePath)

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Plugin %s deleted. Call /api/plugins/reload to complete removal.", name),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendError sends an error response
func (h *PluginHandler) sendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

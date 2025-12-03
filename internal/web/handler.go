/*
Package web provides static file serving for the Vue.js frontend.

This package supports two modes:
1. Embedded mode: Frontend assets are embedded in the Go binary using go:embed
2. Development mode: Serves files from the filesystem (for hot-reload during development)
*/
package web

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

//go:embed dist/*
var embeddedFS embed.FS

// Handler provides HTTP handlers for serving the frontend
type Handler struct {
	fileServer http.Handler
	devMode    bool
	devPath    string
}

// NewHandler creates a new frontend handler
// If devPath is provided, it serves from filesystem (development mode)
// Otherwise, it serves from embedded files (production mode)
func NewHandler(devPath string) *Handler {
	h := &Handler{
		devMode: devPath != "",
		devPath: devPath,
	}

	if h.devMode {
		log.Printf("Frontend: Development mode (serving from %s)", devPath)
		h.fileServer = http.FileServer(http.Dir(devPath))
	} else {
		log.Printf("Frontend: Production mode (serving embedded files)")
		distFS, err := fs.Sub(embeddedFS, "dist")
		if err != nil {
			log.Printf("Warning: Failed to access embedded frontend: %v", err)
			return h
		}
		h.fileServer = http.FileServer(http.FS(distFS))
	}

	return h
}

// RegisterRoutes registers frontend routes with the router
func (h *Handler) RegisterRoutes(router *mux.Router) {
	// Serve static assets
	router.PathPrefix("/assets/").Handler(h.fileServer)

	// Serve favicon and other root static files
	router.HandleFunc("/favicon.ico", h.serveFavicon)
	router.HandleFunc("/favicon.svg", h.serveFavicon)

	// Catch-all for SPA routing - must be last
	router.PathPrefix("/").HandlerFunc(h.serveSPA)
}

// serveSPA serves the index.html for all non-API, non-asset routes
// This enables client-side routing in the Vue.js SPA
func (h *Handler) serveSPA(w http.ResponseWriter, r *http.Request) {
	// Skip API routes
	if strings.HasPrefix(r.URL.Path, "/api/") ||
		strings.HasPrefix(r.URL.Path, "/health") ||
		strings.HasPrefix(r.URL.Path, "/webhook/") {
		http.NotFound(w, r)
		return
	}

	// Check if this is a static file request
	if h.isStaticFile(r.URL.Path) {
		h.fileServer.ServeHTTP(w, r)
		return
	}

	// Serve index.html for SPA routes
	h.serveIndex(w, r)
}

// serveIndex serves the index.html file
func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	var indexContent []byte
	var err error

	if h.devMode {
		indexPath := filepath.Join(h.devPath, "index.html")
		indexContent, err = os.ReadFile(indexPath)
	} else {
		indexContent, err = embeddedFS.ReadFile("dist/index.html")
	}

	if err != nil {
		log.Printf("Error reading index.html: %v", err)
		http.Error(w, "Frontend not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(indexContent)
}

// serveFavicon serves favicon files
func (h *Handler) serveFavicon(w http.ResponseWriter, r *http.Request) {
	h.fileServer.ServeHTTP(w, r)
}

// isStaticFile checks if the path is a static file (has a file extension)
func (h *Handler) isStaticFile(path string) bool {
	ext := filepath.Ext(path)
	staticExtensions := map[string]bool{
		".js":    true,
		".css":   true,
		".png":   true,
		".jpg":   true,
		".jpeg":  true,
		".gif":   true,
		".svg":   true,
		".ico":   true,
		".woff":  true,
		".woff2": true,
		".ttf":   true,
		".eot":   true,
		".map":   true,
		".json":  true,
	}
	return staticExtensions[ext]
}

// HealthCheck returns the frontend health status
func (h *Handler) HealthCheck() map[string]interface{} {
	status := map[string]interface{}{
		"mode": "embedded",
	}

	if h.devMode {
		status["mode"] = "development"
		status["path"] = h.devPath
	}

	return status
}

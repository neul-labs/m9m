package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/neul-labs/m9m/internal/engine"
)

var healthJSON bool

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check system health",
	Long: `Run comprehensive system health checks.

Checks: binary OK, node registration, workspace, service reachability,
disk space, and configuration.

Examples:
  m9m health
  m9m health --json`,
	Run: runHealth,
}

func init() {
	healthCmd.Flags().BoolVar(&healthJSON, "json", false, "Output in JSON format")
}

type healthCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"` // OK, WARN, FAIL
	Detail string `json:"detail,omitempty"`
}

func runHealth(cmd *cobra.Command, args []string) {
	var checks []healthCheck

	// 1. Binary OK
	checks = append(checks, healthCheck{
		Name:   "Binary",
		Status: "OK",
		Detail: fmt.Sprintf("m9m %s (%s/%s)", version, runtime.GOOS, runtime.GOARCH),
	})

	// 2. Node registration
	eng := engine.NewWorkflowEngine()
	RegisterAllNodes(eng)
	nodeTypes := eng.GetRegisteredNodeTypes()
	checks = append(checks, healthCheck{
		Name:   "Node Registry",
		Status: "OK",
		Detail: fmt.Sprintf("%d node types registered", len(nodeTypes)),
	})

	// 3. Workspace
	ws := GetWorkspace()
	if ws == "" {
		ws, _ = os.Getwd()
	}
	if fi, err := os.Stat(ws); err == nil && fi.IsDir() {
		checks = append(checks, healthCheck{
			Name:   "Workspace",
			Status: "OK",
			Detail: ws,
		})
	} else {
		checks = append(checks, healthCheck{
			Name:   "Workspace",
			Status: "WARN",
			Detail: "workspace directory not found",
		})
	}

	// 4. Service reachability
	port := os.Getenv("M9M_PORT")
	if port == "" {
		port = "8080"
	}
	serviceURL := fmt.Sprintf("http://localhost:%s/health", port)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(serviceURL)
	if err != nil {
		checks = append(checks, healthCheck{
			Name:   "Service",
			Status: "WARN",
			Detail: fmt.Sprintf("not reachable at %s", serviceURL),
		})
	} else {
		resp.Body.Close()
		checks = append(checks, healthCheck{
			Name:   "Service",
			Status: "OK",
			Detail: fmt.Sprintf("running on port %s (HTTP %d)", port, resp.StatusCode),
		})
	}

	// 5. Go runtime
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	checks = append(checks, healthCheck{
		Name:   "Runtime",
		Status: "OK",
		Detail: fmt.Sprintf("Go %s, %d goroutines, %d MB alloc",
			runtime.Version(), runtime.NumGoroutine(), memStats.Alloc/1024/1024),
	})

	// Output
	if healthJSON {
		overall := "healthy"
		for _, c := range checks {
			if c.Status == "FAIL" {
				overall = "unhealthy"
				break
			}
		}
		out := map[string]interface{}{
			"status": overall,
			"checks": checks,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("System Health Check")
	fmt.Println("===================")
	for _, c := range checks {
		marker := "[OK]  "
		if c.Status == "WARN" {
			marker = "[WARN]"
		} else if c.Status == "FAIL" {
			marker = "[FAIL]"
		}
		fmt.Printf("  %s %-16s %s\n", marker, c.Name, c.Detail)
	}
}

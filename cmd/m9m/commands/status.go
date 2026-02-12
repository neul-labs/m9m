package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/neul-labs/m9m/internal/workspace"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show workspace and service status",
	Run:   runStatus,
}

func runStatus(cmd *cobra.Command, args []string) {
	mgr, err := workspace.NewManager()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	current, _ := mgr.GetCurrent()

	status := map[string]interface{}{
		"currentWorkspace": current,
		"m9mDir":           mgr.GetBaseDir(),
		"serviceRunning":   false, // TODO: check daemon status
	}

	if current != "" {
		ws, err := mgr.Get(current)
		if err == nil {
			status["workspace"] = map[string]interface{}{
				"name":        ws.Name,
				"path":        ws.Path,
				"storageType": ws.Config.StorageType,
				"createdAt":   ws.CreatedAt,
				"lastUsed":    ws.LastUsed,
			}
		}
	}

	if outputFlag == "json" {
		data, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("m9m Status")
	fmt.Println("==========")
	fmt.Printf("Config directory: %s\n", mgr.GetBaseDir())
	fmt.Printf("Service running: %v\n", status["serviceRunning"])
	fmt.Println()

	if current == "" {
		fmt.Println("No workspace selected.")
		fmt.Println("\nRun 'm9m init' to create and select a workspace.")
		return
	}

	fmt.Printf("Current Workspace: %s\n", current)
	if ws, ok := status["workspace"].(map[string]interface{}); ok {
		fmt.Printf("  Path: %s\n", ws["path"])
		fmt.Printf("  Storage: %s\n", ws["storageType"])
	}
}

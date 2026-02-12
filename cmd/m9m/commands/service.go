package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/neul-labs/m9m/internal/service"
)

var (
	serviceBackground  bool
	serviceIdleTimeout int
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the m9m background service",
	Long: `Manage the m9m background service (daemon).

The service runs in the background and handles workflow operations.
It automatically starts when needed and exits after idle timeout.`,
}

var serviceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the background service",
	Long: `Start the m9m background service (daemon).

The daemon listens on a Unix socket and handles workflow operations
for all workspaces. It automatically exits after an idle timeout.`,
	Run: runServiceStart,
}

var serviceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background service",
	Run:   runServiceStop,
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check service status",
	Run:   runServiceStatus,
}

var serviceRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the background service",
	Run:   runServiceRestart,
}

func init() {
	serviceStartCmd.Flags().BoolVar(&serviceBackground, "background", false, "Run in background (daemon mode)")
	serviceStartCmd.Flags().IntVar(&serviceIdleTimeout, "idle-timeout", 300, "Idle timeout in seconds before auto-exit")

	serviceCmd.AddCommand(serviceStartCmd)
	serviceCmd.AddCommand(serviceStopCmd)
	serviceCmd.AddCommand(serviceStatusCmd)
	serviceCmd.AddCommand(serviceRestartCmd)
}

func runServiceStart(cmd *cobra.Command, args []string) {
	if serviceBackground {
		// This is called when starting in background mode
		// The actual daemonization is handled by the client
		startDaemonForeground()
		return
	}

	// Start in foreground
	fmt.Println("Starting m9m service in foreground...")
	fmt.Println("Press Ctrl+C to stop")
	startDaemonForeground()
}

func startDaemonForeground() {
	config := &service.DaemonConfig{
		SocketPath:  service.GetSocketPath(),
		IdleTimeout: time.Duration(serviceIdleTimeout) * time.Second,
	}

	daemon, err := service.NewDaemon(config)
	if err != nil {
		fmt.Printf("Error: Failed to create daemon: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := daemon.Start(ctx); err != nil {
		fmt.Printf("Error: Daemon failed: %v\n", err)
		os.Exit(1)
	}
}

func runServiceStop(cmd *cobra.Command, args []string) {
	client := service.NewClient(nil)

	if !client.IsRunning() {
		fmt.Println("Service is not running.")
		return
	}

	fmt.Println("Stopping m9m service...")
	if err := client.StopDaemon(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Service stopped.")
}

func runServiceStatus(cmd *cobra.Command, args []string) {
	client := service.NewClient(nil)

	if !client.IsRunning() {
		fmt.Println("Service is not running.")
		fmt.Println("\nTo start: m9m service start")
		fmt.Println("Note: The service starts automatically when you run workflow commands.")
		return
	}

	status, err := client.Status()
	if err != nil {
		fmt.Printf("Error: Failed to get status: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("m9m Service Status")
	fmt.Println("==================")
	fmt.Printf("Running: %v\n", status["running"])
	fmt.Printf("Socket: %v\n", status["socketPath"])
	fmt.Printf("Idle Timeout: %v\n", status["idleTimeout"])

	if current, ok := status["currentWorkspace"].(string); ok && current != "" {
		fmt.Printf("Current Workspace: %s\n", current)
	}

	if loaded, ok := status["loadedWorkspaces"].([]interface{}); ok && len(loaded) > 0 {
		fmt.Println("\nLoaded Workspaces:")
		for _, ws := range loaded {
			fmt.Printf("  - %v\n", ws)
		}
	}
}

func runServiceRestart(cmd *cobra.Command, args []string) {
	client := service.NewClient(nil)

	if client.IsRunning() {
		fmt.Println("Stopping service...")
		if err := client.StopDaemon(); err != nil {
			fmt.Printf("Warning: Failed to stop cleanly: %v\n", err)
		}
		// Wait for socket to be removed
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("Starting service...")
	if err := client.StartDaemon(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Wait for it to be ready
	time.Sleep(500 * time.Millisecond)
	if client.IsRunning() {
		fmt.Println("Service restarted successfully.")
	} else {
		fmt.Println("Warning: Service may still be starting...")
	}
}

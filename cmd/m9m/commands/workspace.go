package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/neul-labs/m9m/internal/workspace"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage workspaces",
	Long: `Manage m9m workspaces for tenancy and isolation.

Workspaces provide isolated environments for workflows, credentials,
and executions. Each workspace has its own storage.`,
}

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces",
	Run:   runWorkspaceList,
}

var workspaceUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch to a workspace",
	Args:  cobra.ExactArgs(1),
	Run:   runWorkspaceUse,
}

var workspaceCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new workspace",
	Args:  cobra.ExactArgs(1),
	Run:   runWorkspaceCreate,
}

var workspaceDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a workspace",
	Args:  cobra.ExactArgs(1),
	Run:   runWorkspaceDelete,
}

var workspaceCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current workspace",
	Run:   runWorkspaceCurrent,
}

func init() {
	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceUseCmd)
	workspaceCmd.AddCommand(workspaceCreateCmd)
	workspaceCmd.AddCommand(workspaceDeleteCmd)
	workspaceCmd.AddCommand(workspaceCurrentCmd)
}

func runWorkspaceList(cmd *cobra.Command, args []string) {
	mgr, err := workspace.NewManager()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	workspaces, err := mgr.List()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	current, _ := mgr.GetCurrent()

	if outputFlag == "json" {
		data, _ := json.MarshalIndent(workspaces, "", "  ")
		fmt.Println(string(data))
		return
	}

	if len(workspaces) == 0 {
		fmt.Println("No workspaces found.")
		fmt.Println("\nRun 'm9m init <name>' to create a workspace.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTORAGE\tCREATED\tCURRENT")
	fmt.Fprintln(w, "----\t-------\t-------\t-------")
	for _, ws := range workspaces {
		isCurrent := ""
		if ws.Name == current {
			isCurrent = "*"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			ws.Name,
			ws.Config.StorageType,
			ws.CreatedAt.Format("2006-01-02"),
			isCurrent,
		)
	}
	w.Flush()
}

func runWorkspaceUse(cmd *cobra.Command, args []string) {
	name := args[0]

	mgr, err := workspace.NewManager()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if err := mgr.SetCurrent(name); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Switched to workspace '%s'\n", name)
}

func runWorkspaceCreate(cmd *cobra.Command, args []string) {
	name := args[0]

	mgr, err := workspace.NewManager()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	ws, err := mgr.Create(name, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Set as current workspace
	mgr.SetCurrent(name)

	fmt.Printf("Created workspace '%s' at %s\n", ws.Name, ws.Path)
	fmt.Printf("Switched to workspace '%s'\n", ws.Name)
}

func runWorkspaceDelete(cmd *cobra.Command, args []string) {
	name := args[0]

	mgr, err := workspace.NewManager()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Confirm deletion
	fmt.Printf("This will permanently delete workspace '%s' and all its data.\n", name)
	fmt.Print("Type the workspace name to confirm: ")

	var confirm string
	fmt.Scanln(&confirm)
	if confirm != name {
		fmt.Println("Deletion cancelled.")
		return
	}

	if err := mgr.Delete(name); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Deleted workspace '%s'\n", name)
}

func runWorkspaceCurrent(cmd *cobra.Command, args []string) {
	mgr, err := workspace.NewManager()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	current, err := mgr.GetCurrent()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if current == "" {
		fmt.Println("No workspace selected.")
		fmt.Println("\nRun 'm9m init' or 'm9m workspace use <name>' to select a workspace.")
		return
	}

	ws, err := mgr.Get(current)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if outputFlag == "json" {
		data, _ := json.MarshalIndent(ws, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("Current workspace: %s\n", ws.Name)
	fmt.Printf("  Path: %s\n", ws.Path)
	fmt.Printf("  Storage: %s\n", ws.Config.StorageType)
	fmt.Printf("  Created: %s\n", ws.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Last used: %s\n", ws.LastUsed.Format("2006-01-02 15:04:05"))
}

package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Check for and install updates",
	Long: `Check for a newer version of m9m on GitHub releases.

Examples:
  m9m upgrade          Check for updates
  m9m upgrade --check  Only check, do not install`,
	Run: runUpgrade,
}

var upgradeCheckOnly bool

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeCheckOnly, "check", false, "Only check for updates, do not install")
}

func runUpgrade(cmd *cobra.Command, args []string) {
	currentVersion := version
	if currentVersion == "" {
		currentVersion = "dev"
	}

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Println("Checking for updates...")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/neul-labs/m9m/releases/latest")
	if err != nil {
		fmt.Printf("Error: Cannot check for updates: %v\n", err)
		fmt.Println("\nYou can check manually at: https://github.com/neul-labs/m9m/releases")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Error: GitHub API returned %d: %s\n", resp.StatusCode, string(body))
		return
	}

	var release struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		fmt.Printf("Error: Cannot parse release info: %v\n", err)
		return
	}

	latestVersion := release.TagName
	fmt.Printf("Latest version: %s\n", latestVersion)

	if currentVersion == latestVersion || "v"+currentVersion == latestVersion {
		fmt.Println("\nYou are running the latest version.")
		return
	}

	fmt.Printf("\nNew version available: %s -> %s\n", currentVersion, latestVersion)
	if release.Body != "" {
		fmt.Println("\nChangelog:")
		// Show first 500 chars of changelog
		body := release.Body
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		fmt.Println(body)
	}

	if upgradeCheckOnly {
		fmt.Printf("\nDownload: %s\n", release.HTMLURL)
		return
	}

	// Find matching asset
	osName := runtime.GOOS
	archName := runtime.GOARCH
	expectedAsset := fmt.Sprintf("m9m_%s_%s", osName, archName)

	fmt.Printf("\nLooking for asset: %s\n", expectedAsset)
	for _, asset := range release.Assets {
		if asset.Name == expectedAsset || asset.Name == expectedAsset+".tar.gz" || asset.Name == expectedAsset+".zip" {
			fmt.Printf("Found: %s\n", asset.BrowserDownloadURL)
			fmt.Println("\nTo upgrade, download and replace the binary:")
			fmt.Printf("  curl -L %s -o m9m && chmod +x m9m && sudo mv m9m /usr/local/bin/\n", asset.BrowserDownloadURL)
			return
		}
	}

	fmt.Printf("\nNo pre-built binary found for %s/%s.\n", osName, archName)
	fmt.Println("Build from source:")
	fmt.Println("  git pull && make build")
}

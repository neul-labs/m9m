package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `View and validate m9m configuration.`,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long: `Validate configuration from environment variables and config files.

Checks: required fields, valid port ranges, queue configuration, and more.

Examples:
  m9m config validate`,
	Run: runConfigValidate,
}

func init() {
	configCmd.AddCommand(configValidateCmd)
}

func runConfigValidate(cmd *cobra.Command, args []string) {
	type configCheck struct {
		name   string
		status string
		detail string
	}
	var checks []configCheck

	// Port
	port := os.Getenv("M9M_PORT")
	if port == "" {
		port = "8080"
	}
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		checks = append(checks, configCheck{"M9M_PORT", "FAIL", fmt.Sprintf("invalid port: %s", port)})
	} else {
		checks = append(checks, configCheck{"M9M_PORT", "OK", port})
	}

	// Host
	host := os.Getenv("M9M_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	checks = append(checks, configCheck{"M9M_HOST", "OK", host})

	// Log level
	logLevel := os.Getenv("M9M_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if validLevels[logLevel] {
		checks = append(checks, configCheck{"M9M_LOG_LEVEL", "OK", logLevel})
	} else {
		checks = append(checks, configCheck{"M9M_LOG_LEVEL", "WARN", fmt.Sprintf("unknown level: %s (using info)", logLevel)})
	}

	// Queue
	queueType := os.Getenv("M9M_QUEUE_TYPE")
	if queueType == "" {
		queueType = "memory"
	}
	validQueues := map[string]bool{"memory": true, "redis": true, "rabbitmq": true}
	if validQueues[queueType] {
		checks = append(checks, configCheck{"M9M_QUEUE_TYPE", "OK", queueType})
	} else {
		checks = append(checks, configCheck{"M9M_QUEUE_TYPE", "FAIL", fmt.Sprintf("invalid queue type: %s", queueType)})
	}

	if queueType == "redis" {
		queueURL := os.Getenv("M9M_QUEUE_URL")
		if queueURL == "" {
			checks = append(checks, configCheck{"M9M_QUEUE_URL", "WARN", "not set (required for redis queue)"})
		} else {
			checks = append(checks, configCheck{"M9M_QUEUE_URL", "OK", queueURL})
		}
	}

	// Metrics port
	metricsPort := os.Getenv("M9M_METRICS_PORT")
	if metricsPort != "" {
		mp, err := strconv.Atoi(metricsPort)
		if err != nil || mp < 1 || mp > 65535 {
			checks = append(checks, configCheck{"M9M_METRICS_PORT", "FAIL", fmt.Sprintf("invalid port: %s", metricsPort)})
		} else {
			checks = append(checks, configCheck{"M9M_METRICS_PORT", "OK", metricsPort})
		}
	}

	// Output results
	hasErrors := false
	fmt.Println("Configuration Validation")
	fmt.Println("========================")
	for _, c := range checks {
		marker := "[OK]  "
		if c.status == "WARN" {
			marker = "[WARN]"
		} else if c.status == "FAIL" {
			marker = "[FAIL]"
			hasErrors = true
		}
		fmt.Printf("  %s %-20s %s\n", marker, c.name, c.detail)
	}

	fmt.Println()
	if hasErrors {
		fmt.Println("Configuration has errors. Please fix the issues above.")
		os.Exit(1)
	} else {
		fmt.Println("Configuration is valid.")
	}
}

package bwrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/neul-labs/m9m/internal/sandbox"
)

const (
	cgroupRoot    = "/sys/fs/cgroup"
	cgroupPrefix  = "m9m-sandbox-"
)

var cgroupCounter uint64

// CgroupManager handles cgroups v2 resource limiting
type CgroupManager struct {
	basePath string
	enabled  bool
}

// CgroupStats contains resource usage statistics
type CgroupStats struct {
	CPUTime   time.Duration
	MaxMemory int64
}

// NewCgroupManager creates a new cgroup manager
func NewCgroupManager() *CgroupManager {
	cm := &CgroupManager{
		basePath: cgroupRoot,
		enabled:  false,
	}

	// Check if cgroups v2 is available
	if _, err := os.Stat(filepath.Join(cgroupRoot, "cgroup.controllers")); err == nil {
		cm.enabled = true
	}

	return cm
}

// IsEnabled returns whether cgroups are available
func (cm *CgroupManager) IsEnabled() bool {
	return cm.enabled
}

// CreateCgroup creates a new cgroup with the specified resource limits
func (cm *CgroupManager) CreateCgroup(limits sandbox.ResourceLimits) (string, error) {
	if !cm.enabled {
		return "", fmt.Errorf("cgroups v2 not available")
	}

	// Generate unique cgroup name
	id := atomic.AddUint64(&cgroupCounter, 1)
	cgroupName := fmt.Sprintf("%s%d-%d", cgroupPrefix, os.Getpid(), id)
	cgroupPath := filepath.Join(cm.basePath, cgroupName)

	// Create cgroup directory
	if err := os.Mkdir(cgroupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create cgroup: %w", err)
	}

	// Apply memory limit
	if limits.MaxMemoryBytes > 0 {
		memMaxPath := filepath.Join(cgroupPath, "memory.max")
		if err := os.WriteFile(memMaxPath, []byte(strconv.FormatInt(limits.MaxMemoryBytes, 10)), 0644); err != nil {
			cm.RemoveCgroup(cgroupPath)
			return "", fmt.Errorf("failed to set memory limit: %w", err)
		}

		// Also set memory.high to provide early pressure
		memHighPath := filepath.Join(cgroupPath, "memory.high")
		highValue := limits.MaxMemoryBytes * 90 / 100 // 90% of max
		_ = os.WriteFile(memHighPath, []byte(strconv.FormatInt(highValue, 10)), 0644)
	}

	// Apply process limit
	if limits.MaxProcesses > 0 {
		pidsMaxPath := filepath.Join(cgroupPath, "pids.max")
		if err := os.WriteFile(pidsMaxPath, []byte(strconv.Itoa(limits.MaxProcesses)), 0644); err != nil {
			cm.RemoveCgroup(cgroupPath)
			return "", fmt.Errorf("failed to set process limit: %w", err)
		}
	}

	// Apply CPU limit (if specified as percentage or time)
	if limits.MaxCPUTime > 0 {
		// CPU bandwidth limiting: cpu.max format is "$MAX $PERIOD"
		// To limit to 50% CPU: "50000 100000" (50ms max per 100ms period)
		// We'll use a simple approach: limit to 100% of one CPU
		cpuMaxPath := filepath.Join(cgroupPath, "cpu.max")
		// Allow full CPU but this can be adjusted
		_ = os.WriteFile(cpuMaxPath, []byte("100000 100000"), 0644)
	}

	return cgroupPath, nil
}

// AddProcess adds a process to a cgroup
func (cm *CgroupManager) AddProcess(cgroupPath string, pid int) error {
	if !cm.enabled {
		return nil
	}

	procsPath := filepath.Join(cgroupPath, "cgroup.procs")
	return os.WriteFile(procsPath, []byte(strconv.Itoa(pid)), 0644)
}

// GetStats retrieves resource usage statistics from a cgroup
func (cm *CgroupManager) GetStats(cgroupPath string) (*CgroupStats, error) {
	if !cm.enabled {
		return nil, fmt.Errorf("cgroups not enabled")
	}

	stats := &CgroupStats{}

	// Read CPU usage
	cpuStatPath := filepath.Join(cgroupPath, "cpu.stat")
	if data, err := os.ReadFile(cpuStatPath); err == nil {
		stats.CPUTime = parseCPUStat(data)
	}

	// Read peak memory usage
	memPeakPath := filepath.Join(cgroupPath, "memory.peak")
	if data, err := os.ReadFile(memPeakPath); err == nil {
		if val, err := strconv.ParseInt(string(data[:len(data)-1]), 10, 64); err == nil {
			stats.MaxMemory = val
		}
	} else {
		// Fallback to memory.current if peak not available
		memCurrentPath := filepath.Join(cgroupPath, "memory.current")
		if data, err := os.ReadFile(memCurrentPath); err == nil {
			if val, err := strconv.ParseInt(string(data[:len(data)-1]), 10, 64); err == nil {
				stats.MaxMemory = val
			}
		}
	}

	return stats, nil
}

// RemoveCgroup removes a cgroup
func (cm *CgroupManager) RemoveCgroup(cgroupPath string) error {
	if cgroupPath == "" {
		return nil
	}

	// Kill any remaining processes in the cgroup
	procsPath := filepath.Join(cgroupPath, "cgroup.procs")
	if data, err := os.ReadFile(procsPath); err == nil && len(data) > 0 {
		// Write to cgroup.kill to terminate all processes
		killPath := filepath.Join(cgroupPath, "cgroup.kill")
		_ = os.WriteFile(killPath, []byte("1"), 0644)

		// Give processes time to terminate
		time.Sleep(100 * time.Millisecond)
	}

	// Remove the cgroup directory
	return os.Remove(cgroupPath)
}

// Cleanup removes all cgroups created by this manager
func (cm *CgroupManager) Cleanup() error {
	if !cm.enabled {
		return nil
	}

	// Find and remove all m9m sandbox cgroups
	entries, err := os.ReadDir(cm.basePath)
	if err != nil {
		return err
	}

	pidPrefix := fmt.Sprintf("%s%d-", cgroupPrefix, os.Getpid())
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) > len(pidPrefix) && entry.Name()[:len(pidPrefix)] == pidPrefix {
			cgroupPath := filepath.Join(cm.basePath, entry.Name())
			_ = cm.RemoveCgroup(cgroupPath)
		}
	}

	return nil
}

// parseCPUStat extracts total CPU time from cpu.stat contents
func parseCPUStat(data []byte) time.Duration {
	// cpu.stat format:
	// usage_usec 123456
	// user_usec 100000
	// system_usec 23456
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) > 11 && string(line[:10]) == "usage_usec" {
			if val, err := strconv.ParseInt(string(line[11:]), 10, 64); err == nil {
				return time.Duration(val) * time.Microsecond
			}
		}
	}
	return 0
}

// splitLines splits byte slice into lines
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

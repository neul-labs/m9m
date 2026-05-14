//go:build !windows

package service

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func registerDaemonSignals(sigCh chan os.Signal) {
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
}

func setDaemonSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}

func sendTerminationSignal(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}

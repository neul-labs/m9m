//go:build windows

package service

import (
	"os"
	"os/exec"
	"os/signal"
)

func registerDaemonSignals(sigCh chan os.Signal) {
	signal.Notify(sigCh, os.Interrupt)
}

func setDaemonSysProcAttr(cmd *exec.Cmd) {
	// Windows doesn't support Setsid
}

func sendTerminationSignal(proc *os.Process) error {
	return proc.Kill()
}

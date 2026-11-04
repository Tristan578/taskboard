//go:build !windows

package cli

import (
	"os"
	"os/exec"
	"syscall"
)

func processRunning(pid int) bool {
	p, err := os.FindProcess(pid)
	return err == nil && p.Signal(syscall.Signal(0)) == nil
}
func stopProcess(p *os.Process) error { return p.Signal(syscall.SIGTERM) }
func detachProcess(c *exec.Cmd)       { c.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }

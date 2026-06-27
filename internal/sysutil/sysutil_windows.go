//go:build windows
// +build windows

package sysutil

import (
	"os/exec"
	"syscall"
)

// SetCmdLine sets the raw command line for Windows to avoid escaping issues
func SetCmdLine(cmd *exec.Cmd, cmdLine string) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CmdLine = cmdLine
}

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

// HideConsoleWindow configures the command to run hidden without popping up a console window
func HideConsoleWindow(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= 0x08000000 // CREATE_NO_WINDOW
}


//go:build !windows
// +build !windows

package sysutil

import (
	"os/exec"
)

// SetCmdLine is a no-op on non-Windows platforms
func SetCmdLine(cmd *exec.Cmd, cmdLine string) {
	// Not applicable on non-Windows systems
}

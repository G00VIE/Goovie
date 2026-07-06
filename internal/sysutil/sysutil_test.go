//go:build !windows

package sysutil

import (
	"os/exec"
	"testing"
)

func TestSetCmdLine_NoOp(t *testing.T) {
	// On non-Windows, SetCmdLine should be a no-op and not panic
	cmd := exec.Command("echo", "hello")
	SetCmdLine(cmd, "some arbitrary command line string")

	// If we reach here, it didn't panic - that's the test
	// The command should still function normally
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cmd execution failed: %v", err)
	}
	if string(output) == "" {
		t.Error("expected non-empty output from echo")
	}
}

func TestSetCmdLine_DoesNotModifyCommand(t *testing.T) {
	cmd := exec.Command("echo", "test")
	originalPath := cmd.Path

	SetCmdLine(cmd, "totally different command")

	// Path should remain unchanged on non-Windows
	if cmd.Path != originalPath {
		t.Error("SetCmdLine should not modify cmd.Path on non-Windows")
	}
}

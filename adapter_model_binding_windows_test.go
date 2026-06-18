//go:build windows

package main

import (
	"context"
	"testing"

	"ui_console/shared/executil"
)

func TestWindowsCLIProbeCommandsHideConsoleWindow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := executil.CommandContext(ctx, "cmd.exe", "/c", "echo", "ok")
	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr for Windows CLI probes")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected Windows CLI probes to hide the console window")
	}
	if cmd.SysProcAttr.CreationFlags == 0 {
		t.Fatal("expected Windows CLI probes to set a no-window creation flag")
	}
}

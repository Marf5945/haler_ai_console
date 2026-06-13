//go:build !windows

package executil

import (
	"context"
	"os"
	"os/exec"
)

func Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func HideWindow(cmd *exec.Cmd) {
}

// IsExecutable 回報檔案是否可執行(Unix:檢查執行權限位元)。
func IsExecutable(path string, info os.FileInfo) bool {
	if info == nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

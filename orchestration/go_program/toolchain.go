package go_program

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"ui_console/shared/executil"
)

func ResolveBundledToolchain(appRoot string) (Toolchain, error) {
	candidates := []string{
		os.Getenv("UICONSOLE_BUNDLED_GO"),
		filepath.Join(appRoot, "Contents", "Resources", "go", "bin", goExeName()),
		filepath.Join(appRoot, "resources", "go", "bin", goExeName()),
		filepath.Join(runtime.GOROOT(), "bin", goExeName()),
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			version := goVersion(candidate)
			return Toolchain{GoBinary: candidate, Version: version}, nil
		}
	}
	return Toolchain{}, fmt.Errorf("go_program: bundled Go toolchain not found")
}

func goExeName() string {
	if runtime.GOOS == "windows" {
		return "go.exe"
	}
	return "go"
}

func goVersion(goBinary string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := executil.CommandContext(ctx, goBinary, "version")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

package go_program

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"ui_console/shared/executil"
)

func Build(ctx context.Context, manifest Manifest, toolchain Toolchain, outputDir string, limits Limits) (BuildResult, error) {
	limits = limits.Normalize()
	if toolchain.GoBinary == "" {
		return BuildResult{}, fmt.Errorf("go_program: Go toolchain is required")
	}
	if err := os.MkdirAll(outputDir, 0o700); err != nil {
		return BuildResult{}, fmt.Errorf("go_program: create build dir: %w", err)
	}
	files, err := goFiles(manifest.SourceDir)
	if err != nil {
		return BuildResult{}, err
	}
	args := []string{"build", "-trimpath", "-o", filepath.Join(outputDir, "program"+exeSuffix())}
	for _, file := range files {
		rel, relErr := filepath.Rel(manifest.SourceDir, file)
		if relErr != nil {
			return BuildResult{}, relErr
		}
		args = append(args, filepath.ToSlash(rel))
	}
	bin := filepath.Join(outputDir, "program"+exeSuffix())
	buildCtx, cancel := context.WithTimeout(ctx, limits.BuildTimeout)
	defer cancel()
	cmd := executil.CommandContext(buildCtx, toolchain.GoBinary, args...)
	cmd.Dir = manifest.SourceDir
	cmd.Env = buildEnv(os.Environ(), len(manifest.VendorAllowlist) > 0)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if buildCtx.Err() != nil {
			return BuildResult{BinaryPath: bin, Stdout: stdout.String(), Stderr: stderr.String()}, fmt.Errorf("go_program: build timeout: %w", buildCtx.Err())
		}
		return BuildResult{BinaryPath: bin, Stdout: stdout.String(), Stderr: stderr.String()}, fmt.Errorf("go_program: build failed: %w", err)
	}
	return BuildResult{BinaryPath: bin, Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

func Execute(ctx context.Context, binaryPath string, inputJSON []byte, limits Limits) (ExecuteResult, error) {
	limits = limits.Normalize()
	execCtx, cancel := context.WithTimeout(ctx, limits.ExecuteTimeout)
	defer cancel()
	cmd := executil.CommandContext(execCtx, binaryPath)
	cmd.Env = []string{"PATH=", "HOME=", "TMPDIR="}
	cmd.Stdin = bytes.NewReader(inputJSON)
	stdout := newLimitedBuffer(limits.StdoutBytes)
	stderr := newLimitedBuffer(limits.StderrBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if execCtx.Err() != nil {
			return ExecuteResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, fmt.Errorf("go_program: execute timeout: %w", execCtx.Err())
		}
		return ExecuteResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, fmt.Errorf("go_program: execute failed: %w", err)
	}
	if stdout.Truncated() {
		return ExecuteResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, fmt.Errorf("go_program: stdout exceeds %d bytes", limits.StdoutBytes)
	}
	if stderr.Truncated() {
		return ExecuteResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, fmt.Errorf("go_program: stderr exceeds %d bytes", limits.StderrBytes)
	}
	sum := sha256.Sum256(stdout.Bytes())
	return ExecuteResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), StdoutSHA256: hex.EncodeToString(sum[:])}, nil
}

func buildEnv(base []string, vendorMode bool) []string {
	env := filterEnv(base, "GOPROXY", "GOFLAGS", "GO111MODULE", "GOSUMDB", "GONOSUMDB")
	env = append(env, "GOPROXY=off", "GOSUMDB=off", "GONOSUMDB=*")
	if vendorMode {
		env = append(env, "GO111MODULE=on", "GOFLAGS=-mod=vendor")
	} else {
		env = append(env, "GO111MODULE=off")
	}
	return env
}

func filterEnv(base []string, keys ...string) []string {
	block := map[string]bool{}
	for _, key := range keys {
		block[key] = true
	}
	var out []string
	for _, item := range base {
		key := item
		if idx := strings.Index(item, "="); idx >= 0 {
			key = item[:idx]
		}
		if !block[key] {
			out = append(out, item)
		}
	}
	return out
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int64
	truncated bool
}

func newLimitedBuffer(limit int64) *limitedBuffer {
	return &limitedBuffer{limit: limit}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		return len(p), nil
	}
	remaining := b.limit - int64(b.buf.Len())
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if int64(len(p)) > remaining {
		b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

func (b *limitedBuffer) Bytes() []byte   { return b.buf.Bytes() }
func (b *limitedBuffer) Truncated() bool { return b.truncated }

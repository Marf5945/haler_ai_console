package visual_learning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	DirectMLRuntimeName    = "onnxruntime-directml"
	DirectMLRuntimeVersion = "1.24.4"
	DirectMLRuntimeRID     = "win-x64"
)

var directMLRuntimeRequiredFiles = []string{
	"onnxruntime.dll",
	"onnxruntime_providers_shared.dll",
	"DirectML.dll",
}

type DirectMLRuntimeStatus struct {
	Available       bool     `json:"available"`
	RuntimeName     string   `json:"runtime_name"`
	ExpectedVersion string   `json:"expected_version"`
	RuntimeID       string   `json:"runtime_id"`
	RuntimeDir      string   `json:"runtime_dir,omitempty"`
	RequiredFiles   []string `json:"required_files"`
	MissingFiles    []string `json:"missing_files,omitempty"`
	HashChecked     bool     `json:"hash_checked,omitempty"`
	IntegrityErrors []string `json:"integrity_errors,omitempty"`
	Reason          string   `json:"reason,omitempty"`
}

type DirectMLRuntimeHashManifest struct {
	Runtime string            `json:"runtime"`
	Version string            `json:"version"`
	RID     string            `json:"rid"`
	Files   map[string]string `json:"files"`
}

func CheckDirectMLRuntime() DirectMLRuntimeStatus {
	status := DirectMLRuntimeStatus{
		RuntimeName:     DirectMLRuntimeName,
		ExpectedVersion: DirectMLRuntimeVersion,
		RuntimeID:       DirectMLRuntimeRID,
		RequiredFiles:   append([]string(nil), directMLRuntimeRequiredFiles...),
	}

	if runtime.GOOS != "windows" {
		status.Reason = "DirectML runtime is only supported on Windows"
		return status
	}

	for _, dir := range directMLRuntimeCandidateDirs() {
		missing := missingRuntimeFiles(dir, directMLRuntimeRequiredFiles)
		if len(missing) == 0 {
			status.RuntimeDir = dir
			status.HashChecked = true
			// Runtime DLLs are optional, but present DLLs must match the pinned manifest.
			status.IntegrityErrors = verifyDirectMLRuntimeHashes(dir, directMLRuntimeRequiredFiles)
			if len(status.IntegrityErrors) == 0 {
				status.Available = true
				return status
			}
			status.Reason = fmt.Sprintf("%s %s runtime failed hash verification: %v",
				DirectMLRuntimeName, DirectMLRuntimeVersion, status.IntegrityErrors)
			return status
		}
		if status.RuntimeDir == "" {
			status.RuntimeDir = dir
			status.MissingFiles = missing
		}
	}

	if status.RuntimeDir == "" {
		status.RuntimeDir = filepath.Join("assets", "runtimes", DirectMLRuntimeName, DirectMLRuntimeVersion, DirectMLRuntimeRID)
		status.MissingFiles = append([]string(nil), directMLRuntimeRequiredFiles...)
	}
	status.Reason = fmt.Sprintf("%s %s runtime is not bundled; missing %v under %s",
		DirectMLRuntimeName, DirectMLRuntimeVersion, status.MissingFiles, status.RuntimeDir)
	return status
}

func verifyDirectMLRuntimeHashes(dir string, files []string) []string {
	return verifyDirectMLRuntimeHashesWithManifest(dir, files, embeddedDirectMLRuntimeHashes)
}

func verifyDirectMLRuntimeHashesWithManifest(dir string, files []string, manifestJSON []byte) []string {
	errs := []string{}
	var manifest DirectMLRuntimeHashManifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		return []string{fmt.Sprintf("runtime hash manifest parse failed: %v", err)}
	}
	if manifest.Runtime != DirectMLRuntimeName {
		errs = append(errs, fmt.Sprintf("runtime manifest name %q does not match %q", manifest.Runtime, DirectMLRuntimeName))
	}
	if manifest.Version != DirectMLRuntimeVersion {
		errs = append(errs, fmt.Sprintf("runtime manifest version %q does not match %q", manifest.Version, DirectMLRuntimeVersion))
	}
	if manifest.RID != DirectMLRuntimeRID {
		errs = append(errs, fmt.Sprintf("runtime manifest rid %q does not match %q", manifest.RID, DirectMLRuntimeRID))
	}
	if manifest.Files == nil {
		manifest.Files = map[string]string{}
	}

	for _, file := range files {
		// An unpinned DLL is treated like a failed integrity check.
		expected := strings.TrimSpace(manifest.Files[file])
		if expected == "" {
			errs = append(errs, fmt.Sprintf("%s is not pinned in runtime hash manifest", file))
			continue
		}
		expected = strings.ToLower(expected)
		if !strings.HasPrefix(expected, "sha256:") {
			errs = append(errs, fmt.Sprintf("%s has invalid hash format; expected sha256:<hex>", file))
			continue
		}
		expectedHex := strings.TrimPrefix(expected, "sha256:")
		actualHex, err := computeRuntimeFileHash(filepath.Join(dir, file))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s hash failed: %v", file, err))
			continue
		}
		if actualHex != expectedHex {
			errs = append(errs, fmt.Sprintf("%s hash mismatch: expected sha256:%s got sha256:%s", file, expectedHex, actualHex))
		}
	}

	return errs
}

func computeRuntimeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func directMLRuntimeCandidateDirs() []string {
	rel := filepath.Join("assets", "runtimes", DirectMLRuntimeName, DirectMLRuntimeVersion, DirectMLRuntimeRID)
	dirs := []string{}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		dirs = append(dirs,
			filepath.Join(exeDir, rel),
			filepath.Clean(filepath.Join(exeDir, "..", "..", rel)),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs,
			filepath.Join(cwd, rel),
			filepath.Join(cwd, "..", "..", rel),
		)
	}
	return uniqueStrings(dirs)
}

func missingRuntimeFiles(dir string, files []string) []string {
	missing := []string{}
	for _, file := range files {
		info, err := os.Stat(filepath.Join(dir, file))
		if err != nil || info.IsDir() {
			missing = append(missing, file)
		}
	}
	return missing
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		key := filepath.Clean(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}

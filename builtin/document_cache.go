package builtin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func cacheOriginalFile(store *Store, sourcePath string) (displayName, cachedPath, originalHash, w3aID string, err error) {
	originalHash, err = hashFile(sourcePath)
	if err != nil {
		return "", "", "", "", err
	}

	if err := os.MkdirAll(store.OriginalsDir(), 0o700); err != nil {
		return "", "", "", "", fmt.Errorf("document_cache: mkdir originals: %w", err)
	}

	displayName = uniqueCacheName(store.OriginalsDir(), filepath.Base(sourcePath))
	cachedPath = filepath.Join(store.OriginalsDir(), displayName)
	if err := copyRegularFile(sourcePath, cachedPath); err != nil {
		return "", "", "", "", err
	}
	_ = copySidecarIfPresent(sourcePath, cachedPath)

	return displayName, cachedPath, originalHash, "w3a-doc-" + shortHash(originalHash), nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("document_cache: open hash source: %w", err)
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("document_cache: hash source: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyRegularFile(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("document_cache: open source: %w", err)
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("document_cache: create target: %w", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("document_cache: copy target: %w", err)
	}
	return nil
}

func copyRegularFileOverwrite(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("document_cache: open source: %w", err)
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("document_cache: create target: %w", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("document_cache: copy target: %w", err)
	}
	return nil
}

func copySidecarIfPresent(sourcePath, targetPath string) error {
	sourceSidecar := sourcePath + ".w3a.json"
	if _, err := os.Stat(sourceSidecar); err != nil {
		return nil
	}
	return copyRegularFile(sourceSidecar, targetPath+".w3a.json")
}

func copySidecarIfPresentOverwrite(sourcePath, targetPath string) error {
	sourceSidecar := sourcePath + ".w3a.json"
	if _, err := os.Stat(sourceSidecar); err != nil {
		return nil
	}
	return copyRegularFileOverwrite(sourceSidecar, targetPath+".w3a.json")
}

func uniqueCacheName(dir, baseName string) string {
	baseName = sanitizeFileName(filepath.Base(baseName))
	targetPath := filepath.Join(dir, baseName)
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return baseName
	}

	ext := filepath.Ext(baseName)
	stem := strings.TrimSuffix(baseName, ext)
	for i := 1; i < 10000; i++ {
		candidate := fmt.Sprintf("%s_c%d(copy%d)%s", stem, i, i, ext)
		if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
	return fmt.Sprintf("%s_c%d(copy%d)%s", stem, os.Getpid(), os.Getpid(), ext)
}

func shortHash(hash string) string {
	if len(hash) <= 16 {
		return hash
	}
	return hash[:16]
}

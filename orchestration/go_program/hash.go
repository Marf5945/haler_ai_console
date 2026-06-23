package go_program

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func SourceHash(sourceDir string, manifest Manifest, toolchain Toolchain) (string, []string, error) {
	files, err := goFiles(sourceDir)
	if err != nil {
		return "", nil, err
	}
	h := sha256.New()
	envelope := struct {
		Manifest  Manifest  `json:"manifest"`
		Toolchain Toolchain `json:"toolchain"`
	}{Manifest: manifest, Toolchain: toolchain}
	data, _ := json.Marshal(envelope)
	h.Write(data)
	for _, file := range files {
		rel, _ := filepath.Rel(sourceDir, file)
		h.Write([]byte("\nfile:" + filepath.ToSlash(rel) + "\n"))
		if err := hashFile(h, file); err != nil {
			return "", nil, err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), files, nil
}

func goFiles(sourceDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || strings.HasPrefix(name, ".") {
				if path != sourceDir {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("go_program: list go files: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("go_program: no .go files in source dir")
	}
	return files, nil
}

func hashFile(h io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(h, f)
	return err
}

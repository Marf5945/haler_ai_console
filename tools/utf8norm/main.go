package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"ui_console/builtin"
)

var scanExts = map[string]struct{}{
	".bat":     {},
	".cjs":     {},
	".cmd":     {},
	".command": {},
	".css":     {},
	".d.ts":    {},
	".go":      {},
	".h":       {},
	".html":    {},
	".ini":     {},
	".js":      {},
	".json":    {},
	".jsx":     {},
	".m":       {},
	".md":      {},
	".md5":     {},
	".mjs":     {},
	".ps1":     {},
	".py":      {},
	".sh":      {},
	".svg":     {},
	".toml":    {},
	".ts":      {},
	".tsx":     {},
	".txt":     {},
	".xml":     {},
	".yaml":    {},
	".yml":     {},
}

var scanNames = map[string]struct{}{
	".editorconfig":  {},
	".gitattributes": {},
	".gitignore":     {},
	"disclaimer":     {},
	"license":        {},
	"notice":         {},
	"pre-commit":     {},
	"readme":         {},
	"security":       {},
}

type fileIssue struct {
	Path     string
	Encoding string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("utf8norm", flag.ContinueOnError)
	flags.SetOutput(stderr)

	root := flags.String("root", ".", "root directory to scan")
	write := flags.Bool("write", false, "rewrite detected text files as UTF-8 without BOM")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	issues, errs := scanTree(*root, *write)

	if len(issues) == 0 && len(errs) == 0 {
		fmt.Fprintln(stdout, "All scanned text files are already UTF-8 without BOM.")
		return 0
	}

	if len(issues) > 0 {
		if *write {
			fmt.Fprintln(stdout, "Normalized files to UTF-8 without BOM:")
		} else {
			fmt.Fprintln(stdout, "Files needing UTF-8 normalization:")
		}
		for _, issue := range issues {
			fmt.Fprintf(stdout, "- %s (%s)\n", issue.Path, issue.Encoding)
		}
		if !*write {
			fmt.Fprintln(stdout, "")
			fmt.Fprintln(stdout, "Run `go run ./tools/utf8norm -write` to rewrite them as UTF-8 without BOM.")
		}
	}

	if len(errs) > 0 {
		fmt.Fprintln(stderr, "Errors:")
		for _, err := range errs {
			fmt.Fprintf(stderr, "- %v\n", err)
		}
		return 1
	}

	if *write {
		return 0
	}
	return 1
}

func scanTree(root string, write bool) ([]fileIssue, []error) {
	root = filepath.Clean(root)

	var issues []fileIssue
	var errs []error

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, err)
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			errs = append(errs, relErr)
			return nil
		}

		if d.IsDir() {
			if shouldSkipDir(rel, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !shouldScanFile(rel) {
			return nil
		}

		issue, issueErr := normalizeFile(path, rel, write)
		if issueErr != nil {
			errs = append(errs, issueErr)
			return nil
		}
		if issue != nil {
			issues = append(issues, *issue)
		}
		return nil
	})
	if walkErr != nil {
		errs = append(errs, walkErr)
	}

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Path < issues[j].Path
	})
	sort.Slice(errs, func(i, j int) bool {
		return errs[i].Error() < errs[j].Error()
	})

	return issues, errs
}

func shouldSkipDir(rel, name string) bool {
	if rel == "." {
		return false
	}

	switch name {
	case ".git", "node_modules":
		return true
	}

	rel = filepath.ToSlash(rel)
	return rel == "build/windows"
}

func shouldScanFile(rel string) bool {
	base := strings.ToLower(filepath.Base(rel))
	if _, ok := scanNames[base]; ok {
		return true
	}

	ext := strings.ToLower(filepath.Ext(rel))
	if _, ok := scanExts[ext]; ok {
		return true
	}

	return strings.HasSuffix(base, ".d.ts")
}

func normalizeFile(path, rel string, write bool) (*fileIssue, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: read: %w", filepath.ToSlash(rel), err)
	}

	encoding, normalized, err := detectNormalization(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.ToSlash(rel), err)
	}
	if encoding == "" {
		return nil, nil
	}

	if write {
		info, statErr := os.Stat(path)
		if statErr != nil {
			return nil, fmt.Errorf("%s: stat: %w", filepath.ToSlash(rel), statErr)
		}
		if writeErr := os.WriteFile(path, normalized, info.Mode()); writeErr != nil {
			return nil, fmt.Errorf("%s: write: %w", filepath.ToSlash(rel), writeErr)
		}
	}

	return &fileIssue{
		Path:     filepath.ToSlash(rel),
		Encoding: encoding,
	}, nil
}

func detectNormalization(raw []byte) (string, []byte, error) {
	if hasUTF8BOM(raw) {
		return "utf-8-bom", raw[3:], nil
	}

	if utf8.Valid(raw) {
		return "", nil, nil
	}

	converted, encoding, err := builtin.DetectAndConvert(raw)
	if err != nil {
		return "", nil, err
	}
	if encoding == "unknown" {
		return "", nil, errors.New("encoding could not be determined safely")
	}
	if !utf8.ValidString(converted) {
		return "", nil, errors.New("decoded content is not valid UTF-8")
	}

	return encoding, []byte(converted), nil
}

func hasUTF8BOM(raw []byte) bool {
	return len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF
}

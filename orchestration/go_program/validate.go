package go_program

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func Validate(manifest Manifest, toolchain Toolchain) (ValidationResult, error) {
	hash, files, err := SourceHash(manifest.SourceDir, manifest, toolchain)
	if err != nil {
		return ValidationResult{}, err
	}
	out := ValidationResult{Hash: hash, GoFiles: relFiles(manifest.SourceDir, files)}
	var sawMain bool
	for _, file := range files {
		fileIssues, fileHasMain, err := validateFile(file, manifest)
		if err != nil {
			return ValidationResult{}, err
		}
		sawMain = sawMain || fileHasMain
		out.Issues = append(out.Issues, fileIssues...)
	}
	if !sawMain {
		out.Issues = append(out.Issues, ValidationIssue{
			Kind:   ReviewSandboxRequired,
			Reason: "package main must define func main",
			Review: false,
		})
	}
	out.ReviewRequests = reviewRequests(out.Issues)
	return out, nil
}

func validateFile(path string, manifest Manifest) ([]ValidationIssue, bool, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, false, fmt.Errorf("go_program: parse imports: %w", err)
	}
	full, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return []ValidationIssue{{
			File:   filepath.Base(path),
			Reason: "Go syntax error: " + err.Error(),
			Review: false,
		}}, false, nil
	}
	hasMain := full.Name != nil && full.Name.Name == "main" && hasMainFunc(full.Decls)
	var issues []ValidationIssue
	for _, imp := range file.Imports {
		importPath, _ := jsonUnquote(imp.Path.Value)
		if importPath == "C" || importPath == "unsafe" || importPath == "syscall" || importPath == "plugin" {
			issues = append(issues, ValidationIssue{
				Kind:   ReviewSandboxRequired,
				File:   filepath.Base(path),
				Import: importPath,
				Reason: "import is blocked by the program sandbox",
				Review: true,
			})
			continue
		}
		if isNetworkImport(importPath) && !manifest.Permissions.Network {
			issues = append(issues, ValidationIssue{
				Kind:   ReviewNetworkRequired,
				File:   filepath.Base(path),
				Import: importPath,
				Reason: "network imports require user review and OS-level network sandbox changes",
				Review: true,
			})
			continue
		}
		if importPath == "os/exec" && !manifest.Permissions.ShellSubprocess {
			issues = append(issues, ValidationIssue{
				Kind:   ReviewShellRequired,
				File:   filepath.Base(path),
				Import: importPath,
				Reason: "shell/subprocess use requires user review",
				Review: true,
			})
			continue
		}
		if isStdlib(importPath) {
			continue
		}
		if isVendorAllowed(manifest.SourceDir, importPath, manifest.VendorAllowlist) {
			continue
		}
		issues = append(issues, ValidationIssue{
			Kind:   ReviewUnauthorizedPackage,
			File:   filepath.Base(path),
			Import: importPath,
			Reason: "third-party import is not in the vendor allowlist",
			Review: true,
		})
	}
	return issues, hasMain, nil
}

func hasMainFunc(decls []ast.Decl) bool {
	for _, decl := range decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name != nil && fn.Name.Name == "main" && fn.Recv == nil {
			return true
		}
	}
	return false
}

func jsonUnquote(s string) (string, error) {
	var out string
	err := json.Unmarshal([]byte(s), &out)
	return out, err
}

func isNetworkImport(importPath string) bool {
	return importPath == "net" || strings.HasPrefix(importPath, "net/")
}

func isStdlib(importPath string) bool {
	if strings.Contains(importPath, ".") {
		return false
	}
	pkg, err := build.Default.Import(importPath, "", build.FindOnly)
	return err == nil && pkg.Goroot
}

func isVendorAllowed(sourceDir, importPath string, allowlist []string) bool {
	for _, allowed := range allowlist {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		if importPath == allowed || strings.HasPrefix(importPath, allowed+"/") {
			vendorPath := filepath.Join(sourceDir, "vendor", filepath.FromSlash(importPath))
			if _, err := os.Stat(vendorPath); err == nil {
				return true
			}
		}
	}
	return false
}

func relFiles(root string, files []string) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			out = append(out, file)
			continue
		}
		out = append(out, filepath.ToSlash(rel))
	}
	return out
}

func reviewRequests(issues []ValidationIssue) []ReviewRequest {
	var out []ReviewRequest
	seen := map[string]bool{}
	for _, issue := range issues {
		if !issue.Review {
			continue
		}
		key := string(issue.Kind) + ":" + issue.Import
		if seen[key] {
			continue
		}
		seen[key] = true
		subject := issue.Import
		if subject == "" {
			subject = issue.File
		}
		out = append(out, ReviewRequest{Kind: issue.Kind, Subject: subject, Reason: issue.Reason})
	}
	return out
}

package main

// architecture_imports_test.go — 五層分層約束測試
//
// 用途：搬遷前先跑一次建立基線，搬遷後每次 CI 都跑，確保分層規則不被破壞。
// 執行：go test -run TestArchitectureLayerConstraints -v
//
// 設計原則：
//   - 入口層：可依賴所有層，只做組裝。
//   - 核心層（domain）：不可依賴入口、執行、資料、adapter、Wails、CLI、檔案系統 adapter。
//   - 執行層（orchestration）：可依賴核心與必要資料介面，不可依賴入口層。
//   - 資料層（data）：不可反向依賴核心流程、執行層、入口層。
//   - Adapter 層：可依賴核心定義的 policy/contract，不可被核心直接依賴。
//   - 模糊層（fuzzy）：先列白名單，逐步拆開，不假裝它們已經乾淨。

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Layer definitions
// ---------------------------------------------------------------------------

type Layer string

const (
	LayerEntry         Layer = "entry"
	LayerDomain        Layer = "domain"
	LayerOrchestration Layer = "orchestration"
	LayerData          Layer = "data"
	LayerAdapter       Layer = "adapter"
	LayerFuzzy         Layer = "fuzzy"
)

const modulePath = "ui_console"

// packageLayer maps each local package (import path relative to module) to its layer.
// Post-migration: packages live under domain/, orchestration/, data/, adapter/, shared/.
var packageLayer = map[string]Layer{
	// Entry layer — root main package files (not a sub-package, handled separately)
	// These files: app.go, wails_main.go, main.go, sidecar_runtime.go, etc.

	// Domain layer — pure business rules, no external deps
	"domain/risk":             LayerDomain,
	"domain/source_trust":     LayerDomain,
	"domain/controlled_trust": LayerDomain,
	"domain/llm_context":      LayerDomain,
	"domain/review":           LayerDomain,
	"domain/degraded":         LayerDomain,
	"domain/execution_hook":   LayerDomain,

	// Orchestration layer — coordination & scheduling
	"orchestration/skill_step":         LayerOrchestration,
	"orchestration/skill_flow":         LayerOrchestration,
	"orchestration/cli_manager":        LayerOrchestration,
	"orchestration/dag":                LayerOrchestration,
	"orchestration/step_outline":       LayerOrchestration,
	"orchestration/stop_recovery":      LayerOrchestration,
	"orchestration/delegation":         LayerOrchestration,
	"orchestration/spec_patch_checker": LayerOrchestration,

	// Data layer — persistence & messaging infrastructure
	"data/storage":           LayerData,
	"data/backup":            LayerData,
	"data/memory":            LayerData,
	"data/conversation":      LayerData,
	"data/subexport":         LayerData,
	"data/project_lifecycle": LayerData,

	// Adapter layer — replaceable external integrations
	"adapter/adapter_registry": LayerAdapter,
	"adapter/remote_bridge":    LayerAdapter,
	"adapter/w3a_media":        LayerAdapter,
	"adapter/persona_avatar":   LayerAdapter,
	"adapter/external_link":    LayerAdapter,
	"adapter/visual_learning":  LayerAdapter,
	"adapter/debugtrace":       LayerAdapter,

	// Fuzzy layer — UI state + persistence hybrids, managed by whitelist
	"shared/eventbus":       LayerFuzzy, // imports wails/runtime → UI bridge, not pure data
	"shared/settings":       LayerFuzzy,
	"shared/preference":     LayerFuzzy,
	"shared/browser_pref":   LayerFuzzy,
	"shared/statusrail":     LayerFuzzy,
	"shared/tools":          LayerFuzzy,
	"shared/package_import": LayerFuzzy,
	"shared/health":         LayerFuzzy,
	"shared/taborder":       LayerFuzzy,
	"shared/onboarding":     LayerFuzzy,
	"shared/hookgene":       LayerFuzzy,
}

// ---------------------------------------------------------------------------
// Forbidden import rules (red lines)
// ---------------------------------------------------------------------------

// forbiddenImports defines what each layer must NOT import.
// Key = layer of the importing package.
// Value = list of forbidden import prefixes/packages.
var forbiddenImports = map[Layer][]string{
	LayerDomain: {
		// Domain must not depend on entry, orchestration, data, adapter, Wails, or CLI
		modulePath + "/orchestration/",
		modulePath + "/data/",
		modulePath + "/adapter/",
		modulePath + "/shared/",
		"github.com/wailsapp/wails",
	},
	LayerOrchestration: {
		// Orchestration must not depend on entry layer (there's no sub-package for entry)
		// But it CAN depend on domain + data + adapter (within reason)
		// Red line: must not import wails runtime directly (that's adapter/fuzzy territory)
		"github.com/wailsapp/wails",
	},
	LayerData: {
		// Data must not depend on domain, orchestration, adapter, or entry
		modulePath + "/domain/",
		modulePath + "/orchestration/",
		modulePath + "/adapter/",
	},
	LayerAdapter: {
		// Adapter must not be depended upon by domain (checked via domain rules above).
		// Adapter itself must not import orchestration or entry logic.
		modulePath + "/orchestration/",
	},
	// Fuzzy layer: no strict rules yet — whitelist-managed, violations logged as warnings.
}

// ---------------------------------------------------------------------------
// Explicit whitelist for known acceptable cross-layer imports
// These are documented exceptions, not free passes.
// ---------------------------------------------------------------------------

var whitelist = map[string]map[string]string{
	// package (dir path relative to root) -> import -> reason
	"domain/review": {
		modulePath + "/domain/risk": "review cards use risk.RiskClass enum (same domain layer)",
	},
	"domain/execution_hook": {
		modulePath + "/audit_log": "shared append-only log abstraction (data-layer utility)",
	},
	"domain/controlled_trust": {
		modulePath + "/audit_log": "shared append-only log abstraction (data-layer utility)",
	},
	"orchestration/step_outline": {
		modulePath + "/domain/execution_hook": "orchestration depends on domain (legal: 3→2)",
	},
	"orchestration/cli_manager": {
		modulePath + "/adapter/debugtrace":       "logging utility (should move to shared eventually)",
		modulePath + "/data/conversation":        "orchestration depends on data (legal: 3→4)",
		modulePath + "/orchestration/skill_step": "orchestration internal dependency (same layer)",
	},
	"orchestration/spec_patch_checker": {
		modulePath + "/domain/source_trust":    "orchestration depends on domain (legal: 3→2)",
		modulePath + "/adapter/persona_avatar": "orchestration depends on adapter (legal: 3→5) — checker needs avatar validation",
	},
}

// ---------------------------------------------------------------------------
// Test implementation
// ---------------------------------------------------------------------------

func TestArchitectureLayerConstraints(t *testing.T) {
	projectRoot := findProjectRoot(t)

	// Collect all packages and their imports
	pkgImports := scanLocalImports(t, projectRoot)

	var violations []string
	var warnings []string

	for pkg, imports := range pkgImports {
		layer, ok := packageLayer[pkg]
		if !ok {
			// Root package (main) or unknown — skip constraint check
			continue
		}

		forbidden, hasForbidden := forbiddenImports[layer]
		if !hasForbidden {
			continue
		}

		for _, imp := range imports {
			if !isLocalOrWails(imp) {
				continue
			}

			for _, forbiddenPrefix := range forbidden {
				if strings.HasPrefix(imp, forbiddenPrefix) || imp == forbiddenPrefix {
					// Check whitelist
					if reason, whitelisted := whitelist[pkg][imp]; whitelisted {
						warnings = append(warnings, formatWarning(pkg, layer, imp, reason))
						continue
					}
					violations = append(violations, formatViolation(pkg, layer, imp))
				}
			}
		}
	}

	// Report warnings (whitelisted exceptions)
	if len(warnings) > 0 {
		t.Logf("\n=== WHITELISTED EXCEPTIONS (%d) ===", len(warnings))
		for _, w := range warnings {
			t.Logf("  ⚠️  %s", w)
		}
	}

	// Report and fail on violations
	if len(violations) > 0 {
		t.Errorf("\n=== LAYER VIOLATIONS (%d) ===", len(violations))
		for _, v := range violations {
			t.Errorf("  ❌ %s", v)
		}
		t.Fatalf("\n%d layer constraint violation(s) found. Fix imports or add to whitelist with justification.", len(violations))
	}

	t.Logf("\n✅ All %d packages pass layer constraints (%d whitelisted exceptions).",
		len(pkgImports), len(warnings))
}

// TestDomainRedLines checks the specific red-line rules from the restructure plan:
// domain packages must NOT import app, cli_manager, storage, remote_bridge, frontend, wails/runtime.
func TestDomainRedLines(t *testing.T) {
	projectRoot := findProjectRoot(t)
	pkgImports := scanLocalImports(t, projectRoot)

	domainPackages := []string{
		"domain/risk", "domain/source_trust", "domain/controlled_trust",
		"domain/llm_context", "domain/review", "domain/degraded", "domain/execution_hook",
	}

	redLineImports := []string{
		modulePath + "/orchestration/cli_manager",
		modulePath + "/data/storage",
		modulePath + "/adapter/remote_bridge",
		modulePath + "/shared/settings",
		modulePath + "/shared/eventbus",
		"github.com/wailsapp/wails",
	}

	var violations []string

	for _, pkg := range domainPackages {
		imports, ok := pkgImports[pkg]
		if !ok {
			continue
		}
		for _, imp := range imports {
			for _, redLine := range redLineImports {
				if strings.HasPrefix(imp, redLine) {
					// Check if whitelisted
					if _, wl := whitelist[pkg][imp]; wl {
						continue
					}
					violations = append(violations,
						pkg+" (domain) imports "+imp+" ← RED LINE VIOLATION")
				}
			}
		}
	}

	if len(violations) > 0 {
		for _, v := range violations {
			t.Errorf("❌ %s", v)
		}
		t.Fatalf("%d domain red-line violation(s).", len(violations))
	}
	t.Log("✅ Domain layer: all red lines clear.")
}

// TestDataLayerNoUpwardDeps checks that data layer packages don't import
// domain or orchestration packages.
func TestDataLayerNoUpwardDeps(t *testing.T) {
	projectRoot := findProjectRoot(t)
	pkgImports := scanLocalImports(t, projectRoot)

	dataPackages := []string{
		"data/storage", "data/backup", "data/memory", "data/conversation", "data/subexport", "data/project_lifecycle",
	}

	upwardPackages := []string{
		modulePath + "/domain/",
		modulePath + "/orchestration/",
		modulePath + "/adapter/",
	}

	var violations []string

	for _, pkg := range dataPackages {
		imports, ok := pkgImports[pkg]
		if !ok {
			continue
		}
		for _, imp := range imports {
			for _, upward := range upwardPackages {
				if strings.HasPrefix(imp, upward) {
					violations = append(violations,
						pkg+" (data) imports "+imp+" ← upward dependency!")
				}
			}
		}
	}

	if len(violations) > 0 {
		for _, v := range violations {
			t.Errorf("❌ %s", v)
		}
		t.Fatalf("%d data-layer upward dependency violation(s).", len(violations))
	}
	t.Log("✅ Data layer: no upward dependencies.")
}

// TestAdapterNotImportedByDomain verifies adapters are not reverse-imported by domain.
func TestAdapterNotImportedByDomain(t *testing.T) {
	projectRoot := findProjectRoot(t)
	pkgImports := scanLocalImports(t, projectRoot)

	domainPackages := []string{
		"domain/risk", "domain/source_trust", "domain/controlled_trust",
		"domain/llm_context", "domain/review", "domain/degraded", "domain/execution_hook",
	}

	adapterPackages := []string{
		modulePath + "/adapter/",
	}

	var violations []string

	for _, pkg := range domainPackages {
		imports, ok := pkgImports[pkg]
		if !ok {
			continue
		}
		for _, imp := range imports {
			for _, adapter := range adapterPackages {
				if strings.HasPrefix(imp, adapter) {
					violations = append(violations,
						pkg+" (domain) imports "+imp+" ← adapter reverse dependency!")
				}
			}
		}
	}

	if len(violations) > 0 {
		for _, v := range violations {
			t.Errorf("❌ %s", v)
		}
		t.Fatalf("%d adapter reverse-import violation(s).", len(violations))
	}
	t.Log("✅ Adapters: not reverse-imported by domain.")
}

// TestFuzzyLayerAudit logs all imports from fuzzy-layer packages for visibility.
// Does NOT fail — serves as documentation of current state before cleanup.
func TestFuzzyLayerAudit(t *testing.T) {
	projectRoot := findProjectRoot(t)
	pkgImports := scanLocalImports(t, projectRoot)

	fuzzyPackages := []string{
		"shared/eventbus", "shared/settings", "shared/preference", "shared/browser_pref",
		"shared/statusrail", "shared/tools", "shared/package_import", "shared/health",
		"shared/taborder", "shared/onboarding", "shared/hookgene",
	}

	t.Log("=== FUZZY LAYER AUDIT (whitelist-managed) ===")
	t.Log("These packages have mixed concerns. Documenting dependencies for future cleanup.")
	t.Log("")

	for _, pkg := range fuzzyPackages {
		imports, ok := pkgImports[pkg]
		if !ok {
			t.Logf("  %s: (no local imports found or package missing)", pkg)
			continue
		}

		var localImports []string
		for _, imp := range imports {
			if isLocalOrWails(imp) {
				localImports = append(localImports, imp)
			}
		}

		if len(localImports) == 0 {
			t.Logf("  ✅ %s: no local/wails imports (clean leaf)", pkg)
		} else {
			t.Logf("  📋 %s:", pkg)
			for _, imp := range localImports {
				marker := "    "
				if strings.Contains(imp, "wails") {
					marker = " ⚡ " // wails runtime dependency
				}
				t.Logf("    %s→ %s", marker, imp)
			}
		}
	}
}

// TestEventbusWailsDependency explicitly documents the eventbus → wails violation.
func TestEventbusWailsDependency(t *testing.T) {
	projectRoot := findProjectRoot(t)
	pkgImports := scanLocalImports(t, projectRoot)

	imports, ok := pkgImports["shared/eventbus"]
	if !ok {
		t.Skip("eventbus package not found")
	}

	hasWails := false
	for _, imp := range imports {
		if strings.Contains(imp, "wailsapp/wails") {
			hasWails = true
			break
		}
	}

	if hasWails {
		t.Log("📌 CONFIRMED: eventbus imports github.com/wailsapp/wails/v2/pkg/runtime")
		t.Log("   → This is why eventbus is classified as FUZZY (UI bridge), not pure Data layer.")
		t.Log("   → If eventbus needs to move to Data layer, extract a WailsEmitter interface")
		t.Log("     and inject it at startup, removing the direct wails import.")
	} else {
		t.Log("✅ eventbus no longer imports wails directly — can be reclassified to Data layer.")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// scanLocalImports parses all .go files in sub-packages (including nested layer dirs)
// and returns map[relDirPath][]importPath for local + wails imports only.
// Keys use forward-slash relative paths like "domain/risk", "shared/eventbus".
func scanLocalImports(t *testing.T, root string) map[string][]string {
	t.Helper()
	result := make(map[string][]string)
	fset := token.NewFileSet()

	// Layer directories that contain sub-packages
	layerDirs := []string{"domain", "orchestration", "data", "adapter", "shared"}
	skipDirs := map[string]bool{
		"frontend": true, "build": true, "node_modules": true,
		".gocache": true, ".git": true,
	}

	// Helper: scan a single directory for Go imports
	scanDir := func(dirPath, relPath string) {
		pkgs, err := parser.ParseDir(fset, dirPath, func(fi os.FileInfo) bool {
			return strings.HasSuffix(fi.Name(), ".go")
		}, parser.ImportsOnly)
		if err != nil {
			return
		}
		if len(pkgs) == 0 {
			return
		}

		var imports []string
		seen := make(map[string]bool)

		for _, pkg := range pkgs {
			for _, file := range pkg.Files {
				for _, imp := range file.Imports {
					path := strings.Trim(imp.Path.Value, `"`)
					if seen[path] {
						continue
					}
					seen[path] = true
					if isLocalOrWails(path) {
						imports = append(imports, path)
					}
				}
			}
		}

		result[relPath] = imports
	}

	// Scan layer directories (domain/risk, orchestration/dag, etc.)
	for _, layerDir := range layerDirs {
		layerPath := filepath.Join(root, layerDir)
		entries, err := os.ReadDir(layerPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			subDir := entry.Name()
			if skipDirs[subDir] || strings.HasPrefix(subDir, ".") {
				continue
			}
			relPath := layerDir + "/" + subDir
			scanDir(filepath.Join(layerPath, subDir), relPath)
		}
	}

	// Also scan any remaining top-level package dirs (e.g. audit_log if it exists)
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("cannot read project root: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()
		if skipDirs[dirName] || strings.HasPrefix(dirName, ".") {
			continue
		}
		// Skip layer dirs (already scanned above) and runtime data
		isLayer := false
		for _, ld := range layerDirs {
			if dirName == ld {
				isLayer = true
				break
			}
		}
		if isLayer {
			continue
		}
		scanDir(filepath.Join(root, dirName), dirName)
	}

	return result
}

func isLocalOrWails(importPath string) bool {
	return strings.HasPrefix(importPath, modulePath+"/") ||
		strings.HasPrefix(importPath, "github.com/wailsapp/wails")
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	// Look for go.mod in current directory or parents
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("cannot find project root (go.mod)")
		}
		dir = parent
	}
}

func formatViolation(pkg string, layer Layer, imp string) string {
	targetPkg := strings.TrimPrefix(imp, modulePath+"/")
	targetLayer, ok := packageLayer[targetPkg]
	if !ok {
		if strings.Contains(imp, "wails") {
			return pkg + " (" + string(layer) + ") → " + imp + " [FORBIDDEN: wails runtime in this layer]"
		}
		return pkg + " (" + string(layer) + ") → " + imp + " [FORBIDDEN: unknown package]"
	}
	return pkg + " (" + string(layer) + ") → " + targetPkg + " (" + string(targetLayer) + ") [LAYER VIOLATION]"
}

func formatWarning(pkg string, layer Layer, imp, reason string) string {
	return pkg + " (" + string(layer) + ") → " + imp + " [WHITELISTED: " + reason + "]"
}

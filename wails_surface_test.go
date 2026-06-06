// wails_surface_test.go — Wails 暴露邊界守門員
//
// 第一刀（2026-05-24，SEC-W05 minimal cut）後鎖住三件事，防止回潮：
//
//   T-1  TestBindListOnlyApp
//        wails_main.go 的 Bind 物件只能有 1 個（必須是 app）。
//        防止未來偷偷多綁 devTools / 二次 service，瞬間擴大攻擊面。
//
//   T-2  TestWailsExposedMethodSetMatchesCodegen
//        Go 跨檔的 exported App method 集合，必須與
//        frontend/wailsjs/go/main/App.d.ts 的 export function 集合相同。
//        - Go 多了：忘記跑 `wails generate module`。
//        - codegen 多了：舊 binding 殘留沒清。
//        用「集合相等」而非「count 相等」，可抓「同一 PR 加 1 個刪 1 個」。
//
//   T-3  TestForbiddenInternalMethodsNotExposed
//        指定為 internal-only / orphan 的方法名，永遠不得出現在 App.d.ts
//        或 App.js。第一刀已刪除的 5 個 method 直接列在 forbiddenInternalMethods。
//        未來認定為 internal-only 的新方法，加進此清單。
//
// 執行：
//   go test -run TestBindListOnlyApp -v
//   go test -run TestWailsExposedMethodSetMatchesCodegen -v
//   go test -run TestForbiddenInternalMethodsNotExposed -v
//
// 注意：findProjectRoot helper 已由 architecture_imports_test.go 提供（同 package main）。
package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Forbidden internal method names — 永遠不該出現在 codegen
// ---------------------------------------------------------------------------
//
// 加入規則：
//   - 該方法的本意是「後端 pipeline 內部呼叫」/「自動記錄」/「debug-only」。
//   - 且目前 frontend 沒有任何真實 caller（grep frontend/src 確認）。
//   - 加進清單後，請同時把該 method 從 Go side 移除或改為 lowercase / 搬 service。
var forbiddenInternalMethods = []string{
	// 第一刀（2026-05-24）刪除的 5 個 orphan：
	"EmitDagEvent",
	"RecordLearningEvent",
	"UpdateReadinessGateState",
	"RecordReadinessTrace",
	"ListReadinessTraces",
}

// ---------------------------------------------------------------------------
// T-1  Bind 物件數量
// ---------------------------------------------------------------------------

func TestBindListOnlyApp(t *testing.T) {
	root := findProjectRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "wails_main.go"))
	if err != nil {
		t.Fatalf("read wails_main.go: %v", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "wails_main.go", src, 0)
	if err != nil {
		t.Fatalf("parse wails_main.go: %v", err)
	}

	var bindElts []ast.Expr
	ast.Inspect(f, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Bind" {
			return true
		}
		// 期望結構：Bind: []interface{}{app}
		cl, ok := kv.Value.(*ast.CompositeLit)
		if !ok {
			return true
		}
		bindElts = cl.Elts
		return false
	})

	if bindElts == nil {
		t.Fatalf("找不到 wails_main.go 中的 Bind: []interface{}{...} 結構")
	}
	if len(bindElts) != 1 {
		t.Fatalf("Bind 清單長度應為 1（只有 app），實際為 %d；新增 Bind 物件會擴大 Wails 攻擊面，請改放成 internal service", len(bindElts))
	}
	id, ok := bindElts[0].(*ast.Ident)
	if !ok || id.Name != "app" {
		t.Fatalf("Bind 唯一元素必須是 ident 'app'，實際為 %T(%v)", bindElts[0], bindElts[0])
	}
}

// ---------------------------------------------------------------------------
// T-2  Go exported App method set ≡ App.d.ts export function set
// ---------------------------------------------------------------------------

var exportedAppMethodRe = regexp.MustCompile(`^func \(a \*App\) ([A-Z]\w*)\(`)
var dtsExportFuncRe = regexp.MustCompile(`^export function ([A-Z]\w*)\(`)

func TestWailsExposedMethodSetMatchesCodegen(t *testing.T) {
	root := findProjectRoot(t)

	goSet := collectGoExportedAppMethods(t, root)
	dtsSet := collectDTSExportFunctions(t, root)

	goOnly := setDiff(goSet, dtsSet)
	dtsOnly := setDiff(dtsSet, goSet)

	if len(goOnly) == 0 && len(dtsOnly) == 0 {
		return // 完全對齊
	}

	var msg strings.Builder
	msg.WriteString("Go exported App methods 與 frontend/wailsjs/go/main/App.d.ts 不一致：\n")
	if len(goOnly) > 0 {
		sort.Strings(goOnly)
		msg.WriteString("\n  Go side 有但 App.d.ts 沒有（請執行 `wails generate module`）:\n")
		for _, name := range goOnly {
			msg.WriteString("    + " + name + "\n")
		}
	}
	if len(dtsOnly) > 0 {
		sort.Strings(dtsOnly)
		msg.WriteString("\n  App.d.ts 有但 Go side 沒有（舊 binding 殘留，請重 generate）:\n")
		for _, name := range dtsOnly {
			msg.WriteString("    - " + name + "\n")
		}
	}
	t.Fatal(msg.String())
}

// ---------------------------------------------------------------------------
// T-3  forbidden internal methods 不得出現在 App.d.ts / App.js
// ---------------------------------------------------------------------------

func TestForbiddenInternalMethodsNotExposed(t *testing.T) {
	root := findProjectRoot(t)

	for _, file := range []string{
		filepath.Join("frontend", "wailsjs", "go", "main", "App.d.ts"),
		filepath.Join("frontend", "wailsjs", "go", "main", "App.js"),
	} {
		data, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		content := string(data)
		for _, name := range forbiddenInternalMethods {
			// 同時防止「export function NAME(」與「window[...][NAME]」兩種露出形式。
			if strings.Contains(content, "export function "+name+"(") ||
				strings.Contains(content, "'"+name+"'") {
				t.Errorf("%s 仍包含被禁的 internal-only method %q；請從 Go 端刪除/lowercase 後重 generate", file, name)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func collectGoExportedAppMethods(t *testing.T, root string) map[string]struct{} {
	t.Helper()
	out := map[string]struct{}{}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read root: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			if m := exportedAppMethodRe.FindStringSubmatch(line); m != nil {
				out[m[1]] = struct{}{}
			}
		}
	}
	return out
}

func collectDTSExportFunctions(t *testing.T, root string) map[string]struct{} {
	t.Helper()
	out := map[string]struct{}{}
	data, err := os.ReadFile(filepath.Join(root, "frontend", "wailsjs", "go", "main", "App.d.ts"))
	if err != nil {
		t.Fatalf("read App.d.ts: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if m := dtsExportFuncRe.FindStringSubmatch(line); m != nil {
			out[m[1]] = struct{}{}
		}
	}
	return out
}

func setDiff(a, b map[string]struct{}) []string {
	var out []string
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k)
		}
	}
	return out
}

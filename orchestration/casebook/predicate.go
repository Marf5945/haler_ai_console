// predicate.go — 可機器驗的完成條件（分層判定的第一層）。
//
// planner 在 plan 階段為每個節點吐一個 predicate 字串；節點終局時 Go 直接驗，
// 驗得了就給出 pass/fail，省下一次模型呼叫，也讓「預期」可重現、可回歸。
// 驗不了（Ok=false）的交給 tagger 的模型自評層。
//
// 文法（單行，刻意極簡，全 stdlib）：
//
//	file_exists:<path>        路徑存在（檔或目錄）
//	file_missing:<path>       路徑不存在
//	count>=<n> / count==<n>   把 Actual 解析成整數比較（>=、==、<=、>、<）
//	contains:<substr>        Actual 含子字串
//	not_contains:<substr>    Actual 不含子字串
//	regex:<pattern>          Actual 符合正則
//	nonempty                 Actual 去空白後非空
//
// 留白 / 不認得的文法 → Ok=false（退模型層，不是 fail）。
package casebook

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"ui_console/data/storage"
)

// PredicateInput 是驗 predicate 需要的執行結果。
type PredicateInput struct {
	Actual      string // sanitized 觀察摘要 / 節點 ResultSummary
	ProjectRoot string // file_exists 等路徑類的根（相對路徑以此為基準）
	HadError    bool   // 執行層是否報錯（報錯時多數 predicate 直接判 fail）
}

// EvalPredicate 驗 predicate。
// ok=false 表示「這條 predicate 無法機器判定」，呼叫端應退回模型自評。
func EvalPredicate(predicate string, in PredicateInput) (pass bool, ok bool) {
	p := strings.TrimSpace(predicate)
	if p == "" {
		return false, false // 沒給條件 → 交模型層
	}

	// count 比較：count>=3 / count==1 / count<5 ...
	if strings.HasPrefix(p, "count") {
		return evalCount(p, in.Actual)
	}

	kind, arg, hasArg := splitPredicate(p)
	switch kind {
	case "nonempty":
		return strings.TrimSpace(in.Actual) != "", true
	case "file_exists":
		full, safe := resolveInProject(in.ProjectRoot, arg, hasArg)
		if !safe {
			return false, false // 缺參數 / 絕對路徑 / 逃逸 projectRoot → 不機器判定
		}
		_, err := os.Stat(full)
		return err == nil, true
	case "file_missing":
		full, safe := resolveInProject(in.ProjectRoot, arg, hasArg)
		if !safe {
			return false, false
		}
		_, err := os.Stat(full)
		return os.IsNotExist(err), true
	case "contains":
		if !hasArg {
			return false, false
		}
		return strings.Contains(in.Actual, arg), true
	case "not_contains":
		if !hasArg {
			return false, false
		}
		return !strings.Contains(in.Actual, arg), true
	case "regex":
		if !hasArg {
			return false, false
		}
		re, err := regexp.Compile(arg)
		if err != nil {
			return false, false // 壞 pattern → 交模型層
		}
		return re.MatchString(in.Actual), true
	default:
		return false, false // 不認得 → 交模型層
	}
}

// splitPredicate 切 "kind:arg"；無冒號則 kind=整串、hasArg=false。
func splitPredicate(p string) (kind, arg string, hasArg bool) {
	if i := strings.IndexByte(p, ':'); i >= 0 {
		return strings.TrimSpace(p[:i]), p[i+1:], true
	}
	return p, "", false
}

var countRe = regexp.MustCompile(`^count\s*(>=|<=|==|>|<)\s*(-?\d+)$`)

// evalCount 把 Actual 中第一個整數抓出來與門檻比較。
func evalCount(p, actual string) (pass bool, ok bool) {
	m := countRe.FindStringSubmatch(strings.ReplaceAll(p, " ", ""))
	if m == nil {
		return false, false
	}
	want, _ := strconv.Atoi(m[2])
	got, found := firstInt(actual)
	if !found {
		return false, false // Actual 裡沒有數字 → 無法機器判定
	}
	switch m[1] {
	case ">=":
		return got >= want, true
	case "<=":
		return got <= want, true
	case "==":
		return got == want, true
	case ">":
		return got > want, true
	case "<":
		return got < want, true
	}
	return false, false
}

func firstInt(s string) (int, bool) {
	m := regexp.MustCompile(`-?\d+`).FindString(s)
	if m == "" {
		return 0, false
	}
	n, err := strconv.Atoi(m)
	return n, err == nil
}

// resolveInProject 把 predicate 的路徑參數安全地錨定在 projectRoot 內。
// predicate 可能來自 plan/model 階段，故嚴禁讓它探測 projectRoot 以外的檔案：
//   - 缺參數 / projectRoot 為空 → 不機器判定（safe=false）
//   - 絕對路徑 → 拒絕（不重新錨定，直接 safe=false）
//   - filepath.Join 清理後再用 storage.ValidatePath 確認仍在 projectRoot 邊界內
//     （含 ".."、symlink 逃逸檢查），任何越界 → safe=false。
func resolveInProject(projectRoot, p string, hasArg bool) (full string, safe bool) {
	if !hasArg || strings.TrimSpace(p) == "" || projectRoot == "" || filepath.IsAbs(p) {
		return "", false
	}
	full = filepath.Join(projectRoot, p)
	if err := storage.ValidatePath(full, projectRoot); err != nil {
		return "", false
	}
	return full, true
}

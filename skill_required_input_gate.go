package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"ui_console/orchestration/go_program"
)

// 通用必填輸入閘（data-driven）。
//
// 目的：執行「流程」型 skill 前，先比對「該 skill 自己宣告的必填輸入」與
// 「使用者訊息 + 已載入檔案」，缺哪個就反問哪個——而不是進 need_confirm、
// 等到程式執行才回死的錯誤字串。
//
// 設計重點：所有判斷都從 manifest 的 InputSchema.Required 讀取，不為任何單一
// skill 寫死。新增第二、第三個 skill 時，只要在它的 manifest 宣告必填欄位即
// 自動套用本閘，無需改動程式。

// genericRequiredInputPlaceholders 是「未客製化過的預設佔位欄位」。
// go-program 種子 manifest 預設 Required=["input"]，這類佔位不代表真正缺資料，
// 若拿來反問會把沒設定 schema 的舊 skill 無謂攔下，因此一律放行。
var genericRequiredInputPlaceholders = map[string]bool{
	"input":  true,
	"inputs": true,
	"輸入":     true,
	"data":   true,
	"資料":     true,
	"file":   true,
	"files":  true,
	"檔案":     true,
}

// missingRequiredInputs 回傳「使用者訊息與已載入檔名都沒提供」的必填欄位，
// 維持 required 原始順序、去除重複與佔位欄位。
//
// 判定規則（通用且保守）：欄位關鍵字出現在 userText 或任一已載入檔名中 →
// 視為已提供；兩者皆無 → 視為缺。寧可多問也不要漏問（缺資料時靜默失敗最糟）。
func missingRequiredInputs(required []string, userText string, refNames []string) []string {
	hay := strings.ToLower(userText)
	for _, n := range refNames {
		hay += "\n" + strings.ToLower(n)
	}
	var missing []string
	seen := map[string]bool{}
	for _, raw := range required {
		field := strings.TrimSpace(raw)
		if field == "" {
			continue
		}
		key := strings.ToLower(field)
		if seen[key] {
			continue
		}
		seen[key] = true
		if genericRequiredInputPlaceholders[key] {
			continue
		}
		if strings.Contains(hay, key) {
			continue
		}
		missing = append(missing, field)
	}
	return missing
}

// loadGoProgramManifestFromDisk 讀取已歸檔 skill 保存的 go_program.Manifest
// （與 goProgramSkillDescriptionFromDisk 同一份檔），用來取得 InputSchema 等
// 結構化資訊。找不到或解析失敗回 (zero,false)。
func loadGoProgramManifestFromDisk(skillID string) (go_program.Manifest, bool) {
	path := filepath.Join(appDataRoot(), "data", "skills", skillID, "programs", "program_manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return go_program.Manifest{}, false
	}
	var m go_program.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return go_program.Manifest{}, false
	}
	return m, true
}

// skillRequiredInputs 解析「流程」目標 skill 的必填輸入清單。
// 來源是該 skill 保存的 go_program.Manifest.InputSchema.Required。
// 找不到對應的 go-program manifest 時回 (nil,false)，呼叫端據此不啟用必填閘，
// 維持對「非 go-program skill」的向後相容（不影響既有行為）。
func (a *App) skillRequiredInputs(target string) ([]string, bool) {
	if a == nil || a.skillArchive == nil {
		return nil, false
	}
	want := normalizeGoProgramLookup(target)
	if want == "" {
		return nil, false
	}
	manifests, err := a.skillArchive.ListArchived()
	if err != nil {
		return nil, false
	}
	for _, m := range manifests {
		if normalizeGoProgramLookup(m.DisplayName) != want && normalizeGoProgramLookup(m.SkillID) != want {
			continue
		}
		pm, ok := loadGoProgramManifestFromDisk(m.SkillID)
		if !ok {
			return nil, false
		}
		return pm.InputSchema.Required, true
	}
	return nil, false
}

// skillMissingInputQuestion 組「缺必填輸入」的反問文字（通用，不點名特定欄位語意）。
func skillMissingInputQuestion(target string, missing []string) string {
	return "要用 skill「" + target + "」還缺必要輸入：" + strings.Join(missing, "、") +
		"。請補上後再說一次（若缺的是檔案，請先載入對應檔再試）。"
}

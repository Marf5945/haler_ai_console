package main

// code_artifact_repair.go — 編譯失敗自動回修迴圈（3.1.x）。
//
// 流程：編譯 → 失敗 → 把「原始碼＋編譯錯誤」丟回模型要完整修正版 →
// 覆寫資料區檔案（meta／SHA／標示同步更新）→ 再編譯。最多 2 輪，
// 每輪結果都寫進回報，修不好就老實把剩餘錯誤端出來。
//
// 模型呼叫走既有 callRawModel（API／本地／CLI 三種 adapter 通吃），
// 純標準庫，不引入任何新依賴。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ui_console/adapter/debugtrace"
)

const codeArtifactRepairMaxAttempts = 2
const codeArtifactRepairMaxTokens = 2500

// codeRepairModelCall — 模型呼叫縫合點（測試可替換成 stub，不碰真模型）。
var codeRepairModelCall = func(a *App, adapterID, sessionID, prompt, traceID string) (string, error) {
	return a.callRawModelCapped(adapterID, "code-repair:"+sessionID, prompt, traceID, codeArtifactRepairMaxTokens)
}

func codeArtifactCompileToolDetail(fileName string) string {
	return "後端正在執行 " + codeArtifactCompileToolName(fileName) + "。"
}

func codeArtifactRecompileToolDetail(fileName string) string {
	return "後端正在再次執行 " + codeArtifactCompileToolName(fileName) + "。"
}

func codeArtifactCompileToolName(fileName string) string {
	meta, ok := codeArtifactMetaByFileName(fileName)
	if !ok {
		return "編譯工具"
	}
	switch meta.Language {
	case "go":
		return "go build"
	case "rust":
		return "rustc"
	case "cpp":
		return "c++"
	case "c":
		return "cc"
	default:
		return "編譯工具"
	}
}

// compileCodeArtifactWithAutoRepair — 編譯；失敗自動叫模型修再編。
// 回傳給聊天室的完整回報文字。
func (a *App) compileCodeArtifactWithAutoRepair(adapterID, sessionID, fileName, traceID string) string {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	a.emitCodeArtifactActivity(traceID, sessionID, fileName, "compile_start", "開始編譯資料區程式", codeArtifactCompileToolDetail(fileName), "running", 0)
	result, err := a.CompileCodeArtifact(fileName)
	if err != nil {
		a.emitCodeArtifactActivity(traceID, sessionID, fileName, "compile_error", "編譯程序沒有啟動", err.Error(), "failed", 0)
		return "編譯沒跑起來：" + err.Error()
	}
	if result.Status != "failed" {
		activityStatus := "warning"
		if result.Status == "success" {
			activityStatus = "success"
		}
		a.emitCodeArtifactActivity(traceID, sessionID, fileName, "compile_done", "編譯流程完成", result.Message, activityStatus, 0)
		return result.Message
	}

	a.emitCodeArtifactActivity(traceID, sessionID, fileName, "compile_failed", "編譯失敗，準備自動修復", firstNonEmpty(result.Output, result.Message), "failed", 0)
	report := []string{fmt.Sprintf("第 1 次編譯失敗：\n%s", firstNonEmpty(result.Output, result.Message))}
	for attempt := 1; attempt <= codeArtifactRepairMaxAttempts; attempt++ {
		meta, ok := codeArtifactMetaByFileName(fileName)
		if !ok {
			a.emitCodeArtifactActivity(traceID, sessionID, fileName, "repair_meta_missing", "找不到程式中繼資料", "停止自動修復。", "failed", attempt)
			report = append(report, "中繼資料不見了，停止自動修復。")
			break
		}
		srcPath := filepath.Join(codeArtifactDir(), fileName)
		src, readErr := os.ReadFile(srcPath)
		if readErr != nil {
			a.emitCodeArtifactActivity(traceID, sessionID, fileName, "repair_source_missing", "讀取原始碼失敗", readErr.Error(), "failed", attempt)
			report = append(report, "原始碼讀不到，停止自動修復："+readErr.Error())
			break
		}

		prompt := buildCodeRepairPrompt(meta, string(src), result.Output)
		a.emitCodeArtifactActivity(traceID, sessionID, fileName, "repair_attempt", "送交模型修復", "把完整原始碼與編譯錯誤交給模型，只要求修錯並回完整檔案。", "running", attempt)
		debugtrace.Record("go.codeArtifact.repair.attempt", traceID, map[string]interface{}{
			"file_name": fileName,
			"attempt":   attempt,
		})
		out, callErr := codeRepairModelCall(a, adapterID, sessionID, prompt, traceID)
		if callErr != nil {
			a.emitCodeArtifactActivity(traceID, sessionID, fileName, "repair_model_error", "模型修復呼叫失敗", callErr.Error(), "failed", attempt)
			report = append(report, fmt.Sprintf("第 %d 輪修復叫不動模型：%s", attempt, callErr.Error()))
			break
		}
		a.emitCodeArtifactActivity(traceID, sessionID, fileName, "repair_model_returned", "模型已回傳修正版", fmt.Sprintf("收到 %d 個字元，正在抽取完整程式碼。", len([]rune(out))), "running", attempt)
		_, fixedRaw, _, hasCode := extractLargestFencedCode(out)
		if !hasCode || len(strings.Split(fixedRaw, "\n")) < 3 {
			a.emitCodeArtifactActivity(traceID, sessionID, fileName, "repair_no_code", "模型沒有回完整程式碼", "停止自動修復。", "failed", attempt)
			report = append(report, fmt.Sprintf("第 %d 輪模型沒回完整程式碼，停止自動修復。", attempt))
			break
		}
		clean, marks := parseCodeArtifactMarks(fixedRaw)
		if writeErr := os.WriteFile(srcPath, []byte(clean), 0o600); writeErr != nil {
			a.emitCodeArtifactActivity(traceID, sessionID, fileName, "repair_write_failed", "修正版寫回失敗", writeErr.Error(), "failed", attempt)
			report = append(report, "修正版寫不回資料區："+writeErr.Error())
			break
		}
		a.updateCodeArtifactAfterRepair(fileName, clean, marks)
		a.emitCodeArtifactActivity(traceID, sessionID, fileName, "repair_written", "修正版已寫回資料區", fmt.Sprintf("已同步 SHA 與 %d 個修改標示，準備重編。", len(marks)), "running", attempt)

		var compErr error
		a.emitCodeArtifactActivity(traceID, sessionID, fileName, "recompile_start", "開始重編修正版", codeArtifactRecompileToolDetail(fileName), "running", attempt)
		result, compErr = a.CompileCodeArtifact(fileName)
		if compErr != nil {
			a.emitCodeArtifactActivity(traceID, sessionID, fileName, "recompile_error", "重編譯程序沒有啟動", compErr.Error(), "failed", attempt)
			report = append(report, "重編譯沒跑起來："+compErr.Error())
			break
		}
		if result.Status == "success" {
			a.emitCodeArtifactActivity(traceID, sessionID, fileName, "recompile_success", "修復後編譯通過", result.Message, "success", attempt)
			markNote := ""
			if len(marks) > 0 {
				markNote = fmt.Sprintf("修改處已用標示記下（%d 處），展開就看得到。", len(marks))
			}
			report = append(report, fmt.Sprintf("第 %d 輪修復後編譯通過。%s", attempt, markNote))
			report = append(report, result.Message)
			return strings.Join(report, "\n")
		}
		if result.Status != "failed" {
			// tool_missing / unsupported：不是程式碼的錯，修也沒用
			activityStatus := "warning"
			if result.Status == "success" {
				activityStatus = "success"
			}
			a.emitCodeArtifactActivity(traceID, sessionID, fileName, "recompile_done", "重編譯流程完成", result.Message, activityStatus, attempt)
			report = append(report, result.Message)
			return strings.Join(report, "\n")
		}
		a.emitCodeArtifactActivity(traceID, sessionID, fileName, "recompile_failed", "修復後仍編譯失敗", firstNonEmpty(result.Output, result.Message), "failed", attempt)
		report = append(report, fmt.Sprintf("第 %d 輪修復後仍編譯失敗：\n%s", attempt, firstNonEmpty(result.Output, result.Message)))
	}
	a.emitCodeArtifactActivity(traceID, sessionID, fileName, "repair_exhausted", "自動修復輪次用完", "保留最後錯誤供使用者檢查。", "failed", codeArtifactRepairMaxAttempts)
	report = append(report, fmt.Sprintf("自動修復 %d 輪用罄。剩餘錯誤在上面——展開程式碼把該段貼回來，或換個說法讓我重寫。", codeArtifactRepairMaxAttempts))
	return strings.Join(report, "\n")
}

// buildCodeRepairPrompt — 修復提示詞：原始碼＋編譯錯誤，要求回完整可編譯版本。
func buildCodeRepairPrompt(meta CodeArtifactMeta, source, compileErrors string) string {
	if runes := []rune(compileErrors); len(runes) > 1500 {
		compileErrors = string(runes[:1500]) + "…"
	}
	return strings.Join([]string{
		fmt.Sprintf("你在修一支 %s 程式「%s」，它編譯失敗了。", meta.LanguageLabel, meta.Kind),
		"規則：",
		"1. 只修錯誤，不重新設計，保留原本的功能與結構。",
		"2. 回覆只放一個 fenced code block（```" + meta.Language + " 換行 完整程式碼```），前後不要長篇解釋。",
		"3. 必須是完整檔案內容，可直接覆蓋原檔編譯。",
		"4. 把你動過的區段用《標2》…《/標2》包起來（本次修改標示），方便使用者核對。",
		"",
		"編譯錯誤：",
		compileErrors,
		"",
		"目前的完整原始碼：",
		"```" + meta.Language,
		source,
		"```",
	}, "\n")
}

// updateCodeArtifactAfterRepair — 修復覆寫後同步 meta：SHA／標示更新、編譯狀態重置。
func (a *App) updateCodeArtifactAfterRepair(fileName, clean string, marks []CodeArtifactMark) {
	codeArtifactMu.Lock()
	defer codeArtifactMu.Unlock()
	index := loadCodeArtifactIndex()
	meta, ok := index[fileName]
	if !ok {
		return
	}
	meta.ContentSHA = codeArtifactSHA([]byte(clean))
	meta.Marks = marks
	meta.CompileStatus = ""
	meta.CompileDetail = ""
	meta.BinaryPath = ""
	index[fileName] = meta
	_ = saveCodeArtifactIndex(index)
	a.emitCodeArtifactUpdated(meta)
}

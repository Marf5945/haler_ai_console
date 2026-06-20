package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"ui_console/adapter/debugtrace"
)

// 本地模型語系驗收矩陣（後端管線層，閘控整合測試）。
//
// 為什麼存在：Computer Use 模擬鍵盤會掉 CJK 字，原生日文/韓文題目打不進 UI。
// 但 userText 只是字串參數——CJK 寫死在 .go 不會掉字。此測試繞過鍵盤，直接驅動
// SendAPIMessage 跑完整矩陣，並用 cjklang.go 的判定機器化攔截語系錯誤
// （例如「韓文角色卻吐日文片假名選項」）。
//
// 預設 SKIP：需要本機 Ollama。開啟方式：
//   UI_CONSOLE_LANG_MATRIX=1 go test ./... -run TestLocalModelLanguageMatrix -v
// 可選環境變數：
//   LANG_MATRIX_MODEL     預設 qwen2.5:14b
//   LANG_MATRIX_ENDPOINT  預設 http://localhost:11434/v1
//
// 判定分層：
//   輸出 / 程式·流程  → presence（自由文字必須含該語系特徵 script）
//   選項              → leak（label 不得混入矛盾 script；核心攔截）
//   網路 / 本機搜尋    → infra-lenient（會被 Go 端直連處理或依賴外部 provider，只記錄不硬斷言）

type langMatrixCase struct {
	label       string   // 套用的角色語言標籤（給 ApplyStyleDiff）
	code        string   // responseLanguage* 代碼（給判定）
	category    string   // 題型
	text        string   // CJK 題目（寫死，不經鍵盤）
	wantActions []string // 期望第一行動作（localAssembled.action）
	check       string   // "presence" | "leak" | ""(infra)
}

func languageMatrixCases() []langMatrixCase {
	ja := func(cat, text string, acts []string, chk string) langMatrixCase {
		return langMatrixCase{"日本語", responseLanguageJA, cat, text, acts, chk}
	}
	ko := func(cat, text string, acts []string, chk string) langMatrixCase {
		return langMatrixCase{"한국어", responseLanguageKO, cat, text, acts, chk}
	}
	zh := func(cat, text string, acts []string, chk string) langMatrixCase {
		return langMatrixCase{"中文", responseLanguageZH, cat, text, acts, chk}
	}
	en := func(cat, text string, acts []string, chk string) langMatrixCase {
		return langMatrixCase{"English", responseLanguageEN, cat, text, acts, chk}
	}
	return []langMatrixCase{
		// 日文 5 題
		ja("一般對答", "こんにちは、今日も手伝ってくれますか？", []string{"輸出", "聊天"}, "presence"),
		ja("網路搜尋", "ネットで夜行性の動物を調べて", []string{"網路", "搜尋"}, ""),
		ja("非顏色選項", "6月21日、6月22日、6月23日から一つ選んで", []string{"選項"}, "leak"),
		ja("本機搜尋", "ローカルで動物のレポートを探して", []string{"搜尋", "本機搜尋"}, ""),
		ja("程式流程", "CSVをグラフに変換するプログラムを作って", []string{"程式", "流程"}, "presence"),
		// 韓文 5 題
		ko("一般對答", "안녕하세요, 오늘도 도와줄 수 있나요?", []string{"輸出", "聊天"}, "presence"),
		ko("網路搜尋", "인터넷에서 야행성 동물을 검색해줘", []string{"網路", "搜尋"}, ""),
		ko("非顏色選項", "6월 21일, 6월 22일, 6월 23일 중에서 하나 골라줘", []string{"選項"}, "leak"),
		ko("本機搜尋", "로컬에서 동물 보고서를 찾아줘", []string{"搜尋", "本機搜尋"}, ""),
		ko("程式流程", "CSV를 그래프로 변환하는 프로그램을 만들어줘", []string{"程式", "流程"}, "presence"),
		// 英文 5 題
		en("一般對答", "Hello, can you help me today?", []string{"輸出", "聊天"}, "presence"),
		en("網路搜尋", "Search the web for nocturnal animal species", []string{"網路", "搜尋"}, ""),
		en("非顏色選項", "Choose one date: June 21, June 22, June 23", []string{"選項"}, "leak"),
		en("本機搜尋", "Find local animal reports", []string{"搜尋", "本機搜尋"}, ""),
		en("程式流程", "Make a program that charts animal counts from CSV", []string{"程式", "流程"}, "presence"),
		// 中文回歸 1 題
		zh("一般對答", "你好，今天可以幫我嗎？", []string{"輸出", "聊天"}, "presence"),
	}
}

func TestLocalModelLanguageMatrix(t *testing.T) {
	if os.Getenv("UI_CONSOLE_LANG_MATRIX") == "" {
		t.Skip("設定 UI_CONSOLE_LANG_MATRIX=1 並啟動本機 Ollama 後才跑此整合測試")
	}
	model := envOr("LANG_MATRIX_MODEL", "qwen2.5:14b")
	endpoint := envOr("LANG_MATRIX_ENDPOINT", "http://localhost:11434/v1")
	adapterID := "local-ollama-" + sanitizeAdapterID(model)

	app := NewApp()
	if app.adapterRegistry == nil {
		t.Fatal("adapterRegistry 為 nil，NewApp 初始化失敗")
	}
	if err := app.adapterRegistry.RegisterLocal(adapterID, "Ollama - "+model, "O", endpoint, model); err != nil {
		t.Fatalf("RegisterLocal: %v", err)
	}

	// 連線預檢：Ollama 不在就 SKIP，避免把 infra 缺失誤判成語系失敗。
	warmTrace := "chat-langmatrix-warmup"
	if _, err := app.SendAPIMessage(adapterID, "langmatrix", "ping", warmTrace); err != nil {
		if isConnRefused(err) {
			t.Skipf("Ollama 無法連線（%s）：%v", endpoint, err)
		}
		t.Fatalf("warmup SendAPIMessage: %v", err)
	}

	for i, c := range languageMatrixCases() {
		c := c
		name := fmt.Sprintf("%s/%s", c.code, c.category)
		t.Run(name, func(t *testing.T) {
			if _, err := app.uiSettingsService.ApplyStyleDiff(
				fmt.Sprintf(`{"panel_language":"繁中","role_language":%q}`, c.label)); err != nil {
				t.Fatalf("切角色語言 %q: %v", c.label, err)
			}
			if got := app.responseLanguage(); got != c.code {
				t.Fatalf("responseLanguage = %q, 期望 %q", got, c.code)
			}

			traceID := fmt.Sprintf("chat-langmatrix-%d", i)
			resp, err := app.SendAPIMessage(adapterID, "langmatrix", c.text, traceID)
			if err != nil {
				if isConnRefused(err) {
					t.Skipf("Ollama 連線中斷：%v", err)
				}
				t.Fatalf("SendAPIMessage: %v", err)
			}

			action, target, found := findLocalAssembled(traceID)

			// infra-lenient 題型：只記錄，不硬斷言語系。
			if c.check == "" {
				if found {
					t.Logf("[infra] action=%q target=%q", action, target)
				} else {
					t.Logf("[infra] 走 Go 端直連處理；resp.action=%q text=%q", resp.Action, langMatrixTruncate(resp.Text, 60))
				}
				return
			}

			if !found {
				t.Fatalf("找不到 localAssembled（traceID=%s）；resp.action=%q text=%q", traceID, resp.Action, langMatrixTruncate(resp.Text, 80))
			}
			if len(c.wantActions) > 0 && !langMatrixContains(c.wantActions, action) {
				t.Errorf("action=%q，期望屬於 %v（target=%q）", action, c.wantActions, target)
			}
			switch c.check {
			case "presence":
				if !matchesExpectedLanguage(c.code, target) {
					t.Errorf("target 不像 %s：%q", c.code, target)
				}
			case "leak":
				if languageScriptLeak(c.code, target) {
					t.Errorf("選項 label 混入矛盾 script（%s）：%q", c.code, target)
				}
			}
			t.Logf("OK action=%q target=%q", action, target)
		})
		time.Sleep(150 * time.Millisecond) // 給本地模型喘息，降低 Ollama 排隊壓力
	}
}

func findLocalAssembled(traceID string) (action, target string, found bool) {
	events := debugtrace.EventsSnapshot()
	for i := len(events) - 1; i >= 0; i-- { // 由新到舊，取最後一筆
		e := events[i]
		if e.TraceID != traceID || e.Node != "go.APIMessage.localAssembled" {
			continue
		}
		m, ok := e.Data.(map[string]interface{})
		if !ok {
			continue
		}
		action, _ = m["action"].(string)
		target, _ = m["target"].(string)
		return action, target, true
	}
	return "", "", false
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func isConnRefused(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "connection refused") || strings.Contains(s, "connect: ") || strings.Contains(s, "dial tcp")
}

func langMatrixContains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func langMatrixTruncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

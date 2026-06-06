// app_url_visual_learning_binding.go — SEC-06: 從 URL 開頁做視覺學習的獨立入口。
//
// 與既有 StartLearningMode（使用者本地操作錄製）分離：本入口開「ephemeral
// browser profile」——用完即丟的乾淨 profile，永不接觸使用者日常 cookie /
// 登入狀態。現有學習流程與使用者選 browser/profile 的行為完全不變。
//
// 支援度：Chrome / Edge（--user-data-dir）、Firefox（-profile -no-remote）。
// Safari 無乾淨 profile 機制 → 直接阻擋並提示換瀏覽器。
package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"ui_console/adapter/debugtrace"
	"ui_console/domain/url_source"
	"ui_console/internal/urlsafe"
	"ui_console/shared/browser_pref"
	"ui_console/shared/executil"
)

// ── ephemeral profile 執行狀態（同時只允許一個 URL 學習 run）──

var (
	urlLearnMu    sync.Mutex
	urlLearnState *urlLearningRun
)

type urlLearningRun struct {
	RunID      string
	ProfileDir string
	Browser    browser_pref.BrowserKind
	Host       string
	Cmd        *exec.Cmd
	StartedAt  time.Time
}

// StartURLVisualLearning 從 URL 開乾淨瀏覽器並啟動視覺學習錄製。
func (a *App) StartURLVisualLearning(rawURL, urlSource, activeWindowHash string) (interface{}, error) {
	src, ok := url_source.ValidSource(urlSource)
	if !ok {
		return nil, fmt.Errorf("未知的 url_source: %q", urlSource)
	}

	// — URL 檢查：僅 http/https + metadata/危險 IP 硬擋 —
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("無法解析 URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("僅允許 http/https（收到 %q）", scheme)
	}
	if err := urlsafe.ScreenExternalOpenTarget(u.Hostname()); err != nil {
		return nil, err
	}

	// — provenance —
	rec, err := getURLRegistry().Record(rawURL, src, "", "", a.trustLabelFor(rawURL))
	if err != nil {
		return nil, err
	}

	urlLearnMu.Lock()
	defer urlLearnMu.Unlock()
	if urlLearnState != nil {
		return nil, fmt.Errorf("已有進行中的 URL 視覺學習（%s），請先停止", urlLearnState.Host)
	}

	// — browser 支援度 —
	pref := a.browserService.Get()
	exe, args, supported := ephemeralBrowserCommand(pref.Browser, rawURL)
	if !supported {
		return nil, fmt.Errorf("瀏覽器 %s 不支援乾淨 profile，URL 視覺學習需要 Chrome / Edge / Firefox，請先在設定切換", pref.Browser)
	}

	// — 建 temp profile + 啟動 —
	runID := fmt.Sprintf("urlvl-%d", time.Now().UnixMilli())
	profileDir := filepath.Join(appDataRoot(), "runtime", "ephemeral_browser_profiles", runID)
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return nil, fmt.Errorf("無法建立 ephemeral profile: %w", err)
	}
	args = withProfileArgs(pref.Browser, profileDir, args)
	cmd := executil.Command(exe, args...)
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(profileDir)
		return nil, fmt.Errorf("無法啟動 %s: %w", pref.Browser, err)
	}

	// — 接現有錄製流程 —
	startResult, err := a.StartLearningMode(activeWindowHash)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = os.RemoveAll(profileDir)
		return nil, err
	}

	urlLearnState = &urlLearningRun{
		RunID:      runID,
		ProfileDir: profileDir,
		Browser:    pref.Browser,
		Host:       rec.NormalizedHost,
		Cmd:        cmd,
		StartedAt:  time.Now(),
	}
	debugtrace.Record("url_visual_learning.start", "", map[string]interface{}{
		"run_id": runID, "host": rec.NormalizedHost,
		"url_source": string(src), "browser": string(pref.Browser),
		"profile_isolated": true,
	})
	return frontendDTO(map[string]interface{}{
		"run_id":           runID,
		"host":             rec.NormalizedHost,
		"browser":          string(pref.Browser),
		"profile_isolated": true,
		"learning":         startResult,
	}), nil
}

// StopURLVisualLearning 停止錄製、關閉瀏覽器、清掉 temp profile。
func (a *App) StopURLVisualLearning() (interface{}, error) {
	urlLearnMu.Lock()
	state := urlLearnState
	urlLearnState = nil
	urlLearnMu.Unlock()
	if state == nil {
		return nil, fmt.Errorf("沒有進行中的 URL 視覺學習")
	}

	stopResult, stopErr := a.StopLearningMode()

	if state.Cmd != nil && state.Cmd.Process != nil {
		if err := state.Cmd.Process.Kill(); err != nil {
			log.Printf("url_visual_learning: kill browser: %v", err)
		}
		_, _ = state.Cmd.Process.Wait()
	}
	// 稍等檔案鎖釋放再清 profile；失敗只 log（下次啟動可再清）
	time.Sleep(300 * time.Millisecond)
	if err := os.RemoveAll(state.ProfileDir); err != nil {
		log.Printf("url_visual_learning: 清除 ephemeral profile 失敗（將於下次啟動重試）: %v", err)
	}
	debugtrace.Record("url_visual_learning.stop", "", map[string]interface{}{
		"run_id": state.RunID, "host": state.Host,
		"profile_isolated": true, "profile_cleaned": true,
	})
	if stopErr != nil {
		return nil, stopErr
	}
	return stopResult, nil
}

// CleanupEphemeralProfiles 啟動時清掉殘留 profile（上次未正常停止）。
// 由 startup 流程或前端呼叫皆可，冪等。
func (a *App) CleanupEphemeralProfiles() error {
	root := filepath.Join(appDataRoot(), "runtime", "ephemeral_browser_profiles")
	urlLearnMu.Lock()
	active := ""
	if urlLearnState != nil {
		active = urlLearnState.RunID
	}
	urlLearnMu.Unlock()
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil // 目錄不存在 = 沒東西可清
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() != active {
			_ = os.RemoveAll(filepath.Join(root, e.Name()))
		}
	}
	return nil
}

// ephemeralBrowserCommand 回傳瀏覽器執行檔與基礎參數；不支援者 supported=false。
func ephemeralBrowserCommand(kind browser_pref.BrowserKind, rawURL string) (exe string, args []string, supported bool) {
	switch kind {
	case browser_pref.BrowserChrome:
		return findBrowserExe("chrome", chromePaths()), []string{"--no-first-run", "--no-default-browser-check", "--new-window", rawURL}, true
	case browser_pref.BrowserEdge:
		return findBrowserExe("msedge", edgePaths()), []string{"--no-first-run", "--no-default-browser-check", "--new-window", rawURL}, true
	case browser_pref.BrowserFirefox:
		return findBrowserExe("firefox", firefoxPaths()), []string{"-no-remote", "-new-window", rawURL}, true
	default: // Safari 等：無乾淨 profile 機制
		return "", nil, false
	}
}

// withProfileArgs 把 profile 目錄參數插到最前（Chrome/Edge 與 Firefox 語法不同）。
func withProfileArgs(kind browser_pref.BrowserKind, profileDir string, args []string) []string {
	if kind == browser_pref.BrowserFirefox {
		return append([]string{"-profile", profileDir}, args...)
	}
	return append([]string{"--user-data-dir=" + profileDir}, args...)
}

// findBrowserExe 先 PATH 再常見安裝路徑；找不到回傳名稱讓 Start 報自然錯誤。
func findBrowserExe(name string, candidates []string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return name
}

func chromePaths() []string {
	if runtime.GOOS != "windows" {
		return []string{"/usr/bin/google-chrome", "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"}
	}
	return []string{
		filepath.Join(os.Getenv("ProgramFiles"), `Google\Chrome\Application\chrome.exe`),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), `Google\Chrome\Application\chrome.exe`),
		filepath.Join(os.Getenv("LOCALAPPDATA"), `Google\Chrome\Application\chrome.exe`),
	}
}

func edgePaths() []string {
	if runtime.GOOS != "windows" {
		return []string{"/usr/bin/microsoft-edge", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"}
	}
	return []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(os.Getenv("ProgramFiles"), `Microsoft\Edge\Application\msedge.exe`),
	}
}

func firefoxPaths() []string {
	if runtime.GOOS != "windows" {
		return []string{"/usr/bin/firefox", "/Applications/Firefox.app/Contents/MacOS/firefox"}
	}
	return []string{
		filepath.Join(os.Getenv("ProgramFiles"), `Mozilla Firefox\firefox.exe`),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), `Mozilla Firefox\firefox.exe`),
	}
}

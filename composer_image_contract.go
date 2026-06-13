// composer_image_contract.go — 「訊息可帶圖」的後端合約。
//
// 設計目標（呼應前面討論）：
//  1. 不落地：圖片以 base64 留在記憶體，API 路徑直接內嵌進 OpenAI 相容的 vision
//     請求，全程不寫檔，所以不需要 temp 資料夾 / janitor / 定期清理。
//  2. 不動共用 struct：app.go 的 openAIChatMessage 同時被「請求」與「回應解析」
//     共用（回應端把 Content 當字串讀），所以這裡另開一組 vision-only 的請求型別，
//     只在「本回合有附圖」時使用，零風險於既有文字路徑與回應解析。
//  3. 兩條路徑都接：API 走 base64 內嵌；CLI inline 讀圖是各 CLI 自家介面，這裡先把
//     圖片消費掉並附上誠實提示（cliImageNotice），留好 hook 讓你接該 CLI 的 image flag。
//
// 流程：前端送出前呼叫 StageSessionImages 暫存本回合圖片 → SendAPIMessage /
// sendCLIMessage 用 takeSessionImages 取用並清除（消費一次）。
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// maxStagedImageBytes 伺服端再擋一次的單張上限（與前端 8MB 一致）。
const maxStagedImageBytes = 8 * 1024 * 1024

// sessionImageStore 以 sessionID 為鍵的本回合圖片暫存（純記憶體、消費後即清）。
var (
	sessionImageMu    sync.Mutex
	sessionImageStore = map[string][]composerImage{}
)

// composerImage 一張已從 data URL 解析出的附帶圖片。
type composerImage struct {
	MIME    string // 例：image/png
	DataB64 string // 純 base64（不含 data: 前綴）
}

// dataURL 還原成 OpenAI vision 需要的 data URL 形式。
func (c composerImage) dataURL() string {
	return "data:" + c.MIME + ";base64," + c.DataB64
}

// parseComposerImageDataURL 解析 "data:image/png;base64,xxxx"，並做大小上限檢查。
func parseComposerImageDataURL(dataURL string) (composerImage, error) {
	const marker = ";base64,"
	idx := strings.Index(dataURL, marker)
	if !strings.HasPrefix(dataURL, "data:image/") || idx < 0 {
		return composerImage{}, fmt.Errorf("invalid image data URL")
	}
	mime := dataURL[len("data:"):idx]
	raw := dataURL[idx+len(marker):]
	dec, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return composerImage{}, fmt.Errorf("base64 decode failed: %w", err)
	}
	if len(dec) == 0 {
		return composerImage{}, fmt.Errorf("empty image")
	}
	if len(dec) > maxStagedImageBytes {
		return composerImage{}, fmt.Errorf("image too large")
	}
	return composerImage{MIME: mime, DataB64: raw}, nil
}

// StageSessionImages 前端在送訊息前呼叫，暫存本回合要附的圖片。
// 回傳成功暫存的張數；個別解析失敗的圖片會被略過，不中斷送出。
func (a *App) StageSessionImages(sessionID string, dataURLs []string) (int, error) {
	imgs := make([]composerImage, 0, len(dataURLs))
	for _, u := range dataURLs {
		img, err := parseComposerImageDataURL(u)
		if err != nil {
			continue
		}
		imgs = append(imgs, img)
	}
	sessionImageMu.Lock()
	if len(imgs) == 0 {
		delete(sessionImageStore, sessionID)
	} else {
		sessionImageStore[sessionID] = imgs
	}
	sessionImageMu.Unlock()
	return len(imgs), nil
}

// ClearSessionImages 前端在清除預覽 / 送出失敗時呼叫，避免圖片殘留到下一則。
func (a *App) ClearSessionImages(sessionID string) {
	sessionImageMu.Lock()
	delete(sessionImageStore, sessionID)
	sessionImageMu.Unlock()
}

// takeSessionImages 後端取用並清除該 session 暫存的圖片（消費一次，避免外洩到下一則）。
func takeSessionImages(sessionID string) []composerImage {
	sessionImageMu.Lock()
	imgs := sessionImageStore[sessionID]
	delete(sessionImageStore, sessionID)
	sessionImageMu.Unlock()
	return imgs
}

// --- OpenAI 相容 vision 請求（只在有圖時使用）-------------------------------

type openAIVisionImageURL struct {
	URL string `json:"url"`
}

type openAIVisionPart struct {
	Type     string                `json:"type"`
	Text     string                `json:"text,omitempty"`
	ImageURL *openAIVisionImageURL `json:"image_url,omitempty"`
}

type openAIVisionMessage struct {
	Role    string             `json:"role"`
	Content []openAIVisionPart `json:"content"`
}

type openAIVisionRequest struct {
	Model    string                `json:"model"`
	Messages []openAIVisionMessage `json:"messages"`
}

// buildOpenAIRequestBody 依是否有附圖回傳對應的請求 JSON：
//   - 無圖：沿用既有的 openAIChatRequest（Content 為字串），行為與改動前完全相同。
//   - 有圖：用 openAIVisionRequest，把 prompt 當 text part、每張圖當 image_url part
//     （base64 內嵌、不落地）。
func buildOpenAIRequestBody(model, prompt string, imgs []composerImage) ([]byte, error) {
	if len(imgs) == 0 {
		return json.Marshal(openAIChatRequest{
			Model:    model,
			Messages: []openAIChatMessage{{Role: "user", Content: prompt}},
		})
	}
	parts := make([]openAIVisionPart, 0, len(imgs)+1)
	parts = append(parts, openAIVisionPart{Type: "text", Text: prompt})
	for _, img := range imgs {
		parts = append(parts, openAIVisionPart{
			Type:     "image_url",
			ImageURL: &openAIVisionImageURL{URL: img.dataURL()},
		})
	}
	return json.Marshal(openAIVisionRequest{
		Model:    model,
		Messages: []openAIVisionMessage{{Role: "user", Content: parts}},
	})
}

// cliImageNotice CLI adapter 尚未支援讀圖時，回傳要附到 prompt 的提示，
// 讓圖片不被靜默吞掉。真正接 CLI vision 時改這裡：依該 CLI 的 image flag /
// stdin 介面把 imgs 編碼進去（若該 CLI 只能吃路徑，再於此處用隔離資料夾落地並
// 於呼叫結束後即刪，與本檔「不落地」原則一致地把落地侷限在這個 hook 內）。
func cliImageNotice(imgs []composerImage) string {
	if len(imgs) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"\n\n[註：使用者附了 %d 張圖片，但目前的 CLI adapter 尚未支援讀圖；"+
			"請改用 API / vision adapter 才能讓模型看到圖片。]",
		len(imgs),
	)
}

// vision_organ.go —「視覺器官」：本地開源 VLM 看圖轉文字。
//
// 設計理念（呼應與使用者的討論）：
//   大腦（claude / codex / gemini / 本地文字模型）不碰像素。看圖是一個獨立的
//   本地多模態模型（預設 Qwen2.5-VL，跑在 Ollama），由它把圖片轉成「結構化中文
//   描述」，再注入大腦的 prompt。這樣大腦是誰都無所謂，連純文字 CLI 也能間接看圖。
//
// 與既有原則一致：
//   - 不落地：圖片以 base64 直接塞進 Ollama /api/generate 的 images 欄位，全程在記憶體。
//   - loopback-only：器官只連本機（127.0.0.1 / localhost / [::1]），由 PolicyLocalLLM
//     再擋一次 LAN/外網（與 highlight 本地模型同一守門）。
//   - 提示工程：把「使用者這一輪的問題」一起餵給器官，描述會聚焦在相關內容，
//     不產通用 caption、不浪費 token，並要求不臆測。
//
// 接線策略（見 composer_image_contract.go / app.go）：
//   - 多模態大腦走 API 路徑 → 直送圖（既有 buildOpenAIRequestBody），不經器官。
//   - 純文字大腦 / 尚未支援直送圖的 CLI 路徑 → 經器官轉文字後注入。
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"ui_console/internal/urlsafe"
)

// visionOrganClient 走 PolicyLocalLLM：允許 loopback、封鎖 LAN/外網。
// timeout 比文字模型寬一些，因為 VLM 推論較慢。
var visionOrganClient = urlsafe.NewSafeClient(urlsafe.PolicyLocalLLM, "vision_organ", 60*time.Second)

// visionOrganModel 回傳器官使用的模型與 endpoint（皆可由環境變數覆寫）。
// 預設 qwen2.5vl（Ollama 標籤）；若使用者 pull 的是 7b/3b 變體，可設 VISION_ORGAN_MODEL。
func visionOrganModel() (model, endpoint string) {
	model = strings.TrimSpace(os.Getenv("VISION_ORGAN_MODEL"))
	if model == "" {
		model = "qwen2.5vl"
	}
	endpoint = strings.TrimSpace(os.Getenv("VISION_ORGAN_ENDPOINT"))
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434/api/generate"
	}
	return model, endpoint
}

// voOllamaReq — Ollama /api/generate 的多模態請求。
// Images 為純 base64（不含 data: 前綴）陣列；多模態模型才會讀。
type voOllamaReq struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images,omitempty"`
	Stream bool     `json:"stream"`
}

type voOllamaResp struct {
	Response string `json:"response"`
}

// buildVisionOrganPrompt 組裝餵給 VLM 的提示，聚焦使用者問題、要求不臆測。
// userText 可為空（純附圖無文字時，退回通用但仍要求忠實的描述）。
func buildVisionOrganPrompt(userText string) string {
	q := strings.TrimSpace(userText)
	var b strings.Builder
	b.WriteString("你是一個視覺辨識器官，負責把圖片內容如實轉述成繁體中文，交給另一個沒有看到圖片的助理使用。\n")
	b.WriteString("規則：\n")
	b.WriteString("1. 只描述圖片中實際存在的內容，不得臆測、不得編造看不到的文字、數字或來源。\n")
	b.WriteString("2. 若圖片含文字（截圖、表格、文件、招牌等），逐字轉錄關鍵文字，並說明版面位置關係。\n")
	b.WriteString("3. 描述要結構化、精簡，避免主觀評論。\n")
	if q != "" {
		b.WriteString("4. 使用者針對這張圖的問題是：「")
		b.WriteString(q)
		b.WriteString("」。請優先、詳細描述與這個問題相關的內容，其餘可略。\n")
	} else {
		b.WriteString("4. 使用者沒有附加問題，請給一段忠實、可被後續追問的整體描述。\n")
	}
	b.WriteString("\n現在開始描述：")
	return b.String()
}

// buildVisionOrganRequestBody 組裝請求 JSON（純函式，便於測試）。
func buildVisionOrganRequestBody(model, userText string, imgs []composerImage) ([]byte, error) {
	if len(imgs) == 0 {
		return nil, fmt.Errorf("vision organ: no images")
	}
	b64s := make([]string, 0, len(imgs))
	for _, img := range imgs {
		b64s = append(b64s, img.DataB64)
	}
	return json.Marshal(voOllamaReq{
		Model:  model,
		Prompt: buildVisionOrganPrompt(userText),
		Images: b64s,
		Stream: false,
	})
}

// describeImages 呼叫本地 VLM，把附圖轉成結構化中文描述。
// 失敗（器官未啟動 / 模型不可用 / 逾時）回 error，由呼叫端決定是否退回誠實提示。
func describeImages(ctx context.Context, userText string, imgs []composerImage) (string, error) {
	if len(imgs) == 0 {
		return "", fmt.Errorf("vision organ: no images")
	}
	model, endpoint := visionOrganModel()
	if !isLoopbackEndpoint(endpoint) {
		return "", fmt.Errorf("vision organ endpoint 必須為 loopback：%s", endpoint)
	}
	body, err := buildVisionOrganRequestBody(model, userText, imgs)
	if err != nil {
		return "", err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := visionOrganClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("vision organ 不可用（請確認已 ollama pull %s 且服務啟動）：%w", model, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("vision organ HTTP %d", resp.StatusCode)
	}
	var decoded voOllamaResp
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&decoded); err != nil {
		return "", err
	}
	out := strings.TrimSpace(decoded.Response)
	if out == "" {
		return "", fmt.Errorf("vision organ 回傳空描述")
	}
	return out, nil
}

// visionOrganInjection 把器官描述包成可注入 prompt 的區塊，標示來源是「轉述」而非原圖。
func visionOrganInjection(description string) string {
	d := strings.TrimSpace(description)
	if d == "" {
		return ""
	}
	return fmt.Sprintf(
		"\n\n[視覺器官轉述：以下是本地視覺模型對使用者所附圖片的客觀描述（你看不到原圖，請以此為準，勿臆測圖中未提及的內容）]\n%s\n[視覺器官轉述結束]",
		d,
	)
}

// isLoopbackEndpoint 僅允許本機位址，與 highlight 本地模型同一守門邏輯。
func isLoopbackEndpoint(endpoint string) bool {
	return strings.HasPrefix(endpoint, "http://127.0.0.1:") ||
		strings.HasPrefix(endpoint, "http://localhost:") ||
		strings.HasPrefix(endpoint, "http://[::1]:")
}

// --- 大腦多模態能力判斷 ------------------------------------------------------

// multimodalModelPattern 比對「模型名稱本身就支援讀圖」的常見命名。
// 例：gpt-4o、claude-3.5、gemini-1.5、qwen2.5-vl、llava、moondream、minicpm-v、pixtral。
var multimodalModelPattern = regexp.MustCompile(
	`(?i)(gpt-4o|gpt-4\.1|gpt-5|claude|gemini|qwen.*vl|llava|moondream|minicpm-?v|pixtral|llama.*vision|vision|-vl\b|janus|internvl|cogvlm)`,
)

// multimodalCLIAdapters 列出「CLI 本身對接的就是多模態大腦」的 adapter。
// 這些走 API 路徑時可直送圖；CLI 路徑的直送圖介面為後續工作，預設仍可退回器官。
var multimodalCLIAdapters = map[string]bool{
	"claude-cli": true,
	"gemini-cli": true,
	"codex-cli":  true,
}

// brainIsMultimodal 判斷目前大腦是否能自己讀圖。
// adapterID 命中已知多模態 CLI，或 model 名稱命中多模態樣式，即視為可直送。
func brainIsMultimodal(adapterID, model string) bool {
	if multimodalCLIAdapters[strings.ToLower(strings.TrimSpace(adapterID))] {
		return true
	}
	if m := strings.TrimSpace(model); m != "" && multimodalModelPattern.MatchString(m) {
		return true
	}
	return false
}

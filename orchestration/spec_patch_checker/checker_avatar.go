// spec_patch_checker/checker_avatar.go — v3.5.0 Persona Avatar 守則（§10）。
// 共 12 條守則，確保 Avatar 不影響核心決策系統。
package spec_patch_checker

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ──────────────────────────────────────────────
// §10 Persona Avatar 守則
// ──────────────────────────────────────────────

// CheckAvatarNotInLLMContext 驗證 Avatar 資料不出現在 LLM context payload。
// §10.1：Avatar 僅影響 UI 顯示，不得注入 LLM context。
func CheckAvatarNotInLLMContext(contextPayloadJSON string) error {
	lower := strings.ToLower(contextPayloadJSON)
	forbidden := []string{
		"avatar_expression", "avatar_mood", "avatar_state",
		"persona_avatar", "avatar_image", "avatar_emotion",
	}
	for _, kw := range forbidden {
		if strings.Contains(lower, kw) {
			return fmt.Errorf("§10.1 違規: LLM context 不得包含 Avatar 欄位 %s", kw)
		}
	}
	return nil
}

// CheckAvatarDoesNotAffectRiskPolicy 驗證 Avatar 不影響風險政策。
// §10.1：Avatar 狀態變更不得觸發風險等級變動。
func CheckAvatarDoesNotAffectRiskPolicy(eventJSON string) error {
	var event struct {
		Source     string `json:"source"`
		Target     string `json:"target"`
		FieldChanged string `json:"field_changed"`
	}
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		return nil
	}
	if event.Source == "persona_avatar" && event.Target == "risk_policy" {
		return fmt.Errorf("§10.1 違規: Avatar 不得影響風險政策（變更欄位: %s）", event.FieldChanged)
	}
	return nil
}

// CheckAvatarDoesNotAffectSourceTrust 驗證 Avatar 不影響來源信任。
// §10.1：Avatar 不得修改來源信任分數或白名單。
func CheckAvatarDoesNotAffectSourceTrust(eventJSON string) error {
	var event struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		return nil
	}
	if event.Source == "persona_avatar" && event.Target == "source_trust" {
		return fmt.Errorf("§10.1 違規: Avatar 不得影響來源信任系統")
	}
	return nil
}

// CheckAvatarDoesNotAffectToolRouting 驗證 Avatar 不影響工具路由。
// §10.1：Avatar 狀態不得改變工具選擇或路由邏輯。
func CheckAvatarDoesNotAffectToolRouting(eventJSON string) error {
	var event struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		return nil
	}
	if event.Source == "persona_avatar" && event.Target == "tool_routing" {
		return fmt.Errorf("§10.1 違規: Avatar 不得影響工具路由")
	}
	return nil
}

// CheckAvatarDoesNotAffectMemory 驗證 Avatar 不影響記憶系統。
// §10.1：Avatar 狀態不得寫入主記憶檔案。
func CheckAvatarDoesNotAffectMemory(eventJSON string) error {
	var event struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
		return nil
	}
	if event.Source == "persona_avatar" && event.Target == "memory" {
		return fmt.Errorf("§10.1 違規: Avatar 不得影響記憶系統")
	}
	return nil
}

// CheckAvatarMetadataNotInMemoryFiles 驗證 Avatar 中繼資料不在記憶檔案中。
// §10.3：記憶檔案不得包含 Avatar 中繼資料。
func CheckAvatarMetadataNotInMemoryFiles(memoryContent string) error {
	lower := strings.ToLower(memoryContent)
	avatarMetaKeys := []string{
		"avatar_provider", "avatar_preset", "avatar_style",
		"pixel_palette", "avatar_config", "expression_map",
	}
	for _, key := range avatarMetaKeys {
		if strings.Contains(lower, key) {
			return fmt.Errorf("§10.3 違規: 記憶檔案不得包含 Avatar 中繼資料 %s", key)
		}
	}
	return nil
}

// CheckAvatarAPIPromptNoRawConversation 驗證 Avatar API 提示不含原始對話。
// §10.2：傳給圖像 API 的 prompt 不得包含使用者的原始對話內容。
func CheckAvatarAPIPromptNoRawConversation(promptJSON string) error {
	var prompt struct {
		ContainsRawConversation bool `json:"contains_raw_conversation"`
		ContainsUserMessages    bool `json:"contains_user_messages"`
	}
	if err := json.Unmarshal([]byte(promptJSON), &prompt); err != nil {
		return nil
	}
	if prompt.ContainsRawConversation || prompt.ContainsUserMessages {
		return fmt.Errorf("§10.2 違規: Avatar API prompt 不得包含原始對話內容")
	}
	return nil
}

// CheckAvatarAPIKeyNotInExport 驗證 Avatar API 金鑰不在匯出中。
// §10.2：匯出檔案不得包含圖像生成 API 金鑰。
func CheckAvatarAPIKeyNotInExport(exportContent string) error {
	patterns := []string{
		"openai_api_key", "stability_api_key", "dalle_key",
		"image_gen_key", "avatar_api_key",
	}
	lower := strings.ToLower(exportContent)
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return fmt.Errorf("§10.2 違規: 匯出不得包含 Avatar API 金鑰 %s", p)
		}
	}
	return nil
}

// CheckRendererDoesNotSendFullScreenshot 驗證渲染器不傳送完整螢幕截圖。
// §10.4：Avatar 渲染器不得將完整桌面截圖傳給外部 API。
func CheckRendererDoesNotSendFullScreenshot(requestJSON string) error {
	var req struct {
		ImageType string `json:"image_type"`
		ImageSize int64  `json:"image_size_bytes"`
	}
	if err := json.Unmarshal([]byte(requestJSON), &req); err != nil {
		return nil
	}
	if req.ImageType == "full_screenshot" || req.ImageType == "desktop_capture" {
		return fmt.Errorf("§10.4 違規: 渲染器不得傳送完整螢幕截圖")
	}
	// 圖片大小超過 1MB 視為可疑（Avatar 用小圖）
	if req.ImageSize > 1024*1024 {
		return fmt.Errorf("§10.4 違規: Avatar 圖片不應超過 1MB（目前 %d bytes）", req.ImageSize)
	}
	return nil
}

// CheckRendererDoesNotWriteDirectly 驗證渲染器不直接寫入核心檔案。
// §10.4：Avatar 渲染器僅寫入 avatar/ 目錄。
func CheckRendererDoesNotWriteDirectly(writePathJSON string) error {
	var wp struct {
		Source string `json:"source"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal([]byte(writePathJSON), &wp); err != nil {
		return nil
	}
	if wp.Source == "avatar_renderer" {
		// 只允許寫入 avatar/ 或 persona_avatar/ 開頭的路徑
		if !strings.HasPrefix(wp.Path, "avatar/") && !strings.HasPrefix(wp.Path, "persona_avatar/") {
			return fmt.Errorf("§10.4 違規: Avatar 渲染器僅可寫入 avatar/ 目錄，不得寫入 %s", wp.Path)
		}
	}
	return nil
}

// CheckSVGNotAcceptedInMVP 驗證 MVP 不接受 SVG 格式。
// §10.5：MVP 階段僅支援 PNG/JPEG，不接受 SVG 以避免 XSS。
func CheckSVGNotAcceptedInMVP(fileInfoJSON string) error {
	var info struct {
		Format  string `json:"format"`
		IsMVP   bool   `json:"is_mvp"`
	}
	if err := json.Unmarshal([]byte(fileInfoJSON), &info); err != nil {
		return nil
	}
	if info.IsMVP && (info.Format == "svg" || info.Format == "SVG") {
		return fmt.Errorf("§10.5 違規: MVP 階段不接受 SVG 格式")
	}
	return nil
}

// CheckPixelAssetsLicensed 驗證像素資源有授權資訊。
// §10.5：所有像素資源必須附帶授權聲明。
func CheckPixelAssetsLicensed(assetJSON string) error {
	var asset struct {
		HasLicense  bool   `json:"has_license"`
		LicenseType string `json:"license_type"`
	}
	if err := json.Unmarshal([]byte(assetJSON), &asset); err != nil {
		return nil
	}
	if !asset.HasLicense {
		return fmt.Errorf("§10.5 違規: 像素資源必須附帶授權聲明")
	}
	if asset.LicenseType == "" {
		return fmt.Errorf("§10.5 違規: 像素資源缺少授權類型")
	}
	return nil
}

// w3a_media/transfer.go — §9A.12 原檔傳輸引導。
//
// ┌─────────────────────────────────────────────────────────────┐
// │ 匯出媒體時提供軟性 toast 提示，引導使用者以原檔方式傳輸。  │
// │                                                             │
// │ 推薦方式：原檔傳輸 / 附件模式 / 打包保留 sidecar            │
// │ 不推薦：社群平台直傳 / 壓縮模式 / 截圖轉發 / 非 W3A 工具   │
// │                                                             │
// │ UX：僅 toast 提示，不阻擋使用者操作                         │
// └─────────────────────────────────────────────────────────────┘
package w3a_media

// ──────────────────────────────────────────────
// 傳輸引導
// ──────────────────────────────────────────────

// GetTransferGuidance 回傳原檔傳輸引導建議（§9A.12）。
func GetTransferGuidance() TransferGuidance {
	return TransferGuidance{
		Recommended: []string{
			"使用原檔傳輸（不壓縮）",
			"使用檔案附件模式而非壓縮圖片模式",
			"使用打包方式保留 metadata 與 sidecar",
			"使用 W3A-aware app 進行編輯/匯出",
		},
		NotRecommended: []string{
			"社群平台直接上傳圖片作為保存副本",
			"通訊軟體壓縮圖片模式",
			"截圖轉發",
			"非 W3A-aware 濾鏡/美化/優化/轉換工具",
			"未知工具的音訊正規化或轉碼",
		},
		UIMessage: "為保留 W3A 原始簽章與訓練安全狀態，請使用「原檔 / 文件 / 附件」方式傳送。\n不要使用會壓縮圖片或音訊的平台傳送方式。",
	}
}

// ShouldShowTransferToast 判斷匯出時是否應顯示傳輸引導 toast。
// 只要媒體有 W3A 驗證資訊，就建議提示。
func ShouldShowTransferToast(status VerificationStatus) bool {
	switch status {
	case StatusExactOriginal, StatusW3AAppProcessed:
		return true // 有延伸權的媒體，特別需要提示
	case StatusPlatformProcessed:
		return true // 已是平台處理版本，提醒避免再次壓縮
	default:
		return false // unverified / content_modified 等不需要特別提示
	}
}

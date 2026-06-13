// source_trust/auth_ok.go — AUTH_OK 計算（§9.5）。
// 僅在 VERIFIED_AUTHORITY + allowlist 未過期 + 無 UGC/disclaimer 時為 true。
// 此函式由 controller 呼叫，LLM 不得自行判定。
package source_trust

// ComputeAuthOK 判斷來源是否通過完整認證。
// 三個條件必須全部滿足：
//  1. 信任標籤為 VERIFIED_AUTHORITY
//  2. Allowlist 狀態為 active（未過期）
//  3. 無 UGC 或 disclaimer 內容旗標
func ComputeAuthOK(e SourceTrustEvidence) bool {
	// 條件 1: 必須是已驗證權威
	if e.SourceTrustLabel != LabelVerifiedAuthority {
		return false
	}

	// 條件 2: allowlist 必須有效
	if e.AllowlistStatus != "active" {
		return false
	}

	// 條件 3: 不能有 UGC 或 disclaimer 旗標
	for _, flag := range e.ContentFlags {
		if flag == ContentDisclaimer || flag == ContentUserGenerated ||
			flag == ContentComment || flag == ContentForum {
			return false
		}
	}

	return true
}

// EnrichAuthOK 將 AUTH_OK 結果寫回 evidence（便利函式）。
func EnrichAuthOK(e *SourceTrustEvidence) {
	e.AuthOK = ComputeAuthOK(*e)
}

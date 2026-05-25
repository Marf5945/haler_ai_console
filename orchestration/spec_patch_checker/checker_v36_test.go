// spec_patch_checker/checker_v36_test.go — 核心 12 條守則測試。
// 涵蓋 §9（source_trust）、§10（avatar）、§11+§7.5（context）。
package spec_patch_checker

import (
	"testing"
)

// ──────────────────────────────────────────────
// §9 GatewaySentinel 守則測試（4 條核心）
// ──────────────────────────────────────────────

// 測試 LLM 不得判定高影響任務 AUTH_OK
func TestCheckLLMDoesNotDecideHighImpact(t *testing.T) {
	// 違規：LLM 自行判定
	err := CheckLLMDoesNotDecideHighImpact(`{"auth_ok_source":"llm"}`)
	if err == nil {
		t.Error("should reject LLM as auth_ok source")
	}

	// 合規：controller 判定
	err = CheckLLMDoesNotDecideHighImpact(`{"auth_ok_source":"controller"}`)
	if err != nil {
		t.Errorf("controller source should pass: %v", err)
	}
}

// 測試不得移除內建高影響領域
func TestCheckUserCannotRemoveBuiltInHighImpactDomain(t *testing.T) {
	// 違規：嘗試移除 medical
	err := CheckUserCannotRemoveBuiltInHighImpactDomain(`{"removed_domains":["medical"]}`)
	if err == nil {
		t.Error("should reject removing built-in domain")
	}

	// 合規：移除自訂領域
	err = CheckUserCannotRemoveBuiltInHighImpactDomain(`{"removed_domains":["custom_domain"]}`)
	if err != nil {
		t.Errorf("custom domain removal should pass: %v", err)
	}
}

// 測試白名單必須有 scope 和過期時間
func TestCheckAllowlistHasScopeAndExpiry(t *testing.T) {
	// 違規：缺少過期時間
	err := CheckAllowlistHasScopeAndExpiry(`{"allowed_for":["read"],"not_allowed_for":["write"]}`)
	if err == nil {
		t.Error("should reject missing expiry")
	}

	// 違規：缺少 allowed_for
	err = CheckAllowlistHasScopeAndExpiry(`{"expires_at":"2025-12-31","not_allowed_for":["write"]}`)
	if err == nil {
		t.Error("should reject missing allowed_for")
	}

	// 合規：完整條目
	err = CheckAllowlistHasScopeAndExpiry(`{"allowed_for":["read"],"not_allowed_for":["write"],"expires_at":"2025-12-31"}`)
	if err != nil {
		t.Errorf("complete entry should pass: %v", err)
	}
}

// 測試 UGC 不得成為 VERIFIED_AUTHORITY
func TestCheckUserGeneratedDoesNotBecomeVerifiedAuthority(t *testing.T) {
	// 違規：UGC + VERIFIED_AUTHORITY
	err := CheckUserGeneratedDoesNotBecomeVerifiedAuthority(`{"content_flags":["ugc_content"],"label":"VERIFIED_AUTHORITY"}`)
	if err == nil {
		t.Error("should reject UGC as verified authority")
	}

	// 合規：非 UGC 的 VERIFIED_AUTHORITY
	err = CheckUserGeneratedDoesNotBecomeVerifiedAuthority(`{"content_flags":["official_doc"],"label":"VERIFIED_AUTHORITY"}`)
	if err != nil {
		t.Errorf("non-UGC verified authority should pass: %v", err)
	}
}

// ──────────────────────────────────────────────
// §10 Persona Avatar 守則測試（5 條核心）
// ──────────────────────────────────────────────

// 測試 Avatar 不出現在 LLM context
func TestCheckAvatarNotInLLMContext(t *testing.T) {
	// 違規：包含 avatar_expression
	err := CheckAvatarNotInLLMContext(`{"blocks":[{"content":"avatar_expression:happy"}]}`)
	if err == nil {
		t.Error("should reject avatar data in LLM context")
	}

	// 合規：無 Avatar 欄位
	err = CheckAvatarNotInLLMContext(`{"blocks":[{"content":"user query about weather"}]}`)
	if err != nil {
		t.Errorf("clean context should pass: %v", err)
	}
}

// 測試 Avatar 不影響風險政策
func TestCheckAvatarDoesNotAffectRiskPolicy(t *testing.T) {
	// 違規：Avatar → risk_policy
	err := CheckAvatarDoesNotAffectRiskPolicy(`{"source":"persona_avatar","target":"risk_policy","field_changed":"risk_level"}`)
	if err == nil {
		t.Error("should reject avatar affecting risk policy")
	}

	// 合規：其他來源修改 risk_policy
	err = CheckAvatarDoesNotAffectRiskPolicy(`{"source":"user_action","target":"risk_policy","field_changed":"risk_level"}`)
	if err != nil {
		t.Errorf("non-avatar source should pass: %v", err)
	}
}

// 測試 Avatar 中繼資料不在記憶檔案中
func TestCheckAvatarMetadataNotInMemoryFiles(t *testing.T) {
	// 違規：記憶包含 avatar_provider
	err := CheckAvatarMetadataNotInMemoryFiles("user prefers avatar_provider: dalle")
	if err == nil {
		t.Error("should reject avatar metadata in memory")
	}

	// 合規：正常記憶內容
	err = CheckAvatarMetadataNotInMemoryFiles("user discussed project architecture")
	if err != nil {
		t.Errorf("clean memory should pass: %v", err)
	}
}

// 測試渲染器不傳送完整螢幕截圖
func TestCheckRendererDoesNotSendFullScreenshot(t *testing.T) {
	// 違規：full_screenshot
	err := CheckRendererDoesNotSendFullScreenshot(`{"image_type":"full_screenshot","image_size_bytes":500000}`)
	if err == nil {
		t.Error("should reject full screenshot")
	}

	// 違規：超大圖片
	err = CheckRendererDoesNotSendFullScreenshot(`{"image_type":"avatar_render","image_size_bytes":2000000}`)
	if err == nil {
		t.Error("should reject oversized image")
	}

	// 合規：正常 Avatar 圖片
	err = CheckRendererDoesNotSendFullScreenshot(`{"image_type":"avatar_render","image_size_bytes":50000}`)
	if err != nil {
		t.Errorf("normal avatar render should pass: %v", err)
	}
}

// 測試 MVP 不接受 SVG
func TestCheckSVGNotAcceptedInMVP(t *testing.T) {
	// 違規：MVP + SVG
	err := CheckSVGNotAcceptedInMVP(`{"format":"svg","is_mvp":true}`)
	if err == nil {
		t.Error("should reject SVG in MVP")
	}

	// 合規：MVP + PNG
	err = CheckSVGNotAcceptedInMVP(`{"format":"png","is_mvp":true}`)
	if err != nil {
		t.Errorf("PNG in MVP should pass: %v", err)
	}

	// 合規：非 MVP 的 SVG
	err = CheckSVGNotAcceptedInMVP(`{"format":"svg","is_mvp":false}`)
	if err != nil {
		t.Errorf("SVG in non-MVP should pass: %v", err)
	}
}

// ──────────────────────────────────────────────
// 跨章節通用守則測試（3 條核心）
// ──────────────────────────────────────────────

// 測試 Lightweight Card 不用於 scope 擴展
func TestCheckLightweightCardNotUsedForScopeExpansion(t *testing.T) {
	// 違規：Lightweight + 新增 scope
	err := CheckLightweightCardNotUsedForScopeExpansion(`{"card_type":"lightweight","old_scope":["read"],"new_scope":["read","write"]}`)
	if err == nil {
		t.Error("should reject scope expansion via lightweight card")
	}

	// 合規：Lightweight + 同 scope
	err = CheckLightweightCardNotUsedForScopeExpansion(`{"card_type":"lightweight","old_scope":["read","write"],"new_scope":["read"]}`)
	if err != nil {
		t.Errorf("same/reduced scope should pass: %v", err)
	}

	// 合規：Full review + 新增 scope（允許）
	err = CheckLightweightCardNotUsedForScopeExpansion(`{"card_type":"full","old_scope":["read"],"new_scope":["read","write"]}`)
	if err != nil {
		t.Errorf("full review scope expansion should pass: %v", err)
	}
}

// 測試 LLM Context Governance 不被繞過
func TestCheckLLMContextGovernanceNotBypassed(t *testing.T) {
	// 違規：未經 EntryFilter 就送 LLM
	err := CheckLLMContextGovernanceNotBypassed(`{"passed_entry_filter":false,"passed_exit_validate":true,"sent_to_llm":true}`)
	if err == nil {
		t.Error("should reject bypassed entry filter")
	}

	// 違規：未經 ExitValidate 就送 LLM
	err = CheckLLMContextGovernanceNotBypassed(`{"passed_entry_filter":true,"passed_exit_validate":false,"sent_to_llm":true}`)
	if err == nil {
		t.Error("should reject bypassed exit validate")
	}

	// 合規：完整通過
	err = CheckLLMContextGovernanceNotBypassed(`{"passed_entry_filter":true,"passed_exit_validate":true,"sent_to_llm":true}`)
	if err != nil {
		t.Errorf("fully validated should pass: %v", err)
	}
}

// 測試全域資源不被專案 purge 刪除
func TestCheckGlobalAssetNotPurgedByProjectPurge(t *testing.T) {
	// 違規：專案 purge 刪全域
	err := CheckGlobalAssetNotPurgedByProjectPurge(`{"scope":"project","target_paths":["global/config.json","project/temp"]}`)
	if err == nil {
		t.Error("should reject project purge deleting global assets")
	}

	// 合規：只刪專案資源
	err = CheckGlobalAssetNotPurgedByProjectPurge(`{"scope":"project","target_paths":["project/temp","project/cache"]}`)
	if err != nil {
		t.Errorf("project-only purge should pass: %v", err)
	}

	// 合規：全域 scope（允許）
	err = CheckGlobalAssetNotPurgedByProjectPurge(`{"scope":"global","target_paths":["global/config.json"]}`)
	if err != nil {
		t.Errorf("global scope purge should pass: %v", err)
	}
}

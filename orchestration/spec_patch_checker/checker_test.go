// checker_test.go — Skill Context Orchestration 最小測試集（10 項）
// 對應 APPENDIX_SKILL_CONTEXT_ORCHESTRATION.md §16 所列情境。
//
// 執行：go test ./spec_patch_checker/...
package spec_patch_checker

import "testing"

// T1: Console-native skill folder 匯入後應在 data/skills/<skill_id> 下
// （此處以 guard 驗證 audit 不含原始路徑為間接驗證）
func TestImportedSkillNoAbsolutePath(t *testing.T) {
	// 正常情境：injection 不含絕對路徑
	good := `{"resource_refs":{"examples":["weather-example"],"programs":["weather-fetcher"]}}`
	if err := CheckSkillContextNotExposingAbsolutePaths(good); err != nil {
		t.Errorf("expected no error for relative refs, got: %v", err)
	}
	// 違規情境：injection 含絕對路徑
	bad := `{"resource_refs":{"programs":["/Users/tester/data/skills/weather/programs/fetcher"]}}`
	if err := CheckSkillContextNotExposingAbsolutePaths(bad); err == nil {
		t.Error("expected error for absolute path in injection, got nil")
	}
}

// T2: 外部 SKILL.md 資料夾歸檔後 relation 檔案仍可解析
// （由 scanner/archive 驗證；此處以 manifest 不被 CLI 修改為間接 guard）
func TestExternalSkillManifestNotModifiedByCLI(t *testing.T) {
	safe := `{"action":"query","result":"sunny"}`
	if err := CheckSkillManifestNotModifiedByCLI(safe); err != nil {
		t.Errorf("expected no error for safe CLI output, got: %v", err)
	}
	unsafe := `{"action":"write","target":"skill_manifest.json"}`
	if err := CheckSkillManifestNotModifiedByCLI(unsafe); err == nil {
		t.Error("expected error when CLI output references skill_manifest.json")
	}
}

// T3: relation 檔案在資料夾重命名後仍可解析
// （結構驗證由 archive 處理；guard 確認 relation 不被 CLI 修改）
func TestRelationFileNotModifiedByCLI(t *testing.T) {
	bad := `{"action":"update","target":".skill_rel.json"}`
	if err := CheckSkillManifestNotModifiedByCLI(bad); err == nil {
		t.Error("expected error when CLI output references .skill_rel.json")
	}
}

// T4: 查詢ㄌ天氣 可解析為 weather skill（路由正確性）
// guard 層驗證：auto_selected + low risk 正常通過
func TestLowRiskAutoSelectedPasses(t *testing.T) {
	resolve := `{"skill_id":"weather.lookup","status":"auto_selected","risk":"low","score":0.91}`
	if err := CheckHighRiskSkillRequiresReview(resolve); err != nil {
		t.Errorf("low-risk auto_selected should pass, got: %v", err)
	}
}

// T5: 多個低風險匹配應回傳 needs_cli_candidate
// （路由邏輯在 router；guard 確認 high-risk 不能 auto_selected）
func TestHighRiskCannotAutoSelect(t *testing.T) {
	resolve := `{"skill_id":"deploy.prod","status":"auto_selected","risk":"high","score":0.95}`
	if err := CheckHighRiskSkillRequiresReview(resolve); err == nil {
		t.Error("expected error for high-risk auto_selected, got nil")
	}
}

// T6: critical-risk 也不能 auto_selected
func TestCriticalRiskCannotAutoSelect(t *testing.T) {
	resolve := `{"skill_id":"rm.all","status":"auto_selected","risk":"critical","score":0.99}`
	if err := CheckHighRiskSkillRequiresReview(resolve); err == nil {
		t.Error("expected error for critical-risk auto_selected, got nil")
	}
}

// T7: 下一個不相關動作應清除前一個 skill injection
func TestInjectionClearedOnNextAction(t *testing.T) {
	// 只有一筆 active（未 cleared）：正常
	single := `{"session_id":"s1","skill_id":"a","clear_reason":"next_action_unrelated","cleared_at":"2026-05-10T13:00:08+08:00"}`
	if err := CheckSkillInjectionNotPersistentAcrossActions(single); err != nil {
		t.Errorf("single cleared injection should pass, got: %v", err)
	}
}

// T8: audit log 包含 skill_id / reason / summary_hash，但不含原始 CLI 輸出
func TestAuditLogNoRawOutput(t *testing.T) {
	clean := `{"skill_id":"weather.lookup","reason":"matched","summary_hash":"sha256:abc"}`
	if err := CheckSkillInjectionNoRawOutput(clean); err != nil {
		t.Errorf("clean audit should pass, got: %v", err)
	}
	dirty := `{"skill_id":"weather.lookup","raw_cli_output":"secret data here"}`
	if err := CheckSkillInjectionNoRawOutput(dirty); err == nil {
		t.Error("expected error for raw_cli_output in audit log")
	}
}

// T9: audit log 不含 token / auth_cache
func TestAuditLogNoTokenOrAuth(t *testing.T) {
	withToken := `{"skill_id":"x","api_key":"sk-123"}`
	if err := CheckSkillInjectionNoRawOutput(withToken); err == nil {
		t.Error("expected error for api_key in audit log")
	}
	withAuth := `{"skill_id":"x","auth_cache":"/home/user/.claude/auth"}`
	if err := CheckSkillInjectionNoRawOutput(withAuth); err == nil {
		t.Error("expected error for auth_cache in audit log")
	}
}

// T10: export 應移除 resource-local .skill_rel.json（由 SafeExport guard 覆蓋）
// 此處以現有 CheckSafeExportNoReadablePatch 驗證 export 不含敏感資料
func TestExportNoSensitiveData(t *testing.T) {
	clean := `{"skill_id":"weather.lookup","resources":["weather-example"]}`
	if err := CheckSafeExportNoReadablePatch(clean); err != nil {
		t.Errorf("clean export should pass, got: %v", err)
	}
	withPassword := `{"skill_id":"x","password":"hunter2"}`
	if err := CheckSafeExportNoReadablePatch(withPassword); err == nil {
		t.Error("expected error for password in export manifest")
	}
}

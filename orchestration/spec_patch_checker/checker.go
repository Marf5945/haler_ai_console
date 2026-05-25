// Package spec_patch_checker implements the v3.3.2 P0 specPatchChecker guard
// (CI gate). It enforces the non-overridable rules from specs v3.0.0–v3.3.2.
//
// Each Check* function returns a non-nil error if the rule is violated.
// These functions are designed to be called in acceptance tests and CI
// pipelines, not in the hot path of production request handling.
package spec_patch_checker

import (
	"errors"
	"fmt"
	"strings"
)

// --- v3.0.0 Execution Hook guards ---

// CheckHookDoesNotWriteRegistryDuringRun enforces that hook evidence files do
// not contain direct mutations to tool_registry.json, risk_policy.json, DAG
// nodes, memory, or existing subagent definitions during a run.
//
// In production this would parse hook evidence JSONL; here we check that no
// forbidden keys appear in the raw payload.
func CheckHookDoesNotWriteRegistryDuringRun(hookEvidencePayload string) error {
	forbidden := []string{
		"tool_registry", "risk_policy", "dag_node_mutate",
		"memory_write", "subagent_replace", "subagent_delete",
	}
	for _, key := range forbidden {
		if strings.Contains(hookEvidencePayload, key) {
			return fmt.Errorf("specPatchChecker[v3.0.0]: hook evidence must not contain %q during run execution", key)
		}
	}
	return nil
}

// CheckCandidateDoesNotReplaceSubagent verifies that a new_subagent_candidate
// record does not carry any replace/disable/delete instruction.
func CheckCandidateDoesNotReplaceSubagent(candidateJSON string) error {
	forbidden := []string{"replace_subagent", "disable_subagent", "delete_subagent"}
	for _, key := range forbidden {
		if strings.Contains(candidateJSON, key) {
			return fmt.Errorf("specPatchChecker[v3.0.0]: new_subagent_candidate must not contain %q", key)
		}
	}
	return nil
}

// --- v3.1.0 Visual Learning guards ---

// CheckLearningModeNotBackground verifies that a learning run record indicates
// it was explicitly started by the user (not a background trigger).
func CheckLearningModeNotBackground(runJSON string) error {
	if strings.Contains(runJSON, `"background_triggered":true`) {
		return errors.New("specPatchChecker[v3.1.0]: Learning Mode must not be triggered in the background; requires explicit user activation")
	}
	return nil
}

// CheckSafeExportNoReadablePatch verifies that a SafeExportManifest does not
// include readable text patches, full screenshots, or sensitive data fields.
func CheckSafeExportNoReadablePatch(manifestJSON string) error {
	forbidden := []string{
		"readable_patch", "full_screenshot", "email", "token",
		"password", "api_key", "payment", "account_number",
	}
	for _, key := range forbidden {
		if strings.Contains(manifestJSON, key) {
			return fmt.Errorf("specPatchChecker[v3.1.0]: SafeExportManifest must not contain %q", key)
		}
	}
	return nil
}

// --- v3.2.0 Controlled Trust guards ---

// CheckOverrideDoesNotLowerFinalRisk verifies that a trust log entry records
// final_risk_changed=false.
func CheckOverrideDoesNotLowerFinalRisk(trustLogEntryJSON string) error {
	if strings.Contains(trustLogEntryJSON, `"final_risk_changed":true`) {
		return errors.New("specPatchChecker[v3.2.0]: trust log entry must have final_risk_changed=false; overrides must never lower final_risk")
	}
	return nil
}

// CheckOverrideDoesNotCoverCritical verifies that a ContextualRiskOverride
// does not cover critical or destructive operations.
func CheckOverrideDoesNotCoverCritical(overrideJSON string) error {
	forbidden := []string{`"operation_family":"critical"`, `"operation_family":"destructive"`}
	for _, f := range forbidden {
		if strings.Contains(overrideJSON, f) {
			return fmt.Errorf("specPatchChecker[v3.2.0]: contextual override must not cover %s operations", f)
		}
	}
	return nil
}

// CheckDraftSandboxNotWritingFormalDictionary verifies that a draft sandbox
// trace record does not contain formal dictionary write operations.
func CheckDraftSandboxNotWritingFormalDictionary(sandboxTraceJSON string) error {
	forbidden := []string{
		"write_action_dictionary", "write_element_dictionary",
		"write_canonical_schema",
	}
	for _, key := range forbidden {
		if strings.Contains(sandboxTraceJSON, key) {
			return fmt.Errorf("specPatchChecker[v3.2.0]: draft sandbox must not write formal dictionary: found %q", key)
		}
	}
	return nil
}

// CheckPendingDigestNoAutoAction verifies that a digest log does not contain
// auto_executed actions (all digest actions must be explicitly triggered).
func CheckPendingDigestNoAutoAction(digestLogJSON string) error {
	if strings.Contains(digestLogJSON, `"auto_executed":true`) {
		return errors.New("specPatchChecker[v3.2.0]: pending digest must not auto-execute any action; all actions require explicit user confirmation")
	}
	return nil
}

// --- v3.3.2 specific guards ---

// CheckPackageNotInstalledBeforeConfirm verifies that a package import record
// does not show tool_registry writes while still in "quarantined" status.
func CheckPackageNotInstalledBeforeConfirm(importRecordJSON string) error {
	if strings.Contains(importRecordJSON, `"status":"quarantined"`) &&
		strings.Contains(importRecordJSON, "tool_registry_written") {
		return errors.New("specPatchChecker[v3.3.2]: package must not write tool_registry before user confirms installation")
	}
	return nil
}

// CheckNoAddToActiveToolsButton verifies that UI markup does not contain the
// forbidden "加入使用工具" text or equivalent English forms.
func CheckNoAddToActiveToolsButton(uiMarkup string) error {
	forbidden := []string{"加入使用工具", "add to active tools", "add to tools", "enable tool"}
	for _, f := range forbidden {
		if strings.Contains(strings.ToLower(uiMarkup), strings.ToLower(f)) {
			return fmt.Errorf("specPatchChecker[v3.3.2]: UI must not contain %q button or text (tools are always visible once installed)", f)
		}
	}
	return nil
}

// CheckRestoreDefaultsDoesNotDeleteNonUI verifies that a restore-defaults
// operation record does not touch non-UI data stores.
func CheckRestoreDefaultsDoesNotDeleteNonUI(operationLogJSON string) error {
	forbidden := []string{
		"memory_deleted", "dag_deleted", "installed_tools_removed",
		"tool_registry_cleared", "review_log_deleted",
	}
	for _, key := range forbidden {
		if strings.Contains(operationLogJSON, key) {
			return fmt.Errorf("specPatchChecker[v3.3.2]: restore defaults must not delete non-UI data: found %q", key)
		}
	}
	return nil
}

// CheckDocumentationLinkNotInRoutingArea verifies that a registered link of
// type "documentation" does not appear in the routing or execution area records.
func CheckDocumentationLinkNotInRoutingArea(routingRecordJSON string) error {
	// The routing record should not contain any link_type=documentation entries.
	if strings.Contains(routingRecordJSON, `"link_type":"documentation"`) {
		return errors.New("specPatchChecker[v3.3.2]: documentation links must not appear in routing / execution area")
	}
	return nil
}

// CheckNoEngineeringTokenInUserText verifies that UI visible text does not
// expose internal state token engineering names.
func CheckNoEngineeringTokenInUserText(visibleText string) error {
	forbidden := []string{
		"passive_white", "pink_cut_black", "forgiveness_green", "defeat_blue",
	}
	for _, token := range forbidden {
		if strings.Contains(visibleText, token) {
			return fmt.Errorf("specPatchChecker[v3.3.2]: engineering token %q must not appear in user-visible text", token)
		}
	}
	return nil
}

// --- Skill Context Orchestration guards (Appendix v1.1) ---

// CheckSkillInjectionNotPersistentAcrossActions 確認 injection audit 記錄
// 包含 cleared_at 與 clear_reason，表示注入在下個不相關動作時被正確清除。
// 若 audit 顯示同一 session 有多筆 active（未 cleared）注入，回傳錯誤。
func CheckSkillInjectionNotPersistentAcrossActions(auditJSON string) error {
	// 多筆 active injection 代表 skill context 未被清除
	if strings.Count(auditJSON, `"clear_reason":""`) > 1 ||
		(strings.Count(auditJSON, `"cleared_at":null`) > 1 &&
			!strings.Contains(auditJSON, `"cleared_at":"20`)) {
		return errors.New("specPatchChecker[skill]: skill injection must not persist across unrelated actions; only one active injection per session")
	}
	return nil
}

// CheckSkillManifestNotModifiedByCLI 確認 CLI 輸出不包含嘗試修改 manifest
// 或 relation 檔案的指令。
func CheckSkillManifestNotModifiedByCLI(cliOutputJSON string) error {
	forbidden := []string{
		"skill_manifest.json",
		".skill_rel.json",
		"manifest_modification",
		"relation_modification",
		"alias_modification",
	}
	for _, key := range forbidden {
		if strings.Contains(cliOutputJSON, key) {
			return fmt.Errorf("specPatchChecker[skill]: CLI output must not modify skill manifest or relation files: found %q", key)
		}
	}
	return nil
}

// CheckSkillInjectionNoRawOutput 確認 audit log 不包含原始 CLI 輸出、
// token 或認證快取等敏感資料。
func CheckSkillInjectionNoRawOutput(auditJSON string) error {
	forbidden := []string{
		"raw_cli_output", "api_key", "auth_cache",
		"access_token", "refresh_token", "password",
	}
	for _, key := range forbidden {
		if strings.Contains(auditJSON, key) {
			return fmt.Errorf("specPatchChecker[skill]: skill audit log must not contain %q", key)
		}
	}
	return nil
}

// CheckHighRiskSkillRequiresReview 確認高風險 skill 的 resolve 結果
// 必須為 needs_user_review，不可 auto_selected。
func CheckHighRiskSkillRequiresReview(resolveJSON string) error {
	if strings.Contains(resolveJSON, `"risk":"high"`) &&
		strings.Contains(resolveJSON, `"status":"auto_selected"`) {
		return errors.New("specPatchChecker[skill]: high-risk skill must not be auto_selected; must require user review")
	}
	if strings.Contains(resolveJSON, `"risk":"critical"`) &&
		strings.Contains(resolveJSON, `"status":"auto_selected"`) {
		return errors.New("specPatchChecker[skill]: critical-risk skill must not be auto_selected; must require user review")
	}
	return nil
}

// CheckSkillContextNotExposingAbsolutePaths 確認注入的 resource_refs
// 不包含絕對路徑，只允許相對路徑或 ID。
func CheckSkillContextNotExposingAbsolutePaths(injectionJSON string) error {
	// 偵測常見絕對路徑前綴
	pathPrefixes := []string{`"/Users/`, `"/home/`, `"C:\\`, `"/var/`, `"/tmp/`}
	for _, prefix := range pathPrefixes {
		if strings.Contains(injectionJSON, prefix) {
			return fmt.Errorf("specPatchChecker[skill]: skill injection must not expose absolute paths: found %s", prefix)
		}
	}
	return nil
}

// ══════════════════════════════════════════════
// v3.3.2 補強 guard（#48 — TASKS_1_6.md 合約補強）
// ══════════════════════════════════════════════

// CheckQuarantinePackageNotInRouting 禁止未確認 package 進入 routing candidate set。
func CheckQuarantinePackageNotInRouting(routingCandidateJSON string) error {
	if strings.Contains(routingCandidateJSON, `"status":"quarantined"`) {
		return errors.New("specPatchChecker[v3.3.2]: quarantined package must not appear in routing candidate set")
	}
	return nil
}

// CheckQuarantinePackageNotInRegistry 禁止未確認 package 寫入任何 registry。
func CheckQuarantinePackageNotInRegistry(registryJSON string) error {
	if strings.Contains(registryJSON, `"status":"quarantined"`) {
		return errors.New("specPatchChecker[v3.3.2]: quarantined package must not be written to tool/skill/MCP registry")
	}
	return nil
}

// CheckQuarantineNoExecutableEntry 禁止 quarantine package 產生可執行入口。
func CheckQuarantineNoExecutableEntry(quarantineRecordJSON string) error {
	forbidden := []string{"executable_entry_created", "entry_point_activated", "tool_registered"}
	for _, key := range forbidden {
		if strings.Contains(quarantineRecordJSON, key) {
			return fmt.Errorf("specPatchChecker[v3.3.2]: quarantine package must not produce executable entry: found %q", key)
		}
	}
	return nil
}

// CheckOrphanQuarantineNotDirectReviewCard 禁止孤兒 quarantine 項目
// 未經安全檢查就直接產生 Review Card。
func CheckOrphanQuarantineNotDirectReviewCard(orphanScanJSON string) error {
	// 孤兒項目必須有 security_check 記錄，且 passed 必須存在
	if strings.Contains(orphanScanJSON, `"orphan_id"`) &&
		!strings.Contains(orphanScanJSON, `"security_check"`) {
		return errors.New("specPatchChecker[v3.3.2]: orphan quarantine item must pass security check before generating Review Card")
	}
	return nil
}

// CheckPermissionMatchesActualContent 禁止權限宣告與實際 entry point / write target 不一致。
func CheckPermissionMatchesActualContent(importRecordJSON string) error {
	if strings.Contains(importRecordJSON, "undeclared_entry_point") ||
		strings.Contains(importRecordJSON, "undeclared_write_target") {
		return errors.New("specPatchChecker[v3.3.2]: declared permissions must match actual entry points and write targets")
	}
	return nil
}

// CheckSecurityCheckFailureNoRegistrySideEffect 禁止安全檢查失敗後保留 registry side effect。
func CheckSecurityCheckFailureNoRegistrySideEffect(failureRecordJSON string) error {
	if strings.Contains(failureRecordJSON, `"security_check_passed":false`) &&
		(strings.Contains(failureRecordJSON, "registry_written") ||
			strings.Contains(failureRecordJSON, "routing_modified") ||
			strings.Contains(failureRecordJSON, "tool_activated")) {
		return errors.New("specPatchChecker[v3.3.2]: failed security check must not leave registry side effects")
	}
	return nil
}

// CheckMCPServerLocalhostNotBlockedInRelease 禁止 release mode 阻擋
// MCP server 的 localhost registration（MCP 走獨立 quarantine 路徑）。
func CheckMCPServerLocalhostNotBlockedInRelease(linkRegistryJSON string) error {
	// MCP server 類型的 localhost 不應出現 "blocked_by_release_mode" 標記
	if strings.Contains(linkRegistryJSON, `"link_type":"mcp_server"`) &&
		strings.Contains(linkRegistryJSON, "blocked_by_release_mode") {
		return errors.New("specPatchChecker[v3.3.2]: MCP server localhost must not be blocked by release mode HTTPS restriction")
	}
	return nil
}

// CheckDisconnectOrderCacheNotInAuditLog 確認斷線排序暫存紀錄
// 不寫入 append-only audit log（屬 UI 暫態）。
func CheckDisconnectOrderCacheNotInAuditLog(auditLogJSON string) error {
	if strings.Contains(auditLogJSON, "disconnect_order_cache") {
		return errors.New("specPatchChecker[v3.3.2]: disconnect order cache is UI transient state and must not appear in audit log")
	}
	return nil
}

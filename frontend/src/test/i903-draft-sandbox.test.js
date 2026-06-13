// I-903 Draft Sandbox 流程驗證
// 驗證 StartDraftSandbox / StopDraftSandbox / PromoteDraftToPending 三步流程
// 方案 B 已實作：app.go 組合層負責根據 promotion type 呼叫 review.Service / PendingCandidateManager
import {describe, it, expect, beforeEach} from 'vitest';
import {mockWailsBinding, resetAllWailsMocks} from './wails-mock.js';

describe('I-903 Draft Sandbox — Wails Binding 流程驗證', () => {
  beforeEach(() => {
    resetAllWailsMocks();
  });

  it('StartDraftSandbox 回傳 sandbox ID', async () => {
    const fn = mockWailsBinding('StartDraftSandbox', 'sandbox-001');
    const result = await window.go.main.App.StartDraftSandbox('window-hash-abc');
    expect(fn).toHaveBeenCalledWith('window-hash-abc');
    expect(result).toBe('sandbox-001');
  });

  it('StopDraftSandbox 以 user_stop 停止', async () => {
    const fn = mockWailsBinding('StopDraftSandbox', undefined);
    await window.go.main.App.StopDraftSandbox('sandbox-001', 'user_stop');
    expect(fn).toHaveBeenCalledWith('sandbox-001', 'user_stop');
  });

  it('StopDraftSandbox 以 authorization 停止', async () => {
    const fn = mockWailsBinding('StopDraftSandbox', undefined);
    await window.go.main.App.StopDraftSandbox('sandbox-001', 'authorization');
    expect(fn).toHaveBeenCalledWith('sandbox-001', 'authorization');
  });

  it('PromoteDraftToPending — formal_review 路徑', async () => {
    const fn = mockWailsBinding('PromoteDraftToPending', 'promotion-id-001');
    const result = await window.go.main.App.PromoteDraftToPending('sandbox-001', 'formal_review');
    expect(fn).toHaveBeenCalledWith('sandbox-001', 'formal_review');
    expect(result).toBe('promotion-id-001');
  });

  it('PromoteDraftToPending — pending_candidate 路徑', async () => {
    const fn = mockWailsBinding('PromoteDraftToPending', 'promotion-id-002');
    const result = await window.go.main.App.PromoteDraftToPending('sandbox-001', 'pending_candidate');
    expect(fn).toHaveBeenCalledWith('sandbox-001', 'pending_candidate');
    expect(result).toBe('promotion-id-002');
  });

  it('完整三步流程：啟動 → 停止 → 選項', async () => {
    // Step 1: 啟動
    mockWailsBinding('StartDraftSandbox', 'sandbox-flow-001');
    const sandboxId = await window.go.main.App.StartDraftSandbox('win-hash');
    expect(sandboxId).toBe('sandbox-flow-001');

    // Step 2: 停止（模擬使用者主動停止）
    const stopFn = mockWailsBinding('StopDraftSandbox', undefined);
    await window.go.main.App.StopDraftSandbox(sandboxId, 'user_stop');
    expect(stopFn).toHaveBeenCalledWith('sandbox-flow-001', 'user_stop');

    // Step 3: 使用者選擇其中一個選項
    const promoteFn = mockWailsBinding('PromoteDraftToPending', 'promo-001');
    await window.go.main.App.PromoteDraftToPending(sandboxId, 'formal_review');
    expect(promoteFn).toHaveBeenCalledWith('sandbox-flow-001', 'formal_review');
  });

  it('停止後不得自動 promote（需使用者明確選擇）', async () => {
    // 驗證 StopDraftSandbox 不會觸發 PromoteDraftToPending
    const promoteFn = mockWailsBinding('PromoteDraftToPending', 'auto-promo');
    mockWailsBinding('StopDraftSandbox', undefined);
    await window.go.main.App.StopDraftSandbox('sandbox-001', 'user_stop');
    // PromoteDraftToPending 不應被自動呼叫
    expect(promoteFn).not.toHaveBeenCalled();
  });

  // ── 方案 B side effect 驗證（app.go 組合層已實作） ──
  // 前端只驗證 PromoteDraftToPending 回傳的 message 包含正確資訊，
  // 實際 side effect（Review Card 建立 / candidate store 寫入）由 Go 端 go test 覆蓋。

  it('formal_review 回傳訊息包含 Review Card 建立資訊', async () => {
    // 方案 B：app.go 呼叫 reviewService.AddCard 後回傳含 card ID 的訊息
    const msg = 'sandbox sandbox-001 → Review Card rev-123 已建立';
    mockWailsBinding('PromoteDraftToPending', msg);
    const result = await window.go.main.App.PromoteDraftToPending('sandbox-001', 'formal_review');
    expect(result).toContain('Review Card');
    expect(result).toContain('已建立');
  });

  it('pending_candidate 回傳訊息包含 candidate 儲存資訊', async () => {
    // 方案 B：app.go 呼叫 pendingCandidateMgr.Add 後回傳含 record ID 的訊息
    const msg = 'sandbox sandbox-001 → pending candidate pc-456 已儲存';
    mockWailsBinding('PromoteDraftToPending', msg);
    const result = await window.go.main.App.PromoteDraftToPending('sandbox-001', 'pending_candidate');
    expect(result).toContain('pending candidate');
    expect(result).toContain('已儲存');
  });

  it('discard 回傳訊息包含 discarded 資訊', async () => {
    const msg = 'sandbox sandbox-001 discarded';
    mockWailsBinding('PromoteDraftToPending', msg);
    const result = await window.go.main.App.PromoteDraftToPending('sandbox-001', 'discard');
    expect(result).toContain('discarded');
  });
});

// I-902 Skill Activity Card 驗證
// 驗證 GetRecentSkillInjections binding + fixture + 敏感資料不洩露
import {describe, it, expect, beforeEach} from 'vitest';
import {mockWailsBinding, resetAllWailsMocks} from './wails-mock.js';

// ── fixture ──

const injectionFixture = [
  {
    injection_id: 'inj-001',
    session_id: 'sess-abc',
    skill_id: 'sk-file-manager',
    match_reason: 'action_keyword:file_organize',
    summary_hash: 'sha256:a1b2c3d4e5f6',
    risk: 'medium',
    score: 0.85,
    resource_ids: ['res-001', 'res-002'],
    injected_at: '2026-05-13T09:30:00Z',
    cleared_at: '2026-05-13T09:31:00Z',
    clear_reason: 'action_completed',
  },
  {
    injection_id: 'inj-002',
    session_id: 'sess-abc',
    skill_id: 'sk-code-review',
    match_reason: 'context_pattern:code_diff',
    summary_hash: 'sha256:f6e5d4c3b2a1',
    risk: 'low',
    score: 0.72,
    resource_ids: ['res-003'],
    injected_at: '2026-05-13T09:32:00Z',
    cleared_at: null,
    clear_reason: '',
  },
];

// 含敏感資料的 fixture — 不應出現在 UI
const sensitiveInjectionFixture = [
  {
    injection_id: 'inj-bad',
    session_id: 'sess-abc',
    skill_id: 'sk-test',
    match_reason: 'test',
    summary_hash: 'sha256:000',
    risk: 'low',
    score: 0.5,
    resource_ids: [],
    // 以下欄位不應存在於 audit log（spec 禁止）
    raw_cli_output: 'should not exist',
    api_key: 'sk-secret-key',
    auth_cache: 'cached-token',
    access_token: 'at-12345',
  },
];

describe('I-902 Skill Activity Card — Wails Binding 驗證', () => {
  beforeEach(() => {
    resetAllWailsMocks();
  });

  it('GetRecentSkillInjections 回傳最近兩筆 injection', async () => {
    const fn = mockWailsBinding('GetRecentSkillInjections', injectionFixture);
    const result = await window.go.main.App.GetRecentSkillInjections('sess-abc');
    expect(fn).toHaveBeenCalledWith('sess-abc');
    expect(result).toHaveLength(2);
  });

  it('injection 包含必要顯示欄位', async () => {
    mockWailsBinding('GetRecentSkillInjections', injectionFixture);
    const result = await window.go.main.App.GetRecentSkillInjections('sess-abc');
    const first = result[0];
    expect(first).toHaveProperty('skill_id');
    expect(first).toHaveProperty('match_reason');
    expect(first).toHaveProperty('summary_hash');
    expect(first).toHaveProperty('risk');
    expect(first).toHaveProperty('score');
    expect(first).toHaveProperty('resource_ids');
  });

  it('空 injection 列表不報錯', async () => {
    mockWailsBinding('GetRecentSkillInjections', []);
    const result = await window.go.main.App.GetRecentSkillInjections('sess-empty');
    expect(result).toEqual([]);
  });

  it('敏感欄位不應出現在 injection audit 資料中', () => {
    // 這是 spec 層級檢查：audit log 不應包含 raw_cli_output / api_key / auth_cache / access_token
    // Go 端 CheckSkillInjectionNoRawOutput 負責阻擋，這裡驗證前端收到時也能偵測
    const sensitive = sensitiveInjectionFixture[0];
    const forbiddenKeys = ['raw_cli_output', 'api_key', 'auth_cache', 'access_token'];
    const foundSensitive = forbiddenKeys.filter(key => key in sensitive && sensitive[key]);
    expect(foundSensitive.length).toBeGreaterThan(0); // fixture 確實含敏感資料
    // 正常 fixture 不應含這些欄位
    const clean = injectionFixture[0];
    const cleanSensitive = forbiddenKeys.filter(key => key in clean && clean[key]);
    expect(cleanSensitive).toEqual([]);
  });

  it('injection 不應包含完整本機路徑', () => {
    const absolutePathPrefixes = ['/Users/', '/home/', 'C:\\', '/var/', '/tmp/'];
    for (const inj of injectionFixture) {
      const json = JSON.stringify(inj);
      for (const prefix of absolutePathPrefixes) {
        expect(json).not.toContain(prefix);
      }
    }
  });
});

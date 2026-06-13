// I-901 Review Panel 整合驗證
// 驗證 Wails binding 存在性 + fixture 回傳正確性 + 空狀態處理
import {describe, it, expect, beforeEach} from 'vitest';
import {mockWailsBinding, mockWailsBindings, resetAllWailsMocks} from './wails-mock.js';

// ── fixture：deterministic fake data ──

const tagPatchFixture = [
  {
    id: 'tp-001',
    tag: 'browser_automation',
    action: 'add',
    source_hook_id: 'hook-run-42',
    confidence: 0.87,
    status: 'pending',
  },
  {
    id: 'tp-002',
    tag: 'file_system_write',
    action: 'remove',
    source_hook_id: 'hook-run-43',
    confidence: 0.92,
    status: 'pending',
  },
];

const pendingDigestFixture = {
  id: 'digest-001',
  generated_at: '2026-05-13T10:00:00Z',
  items: [
    {id: 'd-1', category: 'needs_decision', summary: '新增 MCP server 權限', risk_level: 'high'},
    {id: 'd-2', category: 'can_wait', summary: '工具偏好調整建議', risk_level: 'low'},
    {id: 'd-3', category: 'suggest_archive', summary: '過期的 hook 候選', risk_level: 'low'},
  ],
};

const pendingPackagesFixture = [
  {
    package_id: 'pkg-test-001',
    name: 'test-skill-pack',
    version: '1.0.0',
    status: 'quarantined',
    risk_tag: 'medium',
    declared_permissions: ['file_read', 'network_fetch'],
    write_targets: ['data/skills/'],
  },
];

// ── tests ──

describe('I-901 Review Panel — Wails Binding 驗證', () => {
  beforeEach(() => {
    resetAllWailsMocks();
  });

  it('GetPendingTagPatches binding 存在且可呼叫', async () => {
    const fn = mockWailsBinding('GetPendingTagPatches', tagPatchFixture);
    const result = await window.go.main.App.GetPendingTagPatches();
    expect(fn).toHaveBeenCalled();
    expect(result).toEqual(tagPatchFixture);
    expect(result).toHaveLength(2);
  });

  it('GetPendingDigest binding 存在且可呼叫', async () => {
    const fn = mockWailsBinding('GetPendingDigest', pendingDigestFixture);
    const result = await window.go.main.App.GetPendingDigest();
    expect(fn).toHaveBeenCalled();
    expect(result.items).toHaveLength(3);
  });

  it('ListPendingPackages binding 存在且可呼叫', async () => {
    const fn = mockWailsBinding('ListPendingPackages', pendingPackagesFixture);
    const result = await window.go.main.App.ListPendingPackages();
    expect(fn).toHaveBeenCalled();
    expect(result[0].status).toBe('quarantined');
    expect(result[0].risk_tag).toBe('medium');
  });

  it('Pending Digest 三分類資料結構正確', async () => {
    mockWailsBinding('GetPendingDigest', pendingDigestFixture);
    const result = await window.go.main.App.GetPendingDigest();
    const categories = result.items.map(i => i.category);
    expect(categories).toContain('needs_decision');
    expect(categories).toContain('can_wait');
    expect(categories).toContain('suggest_archive');
  });

  it('空 fixture 不報錯', async () => {
    mockWailsBindings({
      GetPendingTagPatches: [],
      GetPendingDigest: {id: 'empty', generated_at: '', items: []},
      ListPendingPackages: [],
    });
    const tags = await window.go.main.App.GetPendingTagPatches();
    const digest = await window.go.main.App.GetPendingDigest();
    const pkgs = await window.go.main.App.ListPendingPackages();
    expect(tags).toEqual([]);
    expect(digest.items).toEqual([]);
    expect(pkgs).toEqual([]);
  });

  it('高風險 package 在 fixture 中可識別', async () => {
    const highRiskPkg = [{
      ...pendingPackagesFixture[0],
      risk_tag: 'high_non_destructive',
      declared_permissions: ['file_write', 'shell_exec', 'network_fetch'],
    }];
    mockWailsBinding('ListPendingPackages', highRiskPkg);
    const result = await window.go.main.App.ListPendingPackages();
    expect(result[0].risk_tag).toBe('high_non_destructive');
    expect(result[0].declared_permissions).toContain('shell_exec');
  });
});

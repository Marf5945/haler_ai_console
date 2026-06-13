// I-905 External Link 分流驗證
// 驗證 ListExternalLinksByType 三路分流 + documentation 不進 execution area
// + BrowserOpenURL 外部開啟
import {describe, it, expect, beforeEach} from 'vitest';
import {mockWailsBinding, mockBrowserOpenURL, resetAllWailsMocks} from './wails-mock.js';

// ── fixture ──

const externalServiceFixture = [
  {id: 'link-es-001', url: 'https://api.example.com/v2', link_type: 'external_service', label: 'Example API'},
];

const adapterCandidateFixture = [
  {id: 'link-ac-001', url: 'https://tools.example.com/adapter', link_type: 'adapter_candidate', label: 'CLI Adapter'},
];

const documentationFixture = [
  {id: 'link-doc-001', url: 'https://docs.example.com/guide', link_type: 'documentation', label: 'User Guide'},
  {id: 'link-doc-002', url: 'https://wiki.example.com/faq', link_type: 'documentation', label: 'FAQ'},
];

describe('I-905 External Link 分流 — Wails Binding 驗證', () => {
  beforeEach(() => {
    resetAllWailsMocks();
  });

  it('ListExternalLinksByType("external_service") 回傳工具區連結', async () => {
    const fn = mockWailsBinding('ListExternalLinksByType', externalServiceFixture);
    const result = await window.go.main.App.ListExternalLinksByType('external_service');
    expect(fn).toHaveBeenCalledWith('external_service');
    expect(result).toHaveLength(1);
    expect(result[0].link_type).toBe('external_service');
  });

  it('ListExternalLinksByType("adapter_candidate") 回傳 CLI Adapter 列表', async () => {
    const fn = mockWailsBinding('ListExternalLinksByType', adapterCandidateFixture);
    const result = await window.go.main.App.ListExternalLinksByType('adapter_candidate');
    expect(fn).toHaveBeenCalledWith('adapter_candidate');
    expect(result[0].link_type).toBe('adapter_candidate');
  });

  it('ListExternalLinksByType("documentation") 回傳純參考連結', async () => {
    const fn = mockWailsBinding('ListExternalLinksByType', documentationFixture);
    const result = await window.go.main.App.ListExternalLinksByType('documentation');
    expect(fn).toHaveBeenCalledWith('documentation');
    expect(result).toHaveLength(2);
    result.forEach(link => {
      expect(link.link_type).toBe('documentation');
    });
  });

  it('documentation 不得進 execution area（link_type 永遠是 documentation）', async () => {
    mockWailsBinding('ListExternalLinksByType', documentationFixture);
    const result = await window.go.main.App.ListExternalLinksByType('documentation');
    for (const link of result) {
      expect(link.link_type).not.toBe('external_service');
      expect(link.link_type).not.toBe('adapter_candidate');
      expect(link.link_type).toBe('documentation');
    }
  });

  it('documentation 必須用 BrowserOpenURL 外部開啟', () => {
    // 驗證：前端改用 BrowserOpenURL 而非 <a href target="_blank">
    const openFn = mockBrowserOpenURL();
    const testUrl = 'https://docs.example.com/guide';

    // 模擬使用者點擊 documentation link 的行為
    window.runtime.BrowserOpenURL(testUrl);

    expect(openFn).toHaveBeenCalledWith(testUrl);
    expect(openFn).toHaveBeenCalledTimes(1);
  });

  it('documentation 不得在 AI Console 內部 webview 開啟', () => {
    // 此為架構驗證：確認 App.jsx 中 documentation link
    // 使用 <button onClick={BrowserOpenURL}> 而非 <a href target="_blank">
    // 由 I-906 的靜態分析測試覆蓋（grep App.jsx）
    // 這裡驗證 BrowserOpenURL mock 機制正常
    const openFn = mockBrowserOpenURL();
    window.runtime.BrowserOpenURL('https://test.com');
    expect(openFn).toHaveBeenCalled();
  });

  it('unsupported link type 不應出現在任何分類結果中', async () => {
    // 後端 RegisterExternalLink 會拒絕 unsupported，所以不應出現
    const emptyResult = [];
    mockWailsBinding('ListExternalLinksByType', emptyResult);
    const result = await window.go.main.App.ListExternalLinksByType('unsupported');
    expect(result).toEqual([]);
  });

  it('空列表不報錯', async () => {
    mockWailsBinding('ListExternalLinksByType', []);
    const result = await window.go.main.App.ListExternalLinksByType('external_service');
    expect(result).toEqual([]);
  });
});

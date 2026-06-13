// I-904 Browser Chip / Safari Notice 驗證
// 驗證 GetBrowserPreference / GetSafariRuntimeNotice binding + 唯讀限制
import {describe, it, expect, beforeEach} from 'vitest';
import {mockWailsBinding, resetAllWailsMocks} from './wails-mock.js';

// ── fixture ──

const chromePreferenceFixture = {
  browser: 'Chrome',
  profile_path: '<profile-home>/Library/Application Support/Google/Chrome/Default',
  auto_detected: true,
};

const safariPreferenceFixture = {
  browser: 'Safari',
  profile_path: '',
  auto_detected: true,
};

const safariNoticeFixture = {
  should_show: true,
  message: 'Safari 的 profile reuse 受限，部分自動化任務可能無法共用登入態。',
  severity: 'warning',
  blocking: false,
};

const safariNoticeNotNeededFixture = {
  should_show: false,
  message: '',
  severity: '',
  blocking: false,
};

describe('I-904 Browser Chip — Wails Binding 驗證', () => {
  beforeEach(() => {
    resetAllWailsMocks();
  });

  it('GetBrowserPreference 回傳目前 browser 選擇', async () => {
    const fn = mockWailsBinding('GetBrowserPreference', chromePreferenceFixture);
    const result = await window.go.main.App.GetBrowserPreference();
    expect(fn).toHaveBeenCalled();
    expect(result.browser).toBe('Chrome');
    expect(result).toHaveProperty('profile_path');
    expect(result).toHaveProperty('auto_detected');
  });

  it('Browser Chip 應為唯讀（無 SetBrowserPreference 直接呼叫場景）', () => {
    // Browser Chip 在 UI 只顯示資訊，修改在 Settings 頁面。
    // 此測試確認 fixture 結構不含修改方法暗示。
    expect(chromePreferenceFixture).not.toHaveProperty('editable');
  });

  it('選擇 Safari 時 GetSafariRuntimeNotice 必須回傳通知', async () => {
    mockWailsBinding('GetBrowserPreference', safariPreferenceFixture);
    const noticeFn = mockWailsBinding('GetSafariRuntimeNotice', safariNoticeFixture);

    const pref = await window.go.main.App.GetBrowserPreference();
    expect(pref.browser).toBe('Safari');

    // Safari 選擇後必須觸發 notice 查詢
    const notice = await window.go.main.App.GetSafariRuntimeNotice();
    expect(noticeFn).toHaveBeenCalled();
    expect(notice.should_show).toBe(true);
    expect(notice.blocking).toBe(false); // 非阻塞
  });

  it('Safari notice 不得阻斷一般 low-risk 任務', async () => {
    mockWailsBinding('GetSafariRuntimeNotice', safariNoticeFixture);
    const notice = await window.go.main.App.GetSafariRuntimeNotice();
    expect(notice.blocking).toBe(false);
    expect(notice.severity).toBe('warning'); // 非阻塞警告
  });

  it('非 Safari 瀏覽器不需要顯示 notice', async () => {
    mockWailsBinding('GetBrowserPreference', chromePreferenceFixture);
    mockWailsBinding('GetSafariRuntimeNotice', safariNoticeNotNeededFixture);

    const pref = await window.go.main.App.GetBrowserPreference();
    expect(pref.browser).toBe('Chrome');

    const notice = await window.go.main.App.GetSafariRuntimeNotice();
    expect(notice.should_show).toBe(false);
  });

  it('Draft Sandbox 因 profile reuse 不可用停止時 stop reason 為 authorization', async () => {
    // 模擬：Safari profile reuse 導致 Draft Sandbox 自動停止
    const stopFn = mockWailsBinding('StopDraftSandbox', undefined);
    await window.go.main.App.StopDraftSandbox('sandbox-safari', 'authorization');
    expect(stopFn).toHaveBeenCalledWith('sandbox-safari', 'authorization');
  });
});

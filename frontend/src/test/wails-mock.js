// wails-mock.js — 提供 deterministic fixture 注入工具
// 測試中用 mockWailsBinding('GetPendingTagPatches', fixtureData) 來控制回傳值。
//
// 原理：wailsjs/go/main/App.js 的每個 export function 最終呼叫
// window['go']['main']['App'][methodName](args)，
// 所以我們替換 window.go.main.App 上的方法即可。
import {vi} from 'vitest';

/**
 * 設定指定 Wails binding 的回傳值。
 * @param {string} methodName — binding 名稱，如 'GetPendingTagPatches'
 * @param {*} returnValue — 回傳值（會包在 Promise.resolve 裡）
 * @returns {import('vitest').Mock} — vi.fn() mock 物件，可用 .toHaveBeenCalled() 等斷言
 */
export function mockWailsBinding(methodName, returnValue) {
  const fn = vi.fn(() => Promise.resolve(returnValue));
  window.go.main.App[methodName] = fn;
  return fn;
}

/**
 * 批次設定多個 binding。
 * @param {Record<string, *>} bindings — { methodName: returnValue }
 * @returns {Record<string, import('vitest').Mock>}
 */
export function mockWailsBindings(bindings) {
  const mocks = {};
  for (const [name, value] of Object.entries(bindings)) {
    mocks[name] = mockWailsBinding(name, value);
  }
  return mocks;
}

/**
 * 設定 Wails binding 拋出錯誤。
 * @param {string} methodName
 * @param {string|Error} error
 */
export function mockWailsBindingError(methodName, error) {
  const err = typeof error === 'string' ? new Error(error) : error;
  const fn = vi.fn(() => Promise.reject(err));
  window.go.main.App[methodName] = fn;
  return fn;
}

/**
 * Mock BrowserOpenURL（用於驗證 documentation link 外部開啟）
 */
export function mockBrowserOpenURL() {
  const fn = vi.fn();
  window.runtime.BrowserOpenURL = fn;
  return fn;
}

/**
 * 重置所有 Wails binding mock 到預設 no-op。
 */
export function resetAllWailsMocks() {
  // 用普通物件而非 Proxy，這樣 mockWailsBinding 設定的 fn 才能正確覆寫。
  // 未被 mock 的方法呼叫時會是 undefined → 測試中會自然報錯，強迫開發者明確 mock。
  window.go = { main: { App: {} } };
  window.runtime = {
    BrowserOpenURL: vi.fn(),
    EventsOnMultiple: vi.fn(() => () => {}),
    EventsOn: vi.fn(() => () => {}),
    EventsOff: vi.fn(),
    OnFileDrop: vi.fn(),
    OnFileDropOff: vi.fn(),
  };
}

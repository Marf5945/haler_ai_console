// Vitest global setup — 載入 jest-dom 擴充 matchers（toBeInTheDocument 等）
// 並建立 Wails runtime mock，讓測試不依賴真實 Wails 環境。
import '@testing-library/jest-dom';

// ──────────────────────────────────────────────
// Mock: window.go.main.App（Wails binding 入口）
// 所有 wailsjs/go/main/App.js 的 export 最終都呼叫
// window['go']['main']['App'][methodName]()，
// 這裡建立空殼讓 import 不報錯，測試中再用 vi.fn() 覆寫。
// ──────────────────────────────────────────────
if (!window.go) {
  // 用普通空物件，測試中由 mockWailsBinding 注入具體方法。
  // 不使用 Proxy，避免 mock 設定被 Proxy getter 覆蓋。
  window.go = { main: { App: {} } };
}

// ──────────────────────────────────────────────
// Mock: window.runtime（Wails runtime API）
// BrowserOpenURL / EventsOn / OnFileDrop 等
// ──────────────────────────────────────────────
if (!window.runtime) {
  window.runtime = {
    BrowserOpenURL: () => {},
    EventsOnMultiple: () => () => {},
    EventsOn: () => () => {},
    EventsOff: () => {},
    OnFileDrop: () => {},
    OnFileDropOff: () => {},
  };
}

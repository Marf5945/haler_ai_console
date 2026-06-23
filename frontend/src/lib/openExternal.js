import {OpenExternalURL} from '../../wailsjs/go/main/App';
import {BrowserOpenURL} from '../../wailsjs/runtime/runtime';

// SEC-05 2b: 所有外部連結一律走 Go 端 OpenExternalURL 檢查
// （僅 http/https、擋 metadata、loopback 放行，monitor 頁不受影響）。
// catch 兩層：binding 缺失時（測試環境）退回 BrowserOpenURL；正式環境 binding 必存在。
export function openExternal(url) {
  try {
    OpenExternalURL(url).catch((err) => console.error('openExternal blocked:', err));
  } catch {
    BrowserOpenURL(url);
  }
}

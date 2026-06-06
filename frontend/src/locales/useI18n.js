// ============================================================
// i18n 核心模組 — 基於 Zustand，零額外依賴
// 支援語系：zh-TW（預設）、en、ja（尚未建立時自動 fallback）
// ============================================================

import { create } from 'zustand';

// ----------------------------------------------------------
// 靜態匯入翻譯檔（確保被打包工具收錄）
// ----------------------------------------------------------
import zhTW from './zh-TW.json';
import en from './en.json';

// ja.json 佔位：檔案建立後改為 import ja from './ja.json'
const ja = {};

// ----------------------------------------------------------
// 語系對照表
// ----------------------------------------------------------
const TRANSLATION_MAP = {
  'zh-TW': zhTW,
  en,
  ja,
};

// ----------------------------------------------------------
// RTL 語系清單（保留給未來阿拉伯語等右至左語言使用）
// ----------------------------------------------------------
const RTL_LANGUAGES = ['ar', 'he', 'fa', 'ur'];

// ----------------------------------------------------------
// 工具函式：從 localStorage 讀取目前語系，預設 zh-TW
// ----------------------------------------------------------
export function getCurrentLanguage() {
  try {
    return localStorage.getItem('i18n_language') || 'zh-TW';
  } catch {
    return 'zh-TW';
  }
}

// ----------------------------------------------------------
// 工具函式：解析點記法路徑，如 'greeting.pool.0'
// ----------------------------------------------------------
function resolveDotKey(obj, key) {
  if (!obj || typeof obj !== 'object') return undefined;
  return key.split('.').reduce((acc, segment) => {
    if (acc === undefined || acc === null) return undefined;
    return acc[segment];
  }, obj);
}

// ----------------------------------------------------------
// 工具函式：將 {variable} 佔位符替換為 params 中對應的值
// ----------------------------------------------------------
function interpolate(template, params) {
  if (!params || typeof template !== 'string') return template;
  return template.replace(/\{(\w+)\}/g, (_, key) =>
    Object.prototype.hasOwnProperty.call(params, key) ? params[key] : `{${key}}`
  );
}

// ----------------------------------------------------------
// 核心翻譯解析（可在 React 元件外部直接使用）
// ----------------------------------------------------------
function resolveTranslation(translations, fallback, language, key, params) {
  /* 先從目前語系查找 */
  let value = resolveDotKey(translations, key);

  /* 找不到則 fallback 至 zh-TW */
  if (value === undefined) {
    value = resolveDotKey(fallback, key);
    if (value === undefined && import.meta.env.DEV) {
      console.warn(`[i18n] 找不到翻譯 key："${key}"（語系：${language}）`);
    }
  }

  /* 仍找不到則回傳 key 本身，方便除錯 */
  if (value === undefined) return key;

  return interpolate(String(value), params);
}

// ----------------------------------------------------------
// 模組初始化：讀取語系設定
// ----------------------------------------------------------
const _initialLanguage = getCurrentLanguage();
const _initialTranslations = TRANSLATION_MAP[_initialLanguage] ?? {};

function applyDocumentLanguage(lang) {
  if (typeof document === 'undefined') return;
  document.documentElement.lang = lang;
  document.documentElement.dir = RTL_LANGUAGES.includes(lang) ? 'rtl' : 'ltr';
}

// ----------------------------------------------------------
// Zustand Store
// ----------------------------------------------------------
const useI18n = create((set, get) => ({
  language: _initialLanguage,
  translations: _initialTranslations,
  fallback: zhTW,

  /* t(key, params?)：翻譯查找 + 插值 */
  t: (key, params) => {
    const { translations, fallback, language } = get();
    return resolveTranslation(translations, fallback, language, key, params);
  },

  /* setLanguage(lang)：切換語系，不重載 UI */
  setLanguage: (lang) => {
    try { localStorage.setItem('i18n_language', lang); } catch {}
    set({
      language: lang,
      translations: TRANSLATION_MAP[lang] ?? {},
    });
    applyDocumentLanguage(lang);
  },

  /* getDirection()：文字方向（預留 RTL） */
  getDirection: () => {
    return RTL_LANGUAGES.includes(get().language) ? 'rtl' : 'ltr';
  },
}));

// ----------------------------------------------------------
// 獨立 t 函式：React 元件外使用
// ----------------------------------------------------------
export function t(key, params) {
  const { translations, fallback, language } = useI18n.getState();
  return resolveTranslation(translations, fallback, language, key, params);
}

export default useI18n;

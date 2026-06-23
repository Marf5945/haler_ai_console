// fontFaces.js — 動態注入 @font-face
// -----------------------------------------------------------------------------
// 為什麼用 JS 在執行期注入，而不是寫進 .css？
//   字型檔（特別是 CJK）很大，且由 scripts/fetch_fonts.sh 下載到 public/fonts/。
//   若直接在被 Vite 處理的 CSS 裡用 url('./fonts/x.ttf')，缺檔時 build 會直接失敗。
//   改成執行期注入純文字 <style>，Vite 不會做 url 靜態解析；缺檔時瀏覽器只是
//   載入失敗並回退到後備字型，不會中斷建置或啟動。
//
// 檔名為「正規檔名」，必須與 scripts/fetch_fonts.sh 的輸出一致。
// 詳見 FONT_SPEC.md。
// -----------------------------------------------------------------------------

// family 名稱必須與 App.jsx FONT_PRESET_STACKS 內字串一致。
// type: 'var' = 可變字重（font-weight: 100 900）；'static' = 固定字重。
const FONT_FACES = [
  // 基礎覆蓋字型（多個版型共用）
  { family: 'Inter',           file: 'inter.ttf',            type: 'var' },
  { family: 'Noto Sans TC',    file: 'noto-sans-tc.ttf',     type: 'var' },
  { family: 'Noto Sans JP',    file: 'noto-sans-jp.ttf',     type: 'var' },
  { family: 'Noto Serif TC',   file: 'noto-serif-tc.ttf',    type: 'var' },

  // 手寫
  { family: 'Caveat',          file: 'caveat.ttf',           type: 'var' },
  { family: 'Klee One',        file: 'klee-one.ttf',         type: 'static', weight: 400 },

  // 書法 / 毛筆
  { family: 'Dancing Script',  file: 'dancing-script.ttf',   type: 'var' },
  { family: 'LXGW WenKai',     file: 'lxgw-wenkai.ttf',      type: 'static', weight: 400 },
  { family: 'Yuji Syuku',      file: 'yuji-syuku.ttf',       type: 'static', weight: 400 },

  // 圓潤 / 普普
  { family: 'Fredoka',         file: 'fredoka.ttf',          type: 'var' },
  { family: 'jf-openhuninn',   file: 'jf-openhuninn.ttf',    type: 'static', weight: 400 },

  // 等寬
  { family: 'JetBrains Mono',  file: 'jetbrains-mono.ttf',   type: 'var' },
];

// base 設定為 './'（見 vite.config.js），故相對於文件 base 的 'fonts/x.ttf'
// 會解析到 dist/fonts/x.ttf（public/fonts 由 vite build 複製）。
const FONT_DIR = 'fonts/';

function faceRule({ family, file, type, weight }) {
  const src = `url('${FONT_DIR}${file}') format('truetype')`;
  const weightLine = type === 'var' ? 'font-weight: 100 900;' : `font-weight: ${weight || 400};`;
  return [
    '@font-face {',
    `  font-family: '${family}';`,
    `  src: ${src};`,
    `  ${weightLine}`,
    '  font-style: normal;',
    '  font-display: swap;',
    '}',
  ].join('\n');
}

let injected = false;

export function injectFontFaces() {
  if (injected || typeof document === 'undefined') return;
  injected = true;
  const css = FONT_FACES.map(faceRule).join('\n\n');
  const style = document.createElement('style');
  style.id = 'dynamic-font-faces';
  style.textContent = css;
  document.head.appendChild(style);
}

export default injectFontFaces;

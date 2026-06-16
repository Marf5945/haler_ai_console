// XSS guard — 鎖定「前端不得引入原始 HTML 注入點」。
// agent 會收集網路資料（檔名/網頁標題/OCR/metadata/LLM 摘要/skill 描述等），一律以
// React {} / textContent 顯示。若未來有人加了 innerHTML 類 sink，這個測試會直接失敗。
import {describe, it, expect} from 'vitest';
import * as fs from 'node:fs';
import * as path from 'node:path';

const SRC = path.resolve(__dirname, '..');
const FORBIDDEN = [
  /dangerouslySetInnerHTML/,
  /\.innerHTML\s*=/,
  /\.outerHTML\s*=/,
  /insertAdjacentHTML/,
  /document\.write\s*\(/,
];

function listSourceFiles(dir) {
  const out = [];
  for (const name of fs.readdirSync(dir)) {
    if (name === 'node_modules' || name === 'test') continue; // 排除依賴與測試自身
    const full = path.join(dir, name);
    const st = fs.statSync(full);
    if (st.isDirectory()) out.push(...listSourceFiles(full));
    else if (/\.(jsx?|tsx?)$/.test(name)) out.push(full);
  }
  return out;
}

describe('XSS guard: 前端 src 無原始 HTML 注入點', () => {
  const files = listSourceFiles(SRC);

  it('掃描到前端原始碼檔案', () => {
    expect(files.length).toBeGreaterThan(0);
  });

  it('沒有任何檔案使用 innerHTML / dangerouslySetInnerHTML / document.write 等 sink', () => {
    const offenders = [];
    for (const f of files) {
      const code = fs.readFileSync(f, 'utf-8');
      for (const re of FORBIDDEN) {
        if (re.test(code)) offenders.push(`${f.replace(SRC, 'src')} :: ${re}`);
      }
    }
    expect(offenders).toEqual([]);
  });
});

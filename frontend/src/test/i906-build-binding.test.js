// I-906 Build / Wails Binding 驗證
// 驗證所有 I-901–I-905 涉及的 Wails binding 都存在於 App.js stub 中，
// 且前端 import 名稱與 stub 一致。
//
// 注意：npm run build 通過與否需在 CI 或本機驗證，此處做靜態 binding 存在性檢查。
import {describe, it, expect, beforeEach} from 'vitest';
import {resetAllWailsMocks} from './wails-mock.js';
import * as fs from 'node:fs';
import * as path from 'node:path';

// I-901–I-905 涉及的所有 Wails binding 名稱
const REQUIRED_BINDINGS = [
  // I-901 Review Panel
  'GetPendingTagPatches',
  'GetPendingDigest',
  'ListPendingPackages',
  // I-902 Skill Activity Card
  'GetRecentSkillInjections',
  // I-903 Draft Sandbox
  'StartDraftSandbox',
  'StopDraftSandbox',
  'PromoteDraftToPending',
  // I-904 Browser Chip
  'GetBrowserPreference',
  'GetSafariRuntimeNotice',
  // I-905 External Link
  'ListExternalLinksByType',
  'PreviewExternalLink',
  'RegisterExternalLink',
  'ListLLMProviderWhitelist',
  'RegisterLLMAPIAdapter',
  'RenameAdapter',
  // #42 Package Import
  'PreparePackageInstall',
  'ConfirmPackageInstall',
  'RejectPackageInstall',
  'RunPackageSecurityCheck',
  'ScanOrphanQuarantine',
  'CleanOrphanQuarantine',
  // #44 Tool Visibility
  'MarkToolUnavailable',
  'MarkToolAvailable',
  'ListTools',
  // #46 Live Preview
  'StartLivePreview',
  'CommitPreview',
  'CancelPreview',
  'IsLivePreviewActive',
  // TASKS_1_7 Review / destructive flow
  'CreateDestructiveReviewCard',
  'GetReviewExecutionContext',
  'ResolveAndExecuteDestructiveReviewCard',
  // debug monitor link register
  'GetMonitorLinks',
];

describe('I-906 Build / Wails Binding 存在性驗證', () => {
  let appJsContent = '';
  let appDtsContent = '';
  let modelsContent = '';

  beforeEach(() => {
    resetAllWailsMocks();
    // 讀取 Wails 生成的 App.js stub
    const appJsPath = path.resolve(__dirname, '../../wailsjs/go/main/App.js');
    const appDtsPath = path.resolve(__dirname, '../../wailsjs/go/main/App.d.ts');
    const modelsPath = path.resolve(__dirname, '../../wailsjs/go/models.ts');
    try {
      appJsContent = fs.readFileSync(appJsPath, 'utf-8');
    } catch {
      appJsContent = '';
    }
    try {
      appDtsContent = fs.readFileSync(appDtsPath, 'utf-8');
    } catch {
      appDtsContent = '';
    }
    try {
      modelsContent = fs.readFileSync(modelsPath, 'utf-8');
    } catch {
      modelsContent = '';
    }
  });

  it('App.js stub 檔案存在且可讀取', () => {
    expect(appJsContent.length).toBeGreaterThan(0);
  });

  // 為每個必要 binding 建立獨立測試
  for (const bindingName of REQUIRED_BINDINGS) {
    it(`Wails binding "${bindingName}" 存在於 App.js stub`, () => {
      expect(appJsContent).toContain(`export function ${bindingName}(`);
    });
  }

  it('所有 binding 在 App.js stub 中有對應 export（靜態驗證）', () => {
    // 靜態驗證：每個必要 binding 在 App.js stub 都有 export function 定義。
    // 不用 runtime 呼叫（mock 空物件上未設定的方法會是 undefined），
    // 改用 App.js 內容字串比對，已由上方逐一測試覆蓋。
    const missing = REQUIRED_BINDINGS.filter(
      name => !appJsContent.includes(`export function ${name}(`)
    );
    expect(missing).toEqual([]);
  });

  it('Wails boundary 不外洩 time.Time 型別', () => {
    // TASKS_1_8: Wails binding generation must not emit "Not found: time.Time".
    // App bindings should return frontendDTO/plain JSON for structs with time fields.
    expect(`${appDtsContent}\n${modelsContent}`).not.toContain('Go type: time');
    expect(`${appDtsContent}\n${modelsContent}`).not.toContain('time.Time');
  });

  it('Document binding 不直接暴露 builtin.DocMeta', () => {
    expect(appDtsContent).toContain('export function ListProjectDocuments():Promise<any>');
    expect(appDtsContent).not.toContain('Promise<builtin.DocMeta');
  });
});

describe('I-906 前端 import 一致性檢查', () => {
  let appJsxContent = '';

  beforeEach(() => {
    const appJsxPath = path.resolve(__dirname, '../App.jsx');
    try {
      appJsxContent = fs.readFileSync(appJsxPath, 'utf-8');
    } catch {
      appJsxContent = '';
    }
  });

  it('App.jsx 存在且可讀取', () => {
    expect(appJsxContent.length).toBeGreaterThan(0);
  });

  // 驗證前端有 import 這些 binding
  const FRONTEND_MUST_IMPORT = [
    'GetPendingTagPatches',
    'GetPendingDigest',
    'ListPendingPackages',
    'GetRecentSkillInjections',
    'StartDraftSandbox',
    'StopDraftSandbox',
    'PromoteDraftToPending',
    'GetBrowserPreference',
    'GetSafariRuntimeNotice',
    'ListExternalLinksByType',
    'ConfirmPackageInstall',
    'RejectPackageInstall',
    'CreateDestructiveReviewCard',
    'ResolveAndExecuteDestructiveReviewCard',
    'GetMonitorLinks',
  ];

  for (const name of FRONTEND_MUST_IMPORT) {
    it(`App.jsx imports "${name}"`, () => {
      expect(appJsxContent).toContain(name);
    });
  }

  it('App.jsx imports BrowserOpenURL from Wails runtime', () => {
    expect(appJsxContent).toContain('BrowserOpenURL');
  });

  it('documentation link 不使用 <a href target="_blank">（應用 BrowserOpenURL）', () => {
    // 找 documentation 相關區塊，確認不含 <a ... target="_blank" 的 doc link
    // 允許其他非 documentation 的 <a target="_blank"> 存在
    const docLinkPattern = /tool-menu-item-doc[\s\S]{0,200}target="_blank"/;
    expect(appJsxContent).not.toMatch(docLinkPattern);
  });
});

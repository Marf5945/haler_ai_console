// 由 App.jsx 拆出的共用 helpers/常數(Phase 1 codemod 自動產生)
import {ExecuteNativeLearningReplayStep, GetMonitorLinks, OpenExternalURL, RenderPixelAvatarPreview} from '../../wailsjs/go/main/App';
import {BrowserOpenURL} from '../../wailsjs/runtime/runtime';
import {t as _t} from '../locales/useI18n';

// SEC-05 2b: 所有外部連結一律走 Go 端 OpenExternalURL 檢查
// （僅 http/https、擋 metadata、loopback 放行，monitor 頁不受影響）。
// catch 兩層：binding 缺失時（測試環境）退回 BrowserOpenURL；正式環境 binding 必存在。
export const openExternal = (url) => {
  try {
    OpenExternalURL(url).catch((err) => console.error('openExternal blocked:', err));
  } catch {
    BrowserOpenURL(url);
  }
};

export const taskProgressDebugEnabled = typeof window !== 'undefined'
  && (new URLSearchParams(window.location.search).has('taskDebug')
    || window.localStorage?.getItem('task_progress_debug') === '1');

export const fallbackState = {
  /* i18n: fallbackState */ greeting: _t('greeting.hello'),
  statusRail: {text: _t('greeting.hello'), layer: 'L1', degraded: true, lockedCount: 0},
  // #I-806: adapter 清單由 Sidecar IPC 傳入，經 adapter_registry 白名單審查後更新
  // 預設空陣列：未接 CLI 時不顯示任何 adapter 按鈕
  adapters: [],
  haoras: ['主haㄌer'],
  messages: [],
};

/* i18n: greeting pool */
export const getGreetingRotationOptions = () => [
  {text: _t('greeting.pool.0'), expression: 'idle'},
  {text: _t('greeting.pool.1'), expression: 'idle'},
  {text: _t('greeting.pool.2'), expression: 'idle'},
  {text: _t('greeting.pool.3'), expression: 'speechless'},
  {text: _t('greeting.pool.4'), expression: 'idle', rare: true},
  {text: _t('greeting.pool.5'), expression: 'speechless'},
  {text: _t('greeting.pool.6'), expression: 'happy'},
  {text: _t('greeting.pool.7'), expression: 'sleepy'},
  {text: _t('greeting.pool.8'), expression: 'idle'},
  {text: _t('greeting.pool.9'), expression: 'idle'},
];

export function pickRotatingGreeting(currentText) {
  const options = getGreetingRotationOptions();
  const regular = options.filter((item) => !item.rare);
  const rare = options.filter((item) => item.rare);
  const basePool = rare.length > 0 && Math.random() < 0.05 ? rare : regular;
  let pool = basePool.filter((item) => item.text !== currentText);
  if (pool.length === 0) {
    pool = regular.filter((item) => item.text !== currentText);
  }
  if (pool.length === 0) {
    pool = getGreetingRotationOptions();
  }
  return pool[Math.floor(Math.random() * pool.length)];
}

// Panel-style identity is a STABLE key (language-independent). Labels are
// resolved live via _t()/styleLabel() so switching language re-translates the
// buttons (and never resets the active selection).
export const STYLE_KEYS = ['default', 'passiveWhite', 'pinkBetrayal', 'forgiveMeGreen', 'defeatBlue'];

export const STYLE_KEY_I18N = {
  default: 'style.default',
  passiveWhite: 'style.options.passiveWhite',
  pinkBetrayal: 'style.options.pinkBetrayal',
  forgiveMeGreen: 'style.options.forgiveMeGreen',
  defeatBlue: 'style.options.defeatBlue',
};

export const STYLE_KEY_THEME = {
  default: 'onanegiku',
  passiveWhite: 'white',
  pinkBetrayal: 'pink-black',
  forgiveMeGreen: 'green',
  defeatBlue: 'blue',
};

export const defaultPanelStyle = 'default';

export const styleOptions = STYLE_KEYS;

// Panel font scale is a UI-wide preference; CSS consumes it through --ui-font-scale.
export const fontScaleOptions = ['50%', '80%', '90%', '100%', '110%', '120%'];

export const styleOptionOrderStorageKey = 'ai-console.style-option-order';

export function floatTo16BitPCM(view, offset, input) {
  for (let i = 0; i < input.length; i += 1, offset += 2) {
    const sample = Math.max(-1, Math.min(1, input[i]));
    view.setInt16(offset, sample < 0 ? sample * 0x8000 : sample * 0x7fff, true);
  }
}

export function writeWavString(view, offset, value) {
  for (let i = 0; i < value.length; i += 1) {
    view.setUint8(offset + i, value.charCodeAt(i));
  }
}

export function encodeWav(chunks, sampleRate) {
  const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
  const samples = new Float32Array(totalLength);
  let offset = 0;
  chunks.forEach((chunk) => {
    samples.set(chunk, offset);
    offset += chunk.length;
  });
  const buffer = new ArrayBuffer(44 + samples.length * 2);
  const view = new DataView(buffer);
  writeWavString(view, 0, 'RIFF');
  view.setUint32(4, 36 + samples.length * 2, true);
  writeWavString(view, 8, 'WAVE');
  writeWavString(view, 12, 'fmt ');
  view.setUint32(16, 16, true);
  view.setUint16(20, 1, true);
  view.setUint16(22, 1, true);
  view.setUint32(24, sampleRate, true);
  view.setUint32(28, sampleRate * 2, true);
  view.setUint16(32, 2, true);
  view.setUint16(34, 16, true);
  writeWavString(view, 36, 'data');
  view.setUint32(40, samples.length * 2, true);
  floatTo16BitPCM(view, 44, samples);
  return new Blob([view], {type: 'audio/wav'});
}

export function blobToBase64(blob) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onloadend = () => resolve(String(reader.result || '').split(',')[1] || '');
    reader.onerror = reject;
    reader.readAsDataURL(blob);
  });
}

export function waitForVoiceChunks(recorder, timeoutMs = 700) {
  if (recorder.chunks.length > 0) return Promise.resolve();
  return new Promise((resolve) => {
    const startedAt = Date.now();
    const timer = setInterval(() => {
      if (recorder.chunks.length > 0 || Date.now() - startedAt >= timeoutMs) {
        clearInterval(timer);
        resolve();
      }
    }, 50);
  });
}

export const MIN_VOICE_RECORDING_MS = 300;

export const MAX_VOICE_RECORDING_MS = 120000;

// 120 秒錄音上限
export const VOICE_BG_THRESHOLD_MS = 30000;

// >30 秒走背景轉錄

export function mergeDraftText(current, addition) {
  const left = String(current || '').trimEnd();
  const right = String(addition || '').trim();
  if (!left) return right;
  if (!right) return left;
  return `${left}\n${right}`;
}

export function adapterField(adapter, camel, pascal, fallback = '') {
  if (!adapter) return fallback;
  const value = adapter[camel] ?? adapter[pascal];
  return value == null ? fallback : value;
}

export function adapterKey(adapter) {
  return adapterField(adapter, 'id', 'ID', '') || adapterField(adapter, 'name', 'Name', '');
}

export function normalizeAdapterDTO(adapter) {
  if (!adapter) return adapter;
  return {
    ...adapter,
    id: adapterField(adapter, 'id', 'ID', ''),
    name: adapterField(adapter, 'name', 'Name', ''),
    icon: adapterField(adapter, 'icon', 'Icon', ''),
    path: adapterField(adapter, 'path', 'Path', ''),
    endpoint: adapterField(adapter, 'endpoint', 'Endpoint', ''),
    model: adapterField(adapter, 'model', 'Model', ''),
    kind: adapterField(adapter, 'kind', 'Kind', ''),
    status: adapterField(adapter, 'status', 'Status', 'offline'),
  };
}

export function isSubagentAdapter(adapter) {
  return String(adapterField(adapter, 'kind', 'Kind', '')).toLowerCase() === 'sub';
}

export function orderSubagentTabsByTabOrder(tabs, tabOrder) {
  if (!Array.isArray(tabs)) return [];
  const rank = new Map((tabOrder?.sub_order || []).map((id, index) => [id, index]));
  return [...tabs].sort((a, b) => {
    const aKey = adapterKey(a);
    const bKey = adapterKey(b);
    const aRank = rank.has(aKey) ? rank.get(aKey) : Number.MAX_SAFE_INTEGER;
    const bRank = rank.has(bKey) ? rank.get(bKey) : Number.MAX_SAFE_INTEGER;
    if (aRank !== bRank) return aRank - bRank;
    return String(adapterField(a, 'name', 'Name', '') || adapterKey(a)).localeCompare(String(adapterField(b, 'name', 'Name', '') || adapterKey(b)), 'zh-Hant');
  });
}

export function splitAvailableAdapters(adapters, tabOrder) {
  const items = Array.isArray(adapters) ? adapters.map(normalizeAdapterDTO) : [];
  return {
    cliAdapters: items.filter((item) => !isSubagentAdapter(item)),
    subagentTabs: orderSubagentTabsByTabOrder(items.filter(isSubagentAdapter), tabOrder),
  };
}

export function reorderItemsByKeys(items, orderKeys) {
  const source = Array.isArray(items) ? items : [];
  const order = Array.isArray(orderKeys) ? orderKeys : [];
  const byKey = new Map(source.map((item) => [adapterKey(item), item]));
  const used = new Set();
  const next = [];
  for (const key of order) {
    if (byKey.has(key) && !used.has(key)) {
      next.push(byKey.get(key));
      used.add(key);
    }
  }
  for (const item of source) {
    const key = adapterKey(item);
    if (!used.has(key)) next.push(item);
  }
  return next;
}

export const learningDigestStorageKey = 'ai-console.learning-digest-ready';

export const learningIdleDelayMs = 20 * 60 * 1000;

export const dagTaskPattern = /(幫我|請|查|搜尋|整理|建立|執行|打開|開啟|分析|寫|寄|下載|安裝|錄製|學習|流程|天氣|地圖)/;

export const dagHighRiskPattern = /(刪除|安裝|寄|付款|修改系統|開啟地圖|地圖|錄製|螢幕)/;

export const internalControlPrefixPattern = /^[ㄅ-ㄩ]{3}\s*/;

export function stripInternalControlPrefix(text) {
  return String(text || '').trim().replace(/^ㄌㄤㄤ/, '').replace(internalControlPrefixPattern, '').trim();
}

export function stripInternalControlDraft(draft) {
  const raw = String(draft || '').trim();
  if (raw.startsWith('input:')) {
    return `input:${stripInternalControlPrefix(raw.slice('input:'.length))}`;
  }
  return stripInternalControlPrefix(raw);
}

// ── v3.6.4 Readiness Gate UI Interaction Layer 常數與 fallback ──
// 長按確認所需時間（毫秒），對應 §11.3 轉蛋動畫觸發門檻
export const LONG_PRESS_DURATION_MS = 800;

// 轉蛋動畫各階段時間（毫秒）
export const GACHA_COLLAPSE_MS = 300;

export const GACHA_PULSE_MS = 500;

export const GACHA_PARTICLE_MS = 600;

// 粒子爆發方向角度（8 個粒子均分 360 度）
export const PARTICLE_ANGLES = Array.from({length: 8}, (_, i) => i * 45);

// 預設 Readiness Gate 狀態（前端 fallback，對應 Go ReadinessGateState 結構）
export const fallbackReadinessGate = {
  risk_tier: 'none',        // none | normal | medium | high
  missing_slots: [],
  floating_candidates: [],
  clarification_count: 0,
  max_clarifications: 2,
  retrieval_scanning: false,
  retrieval_sources: [],
  impact_explanation: '',
  low_confidence_output: false,
  assumption_used: false,
  auto_output_allowed: false,
};

export const fallbackSettings = {
  panel: {
    /* i18n: settings defaults */ panelLanguage: _t('settings.langZhTW'),
    roleLanguage: _t('settings.roleLangAuto'),
    fontPreset: _t('settings.fontDefault'),
    fontScale: '100%',
    panelStyle: defaultPanelStyle,
  },
  activePersonaId: 'persona-a',
  personas: [
    {id: 'persona-a', name: _t('persona.defaultNameA'), icon: '♙', avatarUrl: '', identity: _t('persona.defaultIdentityA'), replyStrategy: '', roleStrength: '20%', personality: '', scenario: '', description: ''},
    {id: 'persona-b', name: _t('persona.defaultNameB'), icon: '♚', avatarUrl: '', identity: _t('persona.defaultIdentityB'), replyStrategy: '', roleStrength: '20%', personality: '', scenario: '', description: ''},
    {id: 'persona-c', name: _t('persona.defaultNameC'), icon: '★', avatarUrl: '', identity: _t('persona.defaultIdentityC'), replyStrategy: '', roleStrength: '20%', personality: '', scenario: '', description: ''},
  ],
};

export const lockedPersonaId = 'persona-a';

/* i18n: persona locked */ export const lockedPersonaName = _t('persona.lockedName');

/* i18n: reply strategy presets */
export const getReplyStrategyPresets = () => [
  {id: 'concise', label: _t('strategy.save.label'), prompt: _t('strategy.save.prompt')},
  {id: 'reflective_question', label: _t('strategy.counter.label'), prompt: _t('strategy.counter.prompt')},
  {id: 'suggestive', label: _t('strategy.suggest.label'), prompt: _t('strategy.suggest.prompt')},
  {id: 'teacher_question', label: _t('strategy.question.label'), prompt: _t('strategy.question.prompt')},
  {id: 'confirm_before_action', label: _t('strategy.execute.label'), prompt: _t('strategy.execute.prompt')},
  {id: 'decision_tree', label: _t('strategy.analyze.label'), prompt: _t('strategy.analyze.prompt')},
  {id: 'companion', label: _t('strategy.companion.label'), prompt: _t('strategy.companion.prompt')},
  {id: 'creative', label: _t('strategy.creative.label'), prompt: _t('strategy.creative.prompt')},
];

export const avatarStateOptions = ['idle', 'thinking', 'working', 'happy', 'warning', 'blocked', 'sleepy', 'sad', 'speechless'];

/* i18n: avatar state labels */
export const getAvatarStateLabels = () => ({
  idle: _t('avatar.idle'),
  thinking: _t('avatar.thinking'),
  working: _t('avatar.working'),
  happy: _t('avatar.happy'),
  warning: _t('avatar.warning'),
  blocked: _t('avatar.blocked'),
  sleepy: _t('avatar.sleepy'),
  sad: _t('avatar.sad'),
  speechless: _t('avatar.speechless'),
});

/* i18n: tool list */
export const getFallbackTools = () => [
  {id: 'tool-entrance', icon: '⌕', title: _t('tool.useTools.title'), detail: _t('tool.useTools.detail'), enabled: true},
  {id: 'reference-link', icon: '▤', title: _t('tool.citeLink.title'), detail: _t('tool.citeLink.detail'), enabled: true},
  {id: 'doc-entrance', icon: '▤', title: _t('tool.citeFile.title'), detail: _t('tool.citeFile.detail'), enabled: true},
  {id: 'external-link', icon: '↗', title: _t('tool.openExternal.title'), detail: _t('tool.openExternal.detail'), enabled: false},
  {id: 'gmail', icon: '✉', title: _t('tool.gmail.title'), detail: _t('tool.gmail.detail'), enabled: false},
  {id: 'flow-mail-digest', icon: '◇', title: _t('tool.summarizeMail.title'), detail: _t('tool.summarizeMail.detail'), enabled: true},
  {id: 'package-import', icon: '□', title: _t('tool.importPack.title'), detail: _t('tool.importPack.detail'), enabled: true},
];

/* i18n: tool tab options */
export const getToolTabOptions = () => [
  {id: 'external', label: _t('tool.tabExternal')},
  {id: 'flow', label: _t('tool.tabAutoflow')},
  {id: 'package', label: _t('tool.tabToolkit')},
];

export const toolTabById = {
  'external-link': 'external',
  'reference-link': 'external',
  gmail: 'external',
  'doc-entrance': 'external',
  'flow-mail-digest': 'flow',
  'package-import': 'package',
};

export function toolTabFor(tool = {}) {
  if (toolTabById[tool.id]) return toolTabById[tool.id];
  const fields = [tool.id, tool.kind, tool.target, tool.title, tool.detail]
    .map((value) => String(value || '').toLowerCase());
  const haystack = fields.join(' ');
  const kind = fields[1];
  if (kind === 'skill' || haystack.includes('skill') || haystack.includes('recording') || haystack.includes('replay')) {
    return 'flow';
  }
  if (kind === 'mcp' || kind === 'program' || kind === 'package' || kind === 'cli' || haystack.includes('mcp') || haystack.includes('program')) {
    return 'package';
  }
  if (kind === 'connector' || kind === 'external' || kind === 'api' || kind === 'local' || kind === 'website' || haystack.includes('http')) {
    return 'external';
  }
  return 'package';
}

export const fallbackReviewState = {
  highRisk: {
    id: '',
    status: 'none',
    /* i18n: review */ title: _t('dag.highRiskConfirm'),
    action: '',
    skillId: '',
    summaryHash: '',
    permissionSummary: '',
    targetPaths: [],
    diff: [],
    expiresIn: '',
    source: '',
  },
  lowRiskAmbiguity: {
    id: '',
    dismissedUntil: '',
    selectedSkillId: '',
    candidates: [],
  },
  // I-1: Hook candidates (tag patches, subagent candidates, tool registry proposals)
  hookCandidates: {
    tagPatches: [],
    subagentCandidates: [],
    registryProposals: [],
  },
  // I-1: Pending Digest — 五分類映射為三區塊
  pendingDigest: null,
  // I-1: Package install queue
  pendingPackages: [],
  // I-2: Skill activity (最新 2 筆注入紀錄)
  skillInjections: [],
};

export function createDagRunFromMessage(text) {
  const now = Date.now();
  const hasHighRiskStep = dagHighRiskPattern.test(text);
  const nodes = [
    {
      id: `dag-node-${now}-1`,
      /* i18n: DAG nodes */ title: _t('dag.nodeParseTask'),
      action: _t('dag.actionParse'),
      tool: 'main-agent',
      risk: 'low',
      status: 'queued',
    },
    {
      id: `dag-node-${now}-2`,
      title: _t('dag.nodeSelectTool'),
      action: _t('dag.actionSelect'),
      tool: 'skill-router',
      risk: 'medium',
      status: 'queued',
    },
    {
      id: `dag-node-${now}-3`,
      title: hasHighRiskStep ? _t('dag.nodeHighRisk') : _t('dag.nodeOutput'),
      action: hasHighRiskStep ? _t('dag.actionHighRisk') : _t('dag.actionOutput'),
      tool: hasHighRiskStep ? 'review-gate' : 'main-agent',
      risk: hasHighRiskStep ? 'high' : 'low',
      status: 'queued',
    },
  ];

  return {
    id: `dag-${now}`,
    outlineId: `outline-${now}`,
    hookRunId: '',
    title: text.length > 28 ? `${text.slice(0, 28)}...` : text,
    status: 'starting',
    createdAt: new Date(now).toISOString(),
    nodes,
    summaries: [],
    currentNodeId: nodes[0].id,
    summaryHash: '',
  };
}

export function shouldCreateDagRun(text) {
  const raw = String(text || '').trim();
  if (!raw) return false;
  // Auto-DAG runs only after an explicit internal/LLM route. Natural language
  // must reach the LLM first so it can choose chat, search, saved operation, or DAG.
  return /^\/dag\b/i.test(raw) || /^#dag\b/i.test(raw);
}

export function shouldHandleLearningShortcutBeforeLLM() {
  // Keep replay/catalog as LLM-routed tools. The frontend may execute the model's
  // directive later, but it should not classify natural language before the LLM.
  return false;
}

export function getField(source, camelName, snakeName, fallback = '') {
  if (!source) return fallback;
  if (source[camelName] !== undefined && source[camelName] !== null) return source[camelName];
  if (source[snakeName] !== undefined && source[snakeName] !== null) return source[snakeName];
  const pascalName = `${camelName.slice(0, 1).toUpperCase()}${camelName.slice(1)}`;
  if (source[pascalName] !== undefined && source[pascalName] !== null) return source[pascalName];
  return fallback;
}

export function normalizeTaskStatus(status) {
  if (status === 'succeeded') return 'completed';
  if (status === 'waiting_review') return 'waiting_review';
  return status || 'queued';
}

export function isTaskProgressActive(run) {
  return ['starting', 'planning', 'running', 'waiting_review', 'blocked'].includes(run?.status);
}

export function shouldKeepNewerTaskRun(current, incoming) {
  if (!current || !incoming || current.id !== incoming.id) return false;
  const incomingEarly = ['planning', 'running'].includes(incoming.status);
  const currentAdvanced = ['waiting_review', 'completed', 'cancelled', 'failed', 'interrupted'].includes(current.status);
  return incomingEarly && currentAdvanced;
}

export function plannerClarificationFromError(error) {
  const raw = String(error?.message || error || '').replace(/^Error:\s*/i, '').trim();
  if (!raw) return '';
  const candidate = raw.replace(/^任務規劃失敗：\s*/, '').trim();
  const lower = candidate.toLowerCase();
  const hasQuestion = candidate.includes('?') || candidate.includes('？');
  const hasAsk = [
    '請提供', '請告訴', '請確認', '我需要知道', '需要知道', '以下幾點',
    '哪個檔案', '哪個文件', '什麼格式', '什麼內容', '更多資訊',
  ].some((needle) => candidate.includes(needle)) || [
    'please provide', 'please clarify', 'need to know', 'which file',
    'what format', 'more information', 'additional information',
  ].some((needle) => lower.includes(needle));
  const hasBlocker = [
    '無法', '不能', '資訊不足', '資料不足', '不清楚', '太模糊', '缺少',
    '沒有更多資訊', '沒有足夠資訊',
  ].some((needle) => candidate.includes(needle)) || [
    'cannot', "can't", 'unable', 'ambiguous', 'lack', 'not enough information',
  ].some((needle) => lower.includes(needle));
  return (hasAsk && hasBlocker) || (hasQuestion && hasAsk) ? candidate : '';
}

export function mapBackendTaskRun(rawRun) {
  if (!rawRun) return null;
  const nodes = (rawRun.nodes || rawRun.Nodes || []).map((node) => {
    const status = normalizeTaskStatus(getField(node, 'status', 'status', 'queued'));
    const risk = getField(node, 'riskClass', 'risk_class', getField(node, 'risk', 'risk', 'low'));
    const modelRisk = getField(node, 'modelRiskClass', 'model_risk_class', risk);
    return {
      id: getField(node, 'id', 'id'),
      title: getField(node, 'title', 'title', getField(node, 'operation', 'operation', '任務步驟')),
      action: getField(node, 'action', 'action', getField(node, 'operation', 'operation', '')),
      actionCode: getField(node, 'actionCode', 'action_code', ''),
      tool: getField(node, 'executorType', 'executor_type', getField(node, 'tool', 'tool', '')),
      target: getField(node, 'target', 'target', ''),
      risk,
      modelRisk,
      status,
      reviewId: getField(node, 'reviewId', 'review_id', ''),
      startedAt: getField(node, 'startedAt', 'started_at', ''),
      endedAt: getField(node, 'completedAt', 'completed_at', getField(node, 'endedAt', 'ended_at', '')),
      resultSummary: getField(node, 'resultSummary', 'result_summary', ''),
      traceHash: getField(node, 'traceHash', 'trace_hash', ''),
      outputRef: getField(node, 'outputRef', 'output_ref', ''),
    };
  });
  const status = normalizeTaskStatus(getField(rawRun, 'status', 'status', 'starting'));
  return {
    id: getField(rawRun, 'id', 'id', ''),
    outlineId: getField(rawRun, 'outlineId', 'outline_id', ''),
    hookRunId: getField(rawRun, 'hookRunId', 'hook_run_id', ''),
    title: getField(rawRun, 'title', 'title', getField(rawRun, 'prompt', 'prompt', '任務進度')),
    status,
    createdAt: getField(rawRun, 'createdAt', 'created_at', ''),
    nodes,
    summaries: nodes
      .filter((node) => node.resultSummary)
      .map((node) => ({
        nodeId: node.id,
        title: node.title,
        status: node.status,
        risk: node.risk,
        text: node.resultSummary,
      })),
    currentNodeId: getField(rawRun, 'activeNodeId', 'active_node_id', ''),
    summaryHash: getField(rawRun, 'summaryHash', 'summary_hash', ''),
    interruptReason: getField(rawRun, 'interruptReason', 'interrupt_reason', ''),
  };
}

export function formatNodeRiskLabel(node) {
  const finalRisk = node?.risk || 'unknown';
  const modelRisk = node?.modelRisk || finalRisk;
  if (modelRisk && modelRisk !== finalRisk) {
    return `系統判定 ${finalRisk} / 模型 ${modelRisk}`;
  }
  return finalRisk;
}

export function formatDagTime(value) {
  if (!value) return '--:--:--';
  return new Date(value).toLocaleTimeString('zh-TW', {hour12: false});
}

/* i18n: DAG status labels */
export function dagStatusLabel(status) {
  const labels = {
    starting: _t('dag.starting'),
    planning: '規劃中',
    running: _t('dag.running'),
    queued: _t('dag.queued'),
    completed: _t('dag.completed'),
    blocked: _t('dag.blocked'),
    waiting_review: '待確認',
    waiting_user: '等你補充',
    cancelled: '已取消',
    interrupted: '已中斷',
    skipped: '已略過',
    failed: _t('dag.failed'),
  };
  return labels[status] || status || _t('dag.standby');
}

export const adapterMeta = {
  Claude: {className: 'adapter-lime', icon: 'C'},
  Codex: {className: 'adapter-pink', icon: '◎'},
  Gemini: {className: 'adapter-lime', icon: '✦'},
};

export const maxPersonas = 16;

export const personaAvatarUrls = {
  'persona-a': new URL('../assets/persona_avatars/persona-a.svg', import.meta.url).href,
  'persona-b': new URL('../assets/persona_avatars/persona-b.svg', import.meta.url).href,
  'persona-c': new URL('../assets/persona_avatars/persona-c.svg', import.meta.url).href,
  'persona-d': new URL('../assets/persona_avatars/persona-d.svg', import.meta.url).href,
};

export const pixelAvatarRenderSize = 128;

export const pixelAvatarRenderCache = new Map();

export const autoAvatarOverrideStates = new Set(['blocked', 'warning', 'working']);

/* i18n: composer pending */ export const composerPendingInitialText = _t('composer.pendingInitial');

export const composerPendingSlowText = _t('composer.pendingSlow');

export const composerPendingVerySlowText = _t('composer.pendingVerySlow');

export const composerPendingMarker = '\u2063pending:';

export function makeComposerPendingMessage(traceId, text = composerPendingInitialText) {
  return `Ai:${text}${composerPendingMarker}${traceId}`;
}

export function stripComposerPendingMarker(message) {
  const markerIndex = String(message || '').indexOf(composerPendingMarker);
  return markerIndex >= 0 ? message.slice(0, markerIndex) : message;
}

export function replaceComposerPendingMessage(messages, traceId, replacement) {
  const needle = `${composerPendingMarker}${traceId}`;
  const index = messages.findIndex((message) => String(message || '').includes(needle));
  if (index < 0) return [...messages, replacement];
  const next = [...messages];
  next[index] = replacement;
  return next;
}

export const visualLearningInteractiveSelector = [
  'button',
  'a[href]',
  'input',
  'select',
  'textarea',
  '[role="button"]',
  '[role="link"]',
  '[data-vl-target]',
].join(',');

export const learningReplayBlockedSelector = [
  '.rail-mode-record',
  '.sandbox-stop-overlay',
  '.reference-embed-popup',
].join(',');

export const learningRecordingBlockedSelector = [
  '.rail-mode-record',
  '.sandbox-stop-overlay',
  '.reference-embed-popup',
].join(',');

export const visualReplayLastDemoDirective = '[[控制:回放剛剛示範]]';

export const legacyVisualReplayLastDemoDirective = '[[visual_replay:last_demo]]';

export const visualReplayTaggedDirectivePattern = /\[\[控制:回放示範\s+tag=([a-zA-Z0-9_.:-]+)\]\]/;

export const learningReplayStepDelayMs = 950;

export function compactLearningText(value, fallback = '') {
  const text = String(value || '').replace(/\s+/g, ' ').trim();
  return text ? text.slice(0, 120) : fallback;
}

export function cssEscapeIdent(value) {
  if (typeof CSS !== 'undefined' && CSS.escape) return CSS.escape(value);
  return String(value || '').replace(/[^a-zA-Z0-9_-]/g, '\\$&');
}

export function buildLearningSelector(element) {
  if (!element || element.nodeType !== 1) return '';
  if (element.id) return `#${cssEscapeIdent(element.id)}`;
  const parts = [];
  let node = element;
  while (node && node.nodeType === 1 && parts.length < 4) {
    const tag = node.tagName.toLowerCase();
    let part = tag;
    const dataAttr = node.getAttribute('data-testid') ? 'data-testid' : node.getAttribute('data-vl-target') ? 'data-vl-target' : '';
    const testId = dataAttr ? node.getAttribute(dataAttr) : '';
    if (testId) {
      part += `[${dataAttr}="${String(testId).replace(/"/g, '\\"')}"]`;
      parts.unshift(part);
      break;
    }
    if (node.classList?.length) {
      part += `.${Array.from(node.classList).slice(0, 2).map(cssEscapeIdent).join('.')}`;
    }
    const parent = node.parentElement;
    if (parent) {
      const siblings = Array.from(parent.children).filter((child) => child.tagName === node.tagName);
      if (siblings.length > 1) part += `:nth-of-type(${siblings.indexOf(node) + 1})`;
    }
    parts.unshift(part);
    node = parent;
  }
  return parts.join(' > ');
}

export function describeLearningTarget(rawTarget) {
  const element = rawTarget?.nodeType === 1 ? rawTarget : rawTarget?.parentElement;
  const interactive = element?.closest?.(visualLearningInteractiveSelector) || element;
  const rect = interactive?.getBoundingClientRect?.();
  const tag = interactive?.tagName?.toLowerCase?.() || '';
  const role = interactive?.getAttribute?.('role') || (tag === 'a' ? 'link' : tag);
  const label = compactLearningText(
    interactive?.getAttribute?.('aria-label') ||
    interactive?.getAttribute?.('title') ||
    interactive?.innerText ||
    interactive?.value ||
    interactive?.placeholder ||
    tag,
    tag || 'element',
  );
  return {
    element: interactive,
    label,
    role,
    tag,
    selector: buildLearningSelector(interactive),
    rect: rect ? {
      x: Number(rect.x.toFixed(2)),
      y: Number(rect.y.toFixed(2)),
      width: Number(rect.width.toFixed(2)),
      height: Number(rect.height.toFixed(2)),
    } : null,
  };
}

export function buildLearningWindowsAnchor(clickX, clickY, rect, viewport) {
  const width = Math.max(1, Math.round(Number(viewport?.width || window.innerWidth || 1)));
  const height = Math.max(1, Math.round(Number(viewport?.height || window.innerHeight || 1)));
  const click = {
    x: clampReplayCoordinate(Math.round(Number(clickX || 0)), 0, width - 1),
    y: clampReplayCoordinate(Math.round(Number(clickY || 0)), 0, height - 1),
  };
  if (rect && Number(rect.width) > 0 && Number(rect.height) > 0) {
    const box = clampLearningBBox({
      x: Math.round(Number(rect.x || 0)),
      y: Math.round(Number(rect.y || 0)),
      w: Math.round(Number(rect.width || 0)),
      h: Math.round(Number(rect.height || 0)),
    }, width, height);
    return {
      platform: 'windows',
      ok: true,
      mode: 'dom_rect_anchor',
      reason: 'in-app DOM element rect captured during learning; YOLO/OpenCV screenshot resolver was not required',
      click,
      execution_point: {
        x: Math.round(box.x + box.w / 2),
        y: Math.round(box.y + box.h / 2),
      },
      execution_hint: 'click_bbox_center',
      anchor_bbox: box,
      crop_bbox: box,
      ocr_status: 'not_used',
      ocr_note: 'OCR is optional and not used for this recorded DOM anchor.',
      detector_backend: 'dom',
      detector_degraded: false,
      needs_review: false,
    };
  }
  const box = clampLearningBBox({
    x: click.x - 14,
    y: click.y - 14,
    w: 28,
    h: 28,
  }, width, height);
  return {
    platform: 'windows',
    ok: true,
    mode: 'manual_click_box',
    reason: 'no element rectangle was available during learning; preserving the click as a small manual anchor',
    click,
    execution_point: click,
    execution_hint: 'fast_click_original_point',
    anchor_bbox: box,
    crop_bbox: box,
    ocr_status: 'not_used',
    ocr_note: 'OCR is optional and not used for this recorded manual anchor.',
    detector_backend: 'dom',
    detector_degraded: true,
    needs_review: true,
  };
}

export function clampLearningBBox(box, width, height) {
  const w = clampReplayCoordinate(Math.max(1, Math.round(Number(box.w || 1))), 1, width);
  const h = clampReplayCoordinate(Math.max(1, Math.round(Number(box.h || 1))), 1, height);
  const x = clampReplayCoordinate(Math.round(Number(box.x || 0)), 0, Math.max(0, width - w));
  const y = clampReplayCoordinate(Math.round(Number(box.y || 0)), 0, Math.max(0, height - h));
  return { x, y, w, h };
}

export function findActivePersona(settingsState) {
  return settingsState.personas.find((persona) => persona.id === settingsState.activePersonaId) || settingsState.personas[0];
}

export function callWails(fn) {
  try {
    return Promise.resolve(fn());
  } catch (error) {
    return Promise.reject(error);
  }
}

export function isLearningReplayRequest(text) {
  const normalized = String(text || '').trim().toLowerCase();
  return /按照.*剛剛.*示範/.test(normalized)
    || /照.*剛剛.*示範/.test(normalized)
    || /回放.*剛剛.*示範/.test(normalized)
    || /回放.*剛剛.*步驟/.test(normalized)
    || /回放.*剛剛.*操作/.test(normalized)
    || /重播.*剛剛.*示範/.test(normalized)
    || /重播.*剛剛.*步驟/.test(normalized)
    || /重播.*剛剛.*操作/.test(normalized)
    || /再.*示範.*剛剛/.test(normalized)
    || /再.*執行.*剛剛/.test(normalized)
    || /再.*跑.*剛剛/.test(normalized)
    || /剛剛.*步驟/.test(normalized)
    || /replay.*last.*demo/.test(normalized)
    || /replay.*previous.*demo/.test(normalized)
    || /replay.*demo/.test(normalized)
    || /replay.*steps/.test(normalized)
    || /follow.*demo/.test(normalized);
}

export function normalizeLearningOperationQuery(text) {
  const raw = String(text || '').trim();
  if (!raw) return '';
  return raw
    .replace(/操作|操做|operation/gi, ' ')
    .replace(/相關|關於|有關|已保存|已儲存|保存|儲存|錄影紀錄|錄製紀錄|示範紀錄|示範|流程|畫面|tag/gi, ' ')
    .replace(/幫我|請|查詢|搜尋|查找|尋找|找|列出|查看|看看|知道|執行|回放|重播|開啟|打開|有哪些|什麼|甚麼|樣|的/g, ' ')
    .replace(/[，。！？、,.!?;:()[\]{}"'`<>|\\/]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

export function parseLearningOperationCatalogRequest(text) {
  const raw = String(text || '').trim();
  if (!raw) return null;
  const lower = raw.toLowerCase();
  const mentionsSavedCatalog = /已保存|已儲存|保存|儲存|錄影|錄製|示範|紀錄|記錄|catalog/i.test(raw);
  const asksSavedTag = /tag/i.test(raw) && /儲存|保存|已保存|已儲存|畫面|錄影|錄製|示範|紀錄|記錄|操作|操做/.test(raw);
  const asksCatalogList = /有哪些|列出|清單|查看|看看|知道/.test(raw)
    && (mentionsSavedCatalog || /tag/i.test(raw))
    && /操作|操做|錄影|錄製|示範|tag|catalog/i.test(raw);
  // Only explicit catalog/list/tag questions are handled locally. Natural
  // operation requests must go to the LLM so it can choose an action-chain.
  if (!asksSavedTag && !asksCatalogList) return null;
  const catalogQuery = normalizeLearningOperationQuery(raw);
  return catalogQuery ? {mode: 'search', query: catalogQuery} : {mode: 'list', query: ''};
}

export function operationIntentFromCLIResponse(resp) {
  const action = String(resp?.action || '').trim().toLowerCase();
  const target = String(resp?.target || '').trim();
  const next = String(resp?.next || '').trim().toLowerCase();
  if (!target) return null;
  // Internal CLI action-chain contract. User input stays natural language.
  if ((action === '查詢' || action === '搜尋' || action === 'search' || action === 'query') && (next === '操作' || next === '操做')) {
    return {mode: 'query', query: normalizeLearningOperationQuery(target) || target};
  }
  if (action === '操作' || action === '操做') {
    return {mode: 'execute', query: normalizeLearningOperationQuery(target) || target};
  }
  return null;
}

export function resolveLearningOperationMatch(matches) {
  if (!matches.length) return null;
  const [first, second] = matches;
  const firstScore = Number(first?.score || 0);
  const secondScore = Number(second?.score || 0);
  if (firstScore < 1.5) return null;
  if (second && secondScore > 0 && (firstScore - secondScore) < 0.75) return null;
  return first;
}

export function defaultWebSearchProviderOptions() {
  return [
    {
      id: 'tavily',
      name: 'Tavily',
      fields: ['api_key'],
      docs_url: 'https://docs.tavily.com/documentation/api-reference/endpoint/search',
      free_tier_hint: 'AI-agent friendly search; free credits are available from Tavily.',
    },
    {
      id: 'google_cse',
      name: 'Google Custom Search JSON API',
      fields: ['api_key', 'cx'],
      docs_url: 'https://developers.google.com/custom-search/v1/reference/rest/v1/cse/list',
      free_tier_hint: 'Requires an API key and Programmable Search Engine ID.',
    },
    {
      id: 'brave',
      name: 'Brave Search API',
      fields: ['api_key'],
      docs_url: 'https://api-dashboard.search.brave.com/documentation/guides/authentication',
      free_tier_hint: 'Requires a Brave Search API subscription token.',
    },
  ];
}

export function formatLearningOperationSearchResults(query, matches, forExecution = false) {
  if (!matches.length) {
    return `Ai:我找不到符合「${query}」的已保存操作。可以重新示範一次，或換更明確的關鍵詞。`;
  }
  const lines = matches.slice(0, 6).map((item, index) => {
    const risk = item?.risk?.level ? `風險：${item.risk.level}` : '';
    const keywords = Array.isArray(item?.keywords) && item.keywords.length
      ? `關鍵詞：${item.keywords.slice(0, 6).join('、')}`
      : '';
    const opTag = item?.operation_tag ? `操作分類：${item.operation_tag}` : '';
    const score = Number.isFinite(Number(item?.score)) ? `匹配：${Number(item.score).toFixed(2)}` : '';
    const meta = [opTag, keywords, risk, score].filter(Boolean).join('；');
    return `${index + 1}. ${item.title || item.operation_tag || item.tag || item.run_id || '未命名操作'}\n   ${item.summary || '沒有摘要。'}${meta ? `\n   ${meta}` : ''}`;
  });
  const prefix = forExecution
    ? `Ai:「${query}」有多個或不夠明確的操作候選，請換更精準的關鍵詞。`
    : `Ai:我找到這些「${query}」相關操作：`;
  return [prefix, ...lines, '請用更明確的自然語言指定其中一個操作。'].join('\n');
}

export function formatLearningOperationCatalog(items) {
  if (!items.length) {
    return 'Ai:目前還沒有已保存操作。請先開始示範並停止示範，系統就會產生操作名稱、摘要和關鍵詞。';
  }
  const lines = items.slice(0, 10).map((item, index) => {
    const risk = item?.risk?.level ? `風險：${item.risk.level}` : '';
    const keywords = Array.isArray(item?.keywords) && item.keywords.length
      ? `關鍵詞：${item.keywords.slice(0, 6).join('、')}`
      : '';
    const opTag = item?.operation_tag ? `操作分類：${item.operation_tag}` : '';
    const internal = item?.tag || item?.run_id ? `內部：${item.tag || item.run_id}` : '';
    const meta = [opTag, keywords, risk, `步驟：${item.step_count || 0}`, internal].filter(Boolean).join('；');
    return `${index + 1}. ${item.title || item.operation_tag || '未命名操作'}\n   ${item.summary || '沒有摘要。'}\n   ${meta}`;
  });
  return ['Ai:目前已保存的操作如下：', ...lines].join('\n');
}

export function formatLearningOperationLearned(run) {
  const title = run?.title || run?.name || run?.operation_tag || '未命名操作';
  const summary = run?.summary || '已保存這次示範。';
  const keywords = Array.isArray(run?.keywords) && run.keywords.length
    ? `\n關鍵詞：${run.keywords.slice(0, 8).join('、')}`
    : '';
  const opTag = run?.operation_tag ? `\n操作分類：${run.operation_tag}` : '';
  const risk = run?.risk?.level ? `\n風險：${run.risk.level}` : '';
  return `Ai:已保存操作：「${title}」。\n${summary}${opTag}${keywords}${risk}\n之後可以用自然語言請我查找或執行這類操作。`;
}

export function isLearningReplayRelatedText(text) {
  const normalized = String(text || '').trim().toLowerCase();
  if (!normalized) return false;
  return /示範|回放|重播|剛剛.*步驟|剛剛.*操作|照.*剛剛|按照.*剛剛/.test(normalized)
    || /replay|demo|last\s+steps|previous\s+steps/.test(normalized);
}

export function extractVisualReplayDirective(text) {
  const raw = String(text || '');
  const taggedMatch = raw.match(visualReplayTaggedDirectivePattern);
  if (taggedMatch) {
    return {
      shouldReplay: true,
      tag: taggedMatch[1],
      text: raw.replace(visualReplayTaggedDirectivePattern, '').trim(),
    };
  }
  const matchedDirective = raw.includes(visualReplayLastDemoDirective)
    ? visualReplayLastDemoDirective
    : raw.includes(legacyVisualReplayLastDemoDirective)
      ? legacyVisualReplayLastDemoDirective
      : '';
  if (!matchedDirective) {
    return {shouldReplay: false, tag: '', text: raw};
  }
  return {
    shouldReplay: true,
    tag: '',
    text: raw.split(matchedDirective).join('').trim(),
  };
}

export function formatLearningReplayPlan(plan) {
  const steps = Array.isArray(plan?.steps) ? plan.steps : [];
  if (!steps.length) {
    return 'Ai:我還沒有讀到可重放的示範步驟。請先按「開始示範」，點一次目標，然後按「停止示範」。';
  }
  const lines = steps.map((step, index) => {
    const target = step.window_title || step.label || step.role || step.css_selector || step.tag || `座標 (${step.x}, ${step.y})`;
    const selector = step.css_selector ? `，selector: ${step.css_selector}` : '';
    const anchor = step.windows_anchor?.ok ? `，anchor: ${step.windows_anchor.mode || 'available'}` : '，anchor: none';
    const nativeInfo = isNativeReplayStep(step)
      ? `，視窗: ${step.window_title || 'unknown'}，process: ${basenameForDisplay(step.window_process || step.tag || '')}`
      : '';
    const coordLabel = isNativeReplayStep(step) ? '螢幕座標' : '座標';
    return `${index + 1}. ${step.action || 'click'} ${target}，${coordLabel} (${step.x}, ${step.y})${nativeInfo}${selector}${anchor}`;
  });
  const anchorCount = steps.filter((step) => step.windows_anchor?.ok).length;
  return [
    'Ai:我讀到剛剛的示範了，已產生安全 replay plan。確認後會先嘗試操作本視窗內的元素。',
    `Tag: ${plan?.tag || 'demo-last'}，Title: ${plan?.title || plan?.run_name || 'untitled'}。`,
    plan?.run_summary ? `Summary: ${plan.run_summary}` : '',
    `Run: ${plan?.run_id || 'unknown'}，步驟 ${steps.length} 個，visual anchor 覆蓋 ${anchorCount}/${steps.length}。`,
    ...lines,
    '執行時本視窗會先走 selector，外部瀏覽器會走 native 螢幕座標；找不到本視窗元素才 fallback 到示範座標。',
  ].filter(Boolean).join('\n');
}

export function formatLearningReplayConfirmation(plan) {
  const steps = Array.isArray(plan?.steps) ? plan.steps : [];
  const nativeSteps = steps.filter(isNativeReplayStep);
  const domSteps = steps.length - nativeSteps.length;
  const windows = Array.from(new Map(nativeSteps.map((step) => {
    const label = `${basenameForDisplay(step.window_process || step.tag || 'unknown')} - ${step.window_title || 'unknown window'}`;
    return [label, label];
  })).values()).slice(0, 4);
  const lines = [
    `即將依照剛剛的示範執行 ${steps.length} 個步驟。`,
    `本視窗 selector ${domSteps} 步，外部 native click ${nativeSteps.length} 步。`,
  ];
  if (windows.length) {
    lines.push('', '外部目標視窗：', ...windows.map((item) => `- ${item}`));
  }
  lines.push('', '外部 native click 會慢速移動鼠標到目標並短暫停頓；錯誤率會在執行後統整呈現，失敗步驟會跳過並繼續後面。', '確定要執行嗎？');
  return lines.join('\n');
}

export function formatLearningReplayExecutionResult(result) {
  const steps = Array.isArray(result?.steps) ? result.steps : [];
  const okCount = steps.filter((step) => step.ok).length;
  const skipped = steps.filter((step) => step.skipped);
  const failed = steps.filter((step) => !step.ok && !step.skipped);
  const selectorCount = steps.filter((step) => step.ok && step.method === 'selector').length;
  const coordinateCount = steps.filter((step) => step.ok && step.method === 'coordinate').length;
  const nativeTotal = steps.filter((step) => step.method === 'native').length;
  const nativeOK = steps.filter((step) => step.ok && step.method === 'native').length;
  const nativeForegroundOK = steps.filter((step) => step.method === 'native' && step.foreground_ok).length;
  const warned = steps.filter((step) => step.warning);
  const isVisualRelocation = (step) => (
    step.method === 'visual_relocation'
    || String(step.method || '').includes('relocation')
    || Boolean(step.relocation_method)
  );
  const visualTotal = steps.filter(isVisualRelocation).length;
  const visualOK = steps.filter((step) => step.ok && isVisualRelocation(step)).length;
  const visualConfirm = steps.filter((step) => step.needs_confirmation).length;
  const anchorTotal = steps.filter((step) => step.windows_anchor?.ok).length;
  const reviewAnchors = steps.filter((step) => step.windows_anchor?.needs_review).length;
  const lines = [
    `Ai:Replay executor 執行完成：${okCount}/${steps.length} 步成功。`,
    `selector 命中 ${selectorCount} 步，native 座標 ${nativeOK}/${nativeTotal} 步${nativeTotal ? `（前景確認 ${nativeForegroundOK}/${nativeTotal}）` : ''}，座標 fallback ${coordinateCount} 步。`,
    `visual anchor 覆蓋 ${anchorTotal}/${steps.length} 步，需複核 ${reviewAnchors} 步；YOLO/OpenCV 重定位 ${visualOK}/${visualTotal} 步${visualConfirm ? `，待確認 ${visualConfirm} 步` : ''}。`,
    `略過 ${skipped.length} 步，失敗 ${failed.length} 步，警告 ${warned.length} 步。`,
  ];
  const details = steps.filter((step) => step.warning || step.skipped || (!step.ok && !step.skipped) || isVisualRelocation(step)).map((step) => {
    const target = step.label || step.selector || `座標 (${step.x}, ${step.y})`;
    const status = !step.ok && !step.skipped ? 'FAIL' : step.skipped ? 'SKIP' : 'WARN';
    const error = step.error ? `，${step.error}` : step.warning ? `，${step.warning}` : '';
    const relocation = step.relocation_method
      ? `，relocation=${step.relocation_method} ${Number(step.relocation_confidence || 0).toFixed(2)}${step.relocation_reason ? `，${step.relocation_reason}` : ''}`
      : '';
    const debug = step.debug_image_path ? `，debug=${step.debug_image_path}${step.debug_info_path ? `，info=${step.debug_info_path}` : ''}` : '';
    return `${step.index || '?'}. ${status} ${target}${error}${relocation}${debug}`;
  });
  return [...lines, ...details].join('\n');
}

export async function executeLearningReplayPlan(plan, options = {}) {
  const steps = Array.isArray(plan?.steps) ? plan.steps : [];
  const startOffset = Math.max(0, Number(options.startOffset || 0));
  const results = Array.isArray(options.initialResults) ? [...options.initialResults] : [];
  let pendingConfirmation = false;
  for (const step of steps.slice(startOffset)) {
    await delayLearningReplay(learningReplayStepDelayMs);
    const result = await executeLearningReplayStep(step);
    results.push(result);
    if (result?.needs_confirmation) {
      pendingConfirmation = true;
      break;
    }
  }
  return {
    run_id: plan?.run_id || '',
    step_count: steps.length,
    steps: results,
    pending_confirmation: pendingConfirmation,
  };
}

export async function executeLearningReplayStep(step) {
  const selector = String(step?.css_selector || '').trim();
  const action = String(step?.action || 'click').toLowerCase();
  const label = step?.label || step?.role || step?.tag || '';
  if (isNativeReplayStep(step)) {
    return executeNativeLearningReplayStep(step);
  }
  if (isLearningReplayBlockedStep(step)) {
    return {
      ok: false,
      skipped: true,
      method: 'blocked',
      index: step?.index,
      selector,
      label,
      x: step?.x,
      y: step?.y,
      windows_anchor: step?.windows_anchor,
      error: '系統控制項不重播，避免重新開始錄製',
    };
  }
  if (action !== 'click') {
    return {
      ok: false,
      method: 'unsupported',
      index: step?.index,
      selector,
      label,
      x: step?.x,
      y: step?.y,
      windows_anchor: step?.windows_anchor,
      error: `尚未支援 ${action}`,
    };
  }

  const bySelector = findReplayElementBySelector(selector);
  if (bySelector) {
    clickReplayElement(bySelector.element, bySelector.x, bySelector.y);
    return {
      ok: true,
      method: 'selector',
      index: step?.index,
      selector,
      label,
      x: Math.round(bySelector.x),
      y: Math.round(bySelector.y),
      windows_anchor: step?.windows_anchor,
    };
  }

  const point = normalizeReplayPoint(step);
  const byPoint = document.elementFromPoint(point.x, point.y);
  if (byPoint) {
    const interactive = byPoint.closest?.(visualLearningInteractiveSelector) || byPoint;
    if (isLearningReplayBlockedElement(interactive)) {
      return {
        ok: false,
        skipped: true,
        method: 'blocked',
        index: step?.index,
        selector,
        label,
        x: point.x,
        y: point.y,
        windows_anchor: step?.windows_anchor,
        error: '座標落在系統控制項，已略過',
      };
    }
    clickReplayElement(interactive, point.x, point.y);
    return {
      ok: true,
      method: 'coordinate',
      index: step?.index,
      selector,
      label: label || compactLearningText(interactive?.innerText || interactive?.textContent || interactive?.getAttribute?.('aria-label') || ''),
      x: point.x,
      y: point.y,
      windows_anchor: step?.windows_anchor,
    };
  }

  return {
    ok: false,
    method: 'coordinate',
    index: step?.index,
    selector,
    label,
    x: point.x,
    y: point.y,
    windows_anchor: step?.windows_anchor,
    error: '找不到可點擊元素',
  };
}

export function isLearningReplayBlockedStep(step) {
  const selector = String(step?.css_selector || '').toLowerCase();
  const label = String(step?.label || step?.summary || '').trim();
  return selector.includes('rail-mode-record')
    || selector.includes('sandbox-stop-overlay')
    || /示範螢幕操作|開始示範|停止示範|demo|record/i.test(label);
}

export function isLearningReplayBlockedElement(element) {
  return Boolean(element?.closest?.(learningReplayBlockedSelector));
}

export function isNativeReplayStep(step) {
  return step?.source === 'native' || step?.coordinate_space === 'screen';
}

export async function executeNativeLearningReplayStep(step) {
  try {
    const result = await callWails(() => ExecuteNativeLearningReplayStep(JSON.stringify(step)));
    return {
      ok: Boolean(result?.ok),
      skipped: Boolean(result?.skipped),
      needs_confirmation: Boolean(result?.needs_confirmation),
      method: result?.method || 'native',
      index: result?.index || step?.index,
      label: result?.label || step?.label || step?.window_title || '',
      selector: '',
      x: Number.isFinite(Number(result?.x)) ? Number(result.x) : step?.x,
      y: Number.isFinite(Number(result?.y)) ? Number(result.y) : step?.y,
      error: result?.error || '',
      warning: result?.warning || '',
      window_title: result?.window_title || step?.window_title || '',
      window_process: result?.window_process || step?.window_process || step?.tag || '',
      foreground_ok: Boolean(result?.foreground_ok),
      foreground_title: result?.foreground_title || '',
      foreground_process: result?.foreground_process || '',
      relocated: Boolean(result?.relocated),
      relocation_method: result?.relocation_method || '',
      relocation_confidence: Number.isFinite(Number(result?.relocation_confidence)) ? Number(result.relocation_confidence) : 0,
      relocation_reason: result?.relocation_reason || '',
      debug_image_path: result?.debug_image_path || '',
      debug_info_path: result?.debug_info_path || '',
      original_x: Number.isFinite(Number(result?.original_x)) ? Number(result.original_x) : 0,
      original_y: Number.isFinite(Number(result?.original_y)) ? Number(result.original_y) : 0,
      windows_anchor: step?.windows_anchor,
    };
  } catch (error) {
    return {
      ok: false,
      method: 'native',
      index: step?.index,
      label: step?.label || step?.window_title || '',
      x: step?.x,
      y: step?.y,
      error: error?.message || String(error),
      window_title: step?.window_title || '',
      window_process: step?.window_process || step?.tag || '',
      windows_anchor: step?.windows_anchor,
    };
  }
}

export function basenameForDisplay(value) {
  const text = String(value || '').trim();
  if (!text) return '';
  const parts = text.split(/[\\/]/).filter(Boolean);
  return parts[parts.length - 1] || text;
}

export function findReplayElementBySelector(selector) {
  if (!selector) return null;
  try {
    const element = document.querySelector(selector);
    if (!element) return null;
    if (isLearningReplayBlockedElement(element)) return null;
    const rect = element.getBoundingClientRect();
    const x = Math.round(rect.left + rect.width / 2);
    const y = Math.round(rect.top + rect.height / 2);
    if (!isFinite(x) || !isFinite(y)) return null;
    return {element, x, y};
  } catch {
    return null;
  }
}

export function normalizeReplayPoint(step) {
  const sourceWidth = Number(step?.viewport?.width || window.innerWidth || 1);
  const sourceHeight = Number(step?.viewport?.height || window.innerHeight || 1);
  const currentWidth = window.innerWidth || sourceWidth;
  const currentHeight = window.innerHeight || sourceHeight;
  const rawX = Number(step?.x || 0);
  const rawY = Number(step?.y || 0);
  const x = sourceWidth > 0 ? Math.round((rawX / sourceWidth) * currentWidth) : Math.round(rawX);
  const y = sourceHeight > 0 ? Math.round((rawY / sourceHeight) * currentHeight) : Math.round(rawY);
  return {
    x: clampReplayCoordinate(x, 0, Math.max(0, currentWidth - 1)),
    y: clampReplayCoordinate(y, 0, Math.max(0, currentHeight - 1)),
  };
}

export function clampReplayCoordinate(value, min, max) {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, value));
}

export function clickReplayElement(element, x, y) {
  if (!element) return;
  const eventInit = {
    bubbles: true,
    cancelable: true,
    view: window,
    clientX: x,
    clientY: y,
    button: 0,
    buttons: 1,
  };
  if (typeof PointerEvent === 'function') {
    element.dispatchEvent(new PointerEvent('pointerdown', eventInit));
    element.dispatchEvent(new PointerEvent('pointerup', {...eventInit, buttons: 0}));
  }
  element.dispatchEvent(new MouseEvent('mousedown', eventInit));
  element.dispatchEvent(new MouseEvent('mouseup', {...eventInit, buttons: 0}));
  element.dispatchEvent(new MouseEvent('click', {...eventInit, buttons: 0}));
}

export function delayLearningReplay(ms) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

export function appendUniqueReferenceFile(current, file) {
  if (!file) return current;
  if (isInvalidReferencePlaceholder(file)) return current;
  const nextPath = String(file.path || file.name || '');
  const nextName = String(file.name || nextPath);
  let found = false;
  const merged = current.map((item) => {
    const itemPath = String(item.path || item.name || '');
    const itemName = String(item.name || itemPath);
    const isMatch = itemPath === nextPath || (itemName === nextName && nextName);
    if (!isMatch) return item;
    found = true;
    return {...item, ...file};
  });
  return found ? merged : [...current, file];
}

export function referenceFileKey(file) {
  return String(file?.path || file?.name || '');
}

export function normalizeReferenceImportPaths(paths = []) {
  return Array.from(paths || [])
    .map((path) => (typeof path === 'string' ? path : path?.path))
    .map((path) => String(path || '').trim())
    .filter(isNativeFilePath);
}

export function isNativeFilePath(path) {
  return path.startsWith('/') || /^[A-Za-z]:[\\/]/.test(path) || /^\\\\[^\\]+\\[^\\]+/.test(path);
}

export function looksLikeReferenceLocalPath(path) {
  const value = String(path || '').trim();
  return value.startsWith('/') ||
    value.startsWith('~/') ||
    value.startsWith('$HOME/') ||
    value.startsWith('./') ||
    value.startsWith('../') ||
    /^[A-Za-z]:[\\/]/.test(value) ||
    /^\\\\[^\\]+\\[^\\]+/.test(value);
}

export function isInvalidReferencePlaceholder(file) {
  const path = String(file?.path || '').trim();
  const name = String(file?.name || '').trim();
  return (
    file?.status === 'error'
    && !path
    && (!name || name === _t('system.unnamedFile') || name === _t('rightRail.unnamedFile'))
  );
}

export function reorderReferenceFiles(current, draggedKey, targetKey, placement = 'before') {
  if (!draggedKey || !targetKey || draggedKey === targetKey) return current;
  const fromIndex = current.findIndex((file) => referenceFileKey(file) === draggedKey);
  const targetIndex = current.findIndex((file) => referenceFileKey(file) === targetKey);
  if (fromIndex < 0 || targetIndex < 0) return current;

  const next = [...current];
  const [moved] = next.splice(fromIndex, 1);
  const adjustedTargetIndex = next.findIndex((file) => referenceFileKey(file) === targetKey);
  if (adjustedTargetIndex < 0) return current;
  const insertIndex = placement === 'after' ? adjustedTargetIndex + 1 : adjustedTargetIndex;
  next.splice(insertIndex, 0, moved);
  return next;
}

export function mergeReferenceLibraryFiles(current, libraryFiles) {
  const libraryPaths = new Set((libraryFiles || []).map((file) => String(file.path || file.name || '')));
  const nonStaleFiles = (current || []).filter((file) => {
    if (file?.source !== 'library') return true;
    return libraryPaths.has(String(file.path || file.name || ''));
  });
  return (libraryFiles || []).reduce(appendUniqueReferenceFile, nonStaleFiles);
}

export function updateReferenceFileStatus(current, path, patch) {
  const targetPath = String(path || '');
  return current.map((item) => {
    const itemPath = String(item.path || item.name || '');
    return itemPath === targetPath ? {...item, ...patch} : item;
  });
}

export function referenceFileStatusLabel(status) {
  return {
    importing: '處理中',
    checking: '檢查中',
    ready: '已載入',
    error: '失敗',
  }[status] || '已加入';
}

export function shouldShowReferenceFileDetail(file) {
  if (!file?.detail) return false;
  return !(file.source === 'library' && file.status === 'ready');
}

export function avatarStateLabel(state) {
  return getAvatarStateLabels()[state] || state;
}

export function getPixelAvatarDataUrl(pack, state, size = pixelAvatarRenderSize) {
  const cacheKey = `${pack}:${state}:${size}`;
  const cached = pixelAvatarRenderCache.get(cacheKey);
  if (typeof cached === 'string') return Promise.resolve(cached);
  if (cached) return cached;

  const pending = callWails(() => RenderPixelAvatarPreview(pack, state, size))
    .then((bytes) => {
      const dataUrl = bytes?.length ? bytesToDataUrl(bytes) : '';
      if (dataUrl) pixelAvatarRenderCache.set(cacheKey, dataUrl);
      else pixelAvatarRenderCache.delete(cacheKey);
      return dataUrl;
    })
    .catch((error) => {
      pixelAvatarRenderCache.delete(cacheKey);
      throw error;
    });
  pixelAvatarRenderCache.set(cacheKey, pending);
  return pending;
}

export let monitorLinkCache = null;

export let monitorLinkPending = null;

// DEBUG_TRACE_REMOVE: Temporary browser -> local trace viewer bridge.
// Keep while cleaning dead code: this feeds the local debug trace page that
// records UI -> Wails -> Go -> sidecar -> CLI events.
// Uses the monitor-link register because the trace port may move between runs.
export function postDebugTrace(node, traceId, data) {
  resolveMonitorTraceURL()
    .then((url) => {
      const endpoint = `${String(url || '').replace(/\/$/, '')}/trace`;
      return fetch(endpoint, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({node, trace_id: traceId, data}),
      });
    })
    .catch(() => {});
}

export function resolveMonitorTraceURL() {
  if (monitorLinkCache?.url) {
    return Promise.resolve(monitorLinkCache.url);
  }
  if (!monitorLinkPending) {
    monitorLinkPending = Promise.resolve()
      .then(() => GetMonitorLinks?.())
      .then((link) => {
        monitorLinkCache = link || {url: 'http://127.0.0.1:48765'};
        return monitorLinkCache.url;
      })
      .catch(() => {
        monitorLinkCache = {url: 'http://127.0.0.1:48765'};
        return monitorLinkCache.url;
      })
      .finally(() => {
        monitorLinkPending = null;
      });
  }
  return monitorLinkPending;
}

// DEBUG_TRACE_REMOVE: Debug-only trace correlation ID generator.
// Keep with postDebugTrace(): trace IDs are what join UI, Go, and sidecar events.
export function makeDebugTraceID(scope) {
  return `${scope}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

/* i18n: map backend-persisted Chinese labels → current locale display labels.
   The Go backend stores display strings as-is (always Chinese on first save).
   After a locale switch we must translate them so dropdowns & style selectors match. */
export function localizeBackendLabel(value, translationKeyMap) {
  if (!value) return value;
  // If value already matches a current-locale label, keep it
  const currentLabels = Object.values(translationKeyMap);
  if (currentLabels.includes(value)) return value;
  // Otherwise look up Chinese → current-locale mapping
  for (const [zhLabel, tKey] of Object.entries(translationKeyMap)) {
    if (value === zhLabel) return _t(tKey);
  }
  return value; // unknown value, pass through
}

// Mapping tables: Chinese label → translation key
export const _panelLangLabelMap = {
  '繁中': 'settings.langZhTW',
  '中文': 'settings.langZhTW',
  '英文': 'settings.langEn',
  '日文': 'settings.langJa',
  'Traditional Chinese': 'settings.langZhTW',
  'Chinese': 'settings.langZhTW',
  'English': 'settings.langEn',
  'Japanese': 'settings.langJa',
  '中': 'settings.langZhTW',
  'en': 'settings.langEn',
  'ja': 'settings.langJa',
  'pt': 'settings.langPt',
  'pt-PT': 'settings.langPt',
  'es': 'settings.langEs',
  'th': 'settings.langTh',
  'Português': 'settings.langPt',
  '葡萄牙文': 'settings.langPt',
  'Español': 'settings.langEs',
  '西班牙文': 'settings.langEs',
  'ไทย': 'settings.langTh',
  '泰文': 'settings.langTh',
  // self-heal: raw i18n keys leaked by an older build
  'settings.langPt': 'settings.langPt',
  'settings.langEs': 'settings.langEs',
  'settings.langTh': 'settings.langTh',
};

export const _roleLangLabelMap = {
  '自動': 'settings.roleLangAuto',
  '繁中': 'settings.langZhTW',
  '中文': 'settings.langZhTW',
  '英文': 'settings.langEn',
  '日文': 'settings.langJa',
  'Auto': 'settings.roleLangAuto',
  'Traditional Chinese': 'settings.langZhTW',
  'Chinese': 'settings.langZhTW',
  'English': 'settings.langEn',
  'Japanese': 'settings.langJa',
  '中': 'settings.langZhTW',
  'en': 'settings.langEn',
  'ja': 'settings.langJa',
  'pt': 'settings.langPt',
  'es': 'settings.langEs',
  'th': 'settings.langTh',
  'Português': 'settings.langPt',
  '葡萄牙文': 'settings.langPt',
  'Español': 'settings.langEs',
  '西班牙文': 'settings.langEs',
  'ไทย': 'settings.langTh',
  '泰文': 'settings.langTh',
};

export const _fontPresetLabelMap = {
  '預設': 'settings.fontDefault',
  '等寬': 'settings.fontMono',
  '圓潤': 'settings.fontRound',
  'Default': 'settings.fontDefault',
  'Monospace': 'settings.fontMono',
  'Rounded': 'settings.fontRound',
};

// Any historical/localized label (zh-TW / en / ja, current + legacy) → stable key.
// Used to migrate previously-persisted display strings to language-independent keys.
export const _styleLabelToKey = {
  // zh-TW
  '喔黏菊': 'default', '喔黏橘': 'default', '歐黏菊': 'default', '歐黏橘': 'default', '預設': 'default',
  '消極白': 'passiveWhite', '粉切黑': 'pinkBetrayal', '原諒青': 'forgiveMeGreen', '敗北藍': 'defeatBlue',
  // en (literal + meme-pun variants)
  'Sticky Daisy': 'default', 'Legacy Default': 'default', 'Default': 'default', 'Pretty-Please Tangerine': 'default',
  'Passive White': 'passiveWhite', 'Meh-yo White': 'passiveWhite',
  'Pink Betrayal': 'pinkBetrayal', 'Ghosted Pink': 'pinkBetrayal',
  'Forgive-me Green': 'forgiveMeGreen', 'Cuck-Green Pardon': 'forgiveMeGreen',
  'Defeat Blue': 'defeatBlue', 'L-Taken Blue': 'defeatBlue',
  // ja (literal + meme-pun variants)
  'もちもち菊': 'default', 'もちもちオレンジ': 'default', 'おねがいキク': 'default',
  '消極ホワイト': 'passiveWhite', 'どうでも白': 'passiveWhite',
  '裏切りピンク': 'pinkBetrayal', '既読スルーピンク': 'pinkBetrayal',
  '許してグリーン': 'forgiveMeGreen', '寝取られグリーン': 'forgiveMeGreen',
  '敗北ブルー': 'defeatBlue', '完敗ブルー': 'defeatBlue',
  // pt-PT
  'Laranja Faz-Favor': 'default', 'Branco Tanto-Faz': 'passiveWhite',
  'Rosa Deixado em Visto': 'pinkBetrayal', 'Verde Corno Perdoado': 'forgiveMeGreen',
  'Azul Levei um L': 'defeatBlue',
  // es
  'Naranja Porfa': 'default', 'Blanco Da Igual': 'passiveWhite',
  'Rosa Dejado en Visto': 'pinkBetrayal', 'Verde Cornudo Perdón': 'forgiveMeGreen',
  'Azul Me Dieron la L': 'defeatBlue',
  // th
  'ส้มอ้อนวอน': 'default', 'ขาวช่างมัน': 'passiveWhite',
  'ชมพูอ่านไม่ตอบ': 'pinkBetrayal', 'เขียวสวมเขา': 'forgiveMeGreen',
  'ฟ้าแพ้ราบคาบ': 'defeatBlue',
};

// Resolve any value (stable key OR localized label, any language) to a stable key.
export function styleKeyOf(value) {
  if (typeof value !== 'string' || !value) return defaultPanelStyle;
  if (STYLE_KEYS.includes(value)) return value;
  return _styleLabelToKey[value] || defaultPanelStyle;
}

// Live display label for a style, in the current language.
export function styleLabel(value) {
  return _t(STYLE_KEY_I18N[styleKeyOf(value)] || 'style.default');
}

// Kept name for compatibility; now returns a stable key (not a display string).
export function normalizePanelStyle(style) {
  return styleKeyOf(style);
}

export function normalizeStyleOptionOrder(options = []) {
  const ordered = options.map(normalizePanelStyle).filter((option) => styleOptions.includes(option));
  return [...new Set([...ordered, ...styleOptions])];
}

export function loadStyleOptionOrder() {
  try {
    return normalizeStyleOptionOrder(JSON.parse(window.localStorage.getItem(styleOptionOrderStorageKey) || '[]'));
  } catch {
    return styleOptions;
  }
}

export function saveStyleOptionOrder(options) {
  try {
    window.localStorage.setItem(styleOptionOrderStorageKey, JSON.stringify(normalizeStyleOptionOrder(options)));
  } catch {
    /* local UI preference only */
  }
}

export function normalizePanelSettings(panel = {}) {
  return {...panel, panelStyle: normalizePanelStyle(panel.panelStyle)};
}

export function panelFromUISettings(uiSettings = {}, fallbackPanel = fallbackSettings.panel) {
  return {
    panelLanguage: localizeBackendLabel(uiSettings.panel_language, _panelLangLabelMap) || fallbackPanel.panelLanguage,
    roleLanguage: localizeBackendLabel(uiSettings.role_language, _roleLangLabelMap) || fallbackPanel.roleLanguage,
    fontPreset: localizeBackendLabel(uiSettings.font_preset, _fontPresetLabelMap) || fallbackPanel.fontPreset,
    fontScale: uiSettings.font_scale || fallbackPanel.fontScale,
    panelStyle: styleKeyOf(uiSettings.panel_style || fallbackPanel.panelStyle),
  };
}

export function normalizeSettingsState(settingsState = {}, fallback = fallbackSettings) {
  const merged = {
    ...fallback,
    ...settingsState,
    panel: {
      ...(fallback.panel || fallbackSettings.panel),
      ...(settingsState.panel || {}),
    },
  };
  const personas = normalizeLockedPersonas(merged.personas || fallbackSettings.personas);
  const activePersonaId = personas.some((persona) => persona.id === merged.activePersonaId)
    ? merged.activePersonaId
    : lockedPersonaId;
  return {...merged, activePersonaId, personas, panel: normalizePanelSettings(merged.panel)};
}

export function normalizeLockedPersonas(personas = []) {
  // "憂樂傻酷" is a reserved persona identity, not a reserved slot. Keep its
  // name stable while preserving whatever ordering the user chose, because the
  // app treats the first card as the current main persona.
  const normalized = personas.map((persona) => (
    persona.id === lockedPersonaId
      ? {...persona, name: lockedPersonaName, identity: persona.identity || _t('persona.defaultIdentityA')}
      : persona
  ));
  const lockedIndex = normalized.findIndex((persona) => persona.id === lockedPersonaId);
  if (lockedIndex < 0) {
    return [fallbackSettings.personas[0], ...normalized];
  }
  return normalized;
}

export function normalizeToolList(toolList = []) {
  const merged = new Map(getFallbackTools().map((tool) => [tool.id, tool]));
  toolList.forEach((tool) => {
    if (!tool?.id) return;
    merged.set(tool.id, {...(merged.get(tool.id) || {}), ...tool});
  });
  return Array.from(merged.values());
}

export function panelStyleTheme(style) {
  return STYLE_KEY_THEME[styleKeyOf(style)] || 'onanegiku';
}

// Converts persisted values like "80%" into a bounded CSS scale number.
export function fontScaleValue(fontScale) {
  const value = Number.parseInt(String(fontScale || '100%').replace('%', ''), 10);
  const normalized = Number.isNaN(value) ? 100 : value;
  return Math.min(120, Math.max(50, normalized)) / 100;
}

export function getPersonaAvatar(persona) {
  return persona?.avatarUrl || personaAvatarUrls[persona?.id] || personaAvatarUrls['persona-a'];
}

export function bytesToDataUrl(bytes = [], mimeType = 'image/png') {
  if (typeof bytes === 'string') return `data:${mimeType};base64,${bytes}`;
  let binary = '';
  const chunkSize = 8192;
  for (let index = 0; index < bytes.length; index += chunkSize) {
    binary += String.fromCharCode(...bytes.slice(index, index + chunkSize));
  }
  return `data:${mimeType};base64,${window.btoa(binary)}`;
}

export function resolveAvatarProvider(config) {
  return config?.avatar_provider || config?.AvatarProvider || 'built_in_pixel';
}

export function resolveStaticAvatarPath(config) {
  return config?.static_avatar_path || config?.StaticAvatarPath || '';
}

export function defaultPixelPackForPersona(personaID) {
  if (personaID === 'persona-b') return 'uncle';
  if (personaID === 'persona-c') return 'secretary';
  return 'wolf';
}

export function pixelPackForPersona(persona, config) {
  const pack = config?.pixel_pack || config?.PixelPack || '';
  if (['wolf', 'uncle', 'secretary'].includes(pack)) return pack;
  return defaultPixelPackForPersona(persona?.id);
}

export function hasPendingLowRiskAmbiguity(reviewState) {
  const ambiguity = reviewState?.lowRiskAmbiguity;
  if (!ambiguity || ambiguity.dismissedUntil) return false;
  return Boolean(ambiguity.id || ambiguity.candidates?.length) && ambiguity.selectedSkillId === '';
}

export function deriveAvatarExpression({dagRun, reviewState, sourceTrustHint, hasConversation = true, lastMessageText = '', windowInactive = false, idleMs = 0}) {
  if (windowInactive) return 'sleepy';
  if (reviewState?.hasBlocking || hasPendingLowRiskAmbiguity(reviewState)) return 'warning';
  if (sourceTrustHint?.risk_level === 'high' || sourceTrustHint?.risk_level === 'danger') return 'blocked';
  if (sourceTrustHint && sourceTrustHint.level !== 'trusted') return 'warning';
  if (dagRun?.status === 'running' || dagRun?.status === 'starting') return 'working';
  if (dagRun?.status === 'blocked') return 'warning';
  if (dagRun?.status === 'failed') return 'sad';
  if (dagRun?.status === 'completed') return 'happy';
  const messageExpression = deriveMessageAvatarExpression(lastMessageText);
  if (messageExpression) return messageExpression;
  if (!hasConversation) return idleMs >= 15 * 60 * 1000 ? 'idle' : 'sleepy';
  return 'idle';
}

export function deriveMessageAvatarExpression(text = '') {
  const normalized = String(text || '').trim().toLowerCase();
  if (!normalized) return 'speechless';
  if (/(笑話|joke|哈哈|呵呵|好笑|冷笑話)/i.test(normalized)) return 'speechless';
  if (/(讚|太棒|做得好|漂亮|厲害|謝謝|感謝|good job|nice|great|awesome)/i.test(normalized)) return 'happy';
  if (/(罵|爛|笨|廢|糟糕|生氣|bad|stupid|useless)/i.test(normalized)) return 'sad';
  if (/(build fail|build failed|test fail|test failed|測試失敗|建置失敗|api 錯誤|llm 錯誤|錯誤|失敗)/i.test(normalized)) return 'sad';
  if (/(設定缺漏|degraded|未驗證|低風險|warning|warn)/i.test(normalized)) return 'warning';
  if (/(權限不足|credential|token 錯|discord token|高風險|需要確認|blocked|forbidden)/i.test(normalized)) return 'blocked';
  return '';
}

export function staticAvatarSrc(path) {
  if (!path) return '';
  if (/^(data:|https?:|file:)/.test(path)) return path;
  return path.startsWith('/') ? path : `/${path}`;
}

export function resolvePersonaAvatarSrc(persona, config, previews = {}, pixelFallback = '') {
  const provider = resolveAvatarProvider(config);
  const staticPath = resolveStaticAvatarPath(config);
  if (provider === 'static_image' && staticPath) {
    return previews[persona?.id] || staticAvatarSrc(staticPath);
  }
  return pixelFallback || getPersonaAvatar(persona);
}

export function parseToneChance(roleStrength) {
  const value = Number.parseInt(String(roleStrength || '20%').replace('%', ''), 10);
  if (Number.isNaN(value)) return 20;
  return Math.min(100, Math.max(10, Math.round(value / 10) * 10));
}

export function limitChineseText(text, maxLength) {
  return Array.from(String(text || '')).slice(0, maxLength).join('');
}

export function reviewStatusLabel(status) {
  return {
    pending: _t('review.pendingConfirm'),
    approved: _t('review.approved'),
    rejected: _t('review.rejected'),
    custom: _t('review.customInput'),
    expired: _t('review.expired'),
  }[status] || status;
}

export function reviewCardID(card) {
  return card?.review_id || card?.ID || card?.id || '';
}

export function reviewConsequenceText(status, note) {
  if (note) return note;
  return {
    pending: _t('review.pendingHint'),
    approved: _t('review.approvedHint'),
    rejected: _t('review.rejectedHint'),
    custom: _t('review.customHint'),
    expired: _t('review.expiredHint'),
  }[status] || _t('review.syncHint');
}

export function replyStrategyPresetFor(value) {
  const key = String(value || '').trim();
  if (!key) return null;
  return getReplyStrategyPresets().find((preset) => preset.id === key || preset.label === key) || null;
}

export function cycleValue(options, current) {
  const index = options.indexOf(current);
  return options[(index + 1) % options.length] || options[0];
}

export function voiceStatusLabel(status) {
  switch (status) {
    case 'ready':
      return 'whisper.cpp ready';
    case 'invalid_model':
      return _t('voice.voiceModelReinstall');
    case 'missing_binary':
      return _t('settings.voiceNotIncluded');
    case 'missing_binary_and_model':
      return _t('settings.voiceNotInstalled');
    case 'missing_model':
      return _t('settings.voiceModelNotInstalled');
    default:
      return _t('settings.voiceNotInstalled');
  }
}

export function voiceLanguageModeLabel(mode) {
  switch (mode) {
    case 'follow_app':
      return _t('settings.voiceFollowApp');
    case 'manual':
      return _t('settings.voiceManualSpecify');
    case 'auto':
    default:
      return _t('settings.voiceAutoDetect');
  }
}

export function reorderItems(items, sourceItem, targetItem) {
  const sourceIndex = items.indexOf(sourceItem);
  const targetIndex = items.indexOf(targetItem);
  if (sourceIndex < 0 || targetIndex < 0) return items;
  const next = [...items];
  next.splice(sourceIndex, 1);
  next.splice(targetIndex, 0, sourceItem);
  return next;
}

export const goProgramDagStepCatalog = [
  {
    id: 'import',
    title: '接收程式草稿',
    detail: '模型只提交 JSON draft；系統建立受控工作區與 Go 原始碼檔案。',
  },
  {
    id: 'validate',
    title: '契約與權限檢查',
    detail: '檢查 package main、schema、stdlib/vendor allowlist，攔下未授權網路或子程序。',
  },
  {
    id: 'build',
    title: '受控編譯',
    detail: '使用內建 Go toolchain 編譯，禁止下載 module，套用 timeout 與輸出限制。',
  },
  {
    id: 'plan',
    title: '準備資料輸入',
    detail: '把天氣 JSON、衣服表格或 DB 查詢結果整理成 stdin JSON，寫入只走 scratch。',
  },
  {
    id: 'execute',
    title: '執行與結果審核',
    detail: '限時執行，驗證 stdout JSON contract；成功後只保存為 pending skill。',
  },
];

export function goProgramStepStatus(run, index) {
  const status = String(run?.status || '').toLowerCase();
  if (status === 'completed') return 'done';
  if (status === 'needs_user_decision') return index === 0 ? 'blocked' : 'pending';
  if ((run?.attempt_count || 0) > 0) return index === 0 ? 'done' : (index === 1 ? 'active' : 'pending');
  return index === 0 ? 'active' : 'pending';
}

export function goProgramStepStatusLabel(status) {
  return {done: '完成', active: '進行中', blocked: '待確認', pending: '等待'}[status] || '等待';
}

export function summarizeGoProgramPermissions(permissions = {}) {
  const labels = [];
  if (permissions.read_app_data) labels.push('讀取 app data');
  if (permissions.read_outputs) labels.push('讀取 outputs');
  if (permissions.write_outputs_scratch) labels.push('寫入 scratch outputs');
  if (permissions.read_db_as_json) labels.push('讀取系統提供 DB JSON');
  if (Array.isArray(permissions.read_mounted_paths) && permissions.read_mounted_paths.length) labels.push('讀取受限掛載路徑');
  if (permissions.network) labels.push('網路需審核');
  if (permissions.shell_subprocess) labels.push('子程序需審核');
  return labels.length ? labels.join('、') : '無額外權限';
}

export function summarizeGoProgramDataSources(sources = []) {
  const labels = sources.map((source) => source?.name || source?.kind).filter(Boolean);
  return labels.length ? labels.join('、') : '由系統在執行時注入 JSON；尚未綁定外部資料來源';
}

export function goProgramLifecycleLabel(run = {}) {
  if (run?.pending_skill_id) return 'Pending skill';
  return {completed: '完成', ready: '待製作', authoring: '製作中', needs_user_decision: '待確認'}[run?.status] || run?.status || '待命';
}

// ---------------------------------------------------------------------------
// I-4: State Token Legend (static color explanation)
// ---------------------------------------------------------------------------
/* i18n: stateToken */
export function getStateTokenDescriptions() {
  return [
    {color: 'var(--amber, #f5a623)', label: _t('stateToken.highRiskLabel'), desc: _t('stateToken.highRiskDesc')},
    {color: 'var(--msg, #7f2d20)', label: _t('stateToken.interruptedLabel'), desc: _t('stateToken.interruptedDesc')},
    {color: '#4caf50', label: _t('stateToken.normalLabel'), desc: _t('stateToken.normalDesc')},
    {color: 'var(--muted, #bbb5a5)', label: _t('stateToken.idleLabel'), desc: _t('stateToken.idleDesc')},
  ];
}

// SEC-06: 訊息內 URL 偵測（與後端 urlInTextRe 對齊）。
export const MESSAGE_URL_RE = /https?:\/\/[^\s'"<>）)】\]]+/g;

// I-1: Categorize digest items into three UI blocks per spec §12
export function categorizePendingDigest(digest) {
  const blocks = {urgent: [], later: [], archive: []};
  if (!digest || !digest.items) return blocks;
  for (const item of digest.items) {
    const cat = item.category || '';
    if (cat === 'risky' || cat === 'high_value') {
      blocks.urgent.push(item);
    } else if (cat === 'keep_suggestion') {
      blocks.later.push(item);
    } else if (cat === 'archive_suggestion' || cat === 'duplicate_group') {
      blocks.archive.push(item);
    } else {
      blocks.later.push(item);
    }
  }
  return blocks;
}

// §3.1.11 影片副檔名判斷：拖入時用來分流到 data/videos
export function isVideoPath(p) {
  return /\.(mp4|mov|m4v|webm|mkv|avi|wmv|flv|mpe?g|3gp|ogv)$/i.test(String(p || ''));
}

// §3.1.10 檔案方框右下角的副檔名角標（txt / md / jpeg / wmv …）
export function fileExtLabel(name) {
  const m = /\.([A-Za-z0-9]+)$/.exec(String(name || ''));
  return m ? m[1].toLowerCase() : '';
}

export function twoLineFileName(name, fallback = 'Unnamed File') {
  const chars = Array.from(String(name || fallback));
  const first = chars.slice(0, 20).join('');
  const second = chars.slice(20, 40).join('');
  return second ? [first, second] : [first];
}

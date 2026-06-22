import { t as _t } from '../locales/useI18n';

// Shared pure helpers extracted from App.jsx during the component split.
// Grouped here so both app/ and features/ modules share a single source of truth.

// Replay steps -------------------------------------------------------------
export function isNativeReplayStep(step) {
  return step?.source === 'native' || step?.coordinate_space === 'screen';
}

// Text ---------------------------------------------------------------------
export function basenameForDisplay(value) {
  const text = String(value || '').trim();
  if (!text) return '';
  const parts = text.split(/[\\/]/).filter(Boolean);
  return parts[parts.length - 1] || text;
}

// Readiness Gate candidate sets -------------------------------------------
export function isExclusiveCandidateSet(candidates = []) {
  if (candidates.length !== 2) return false;
  const labels = candidates.map((candidate) => String(candidate?.label || '').trim().toLowerCase());
  const yesLike = /^(是|對|好|可以|要|確認|同意|執行|yes|y|ok|okay|confirm|proceed|continue)$/i;
  const noLike = /^(否|不是|不|不要|取消|拒絕|略過|no|n|cancel)$/i;
  return labels.some((label) => yesLike.test(label)) && labels.some((label) => noLike.test(label));
}

// Web search providers -----------------------------------------------------
export function defaultWebSearchProviderOptions() {
  return [
    {
      id: 'tavily',
      name: 'Tavily',
      fields: ['api_key'],
      docs_url: 'https://docs.tavily.com/documentation/api-reference/endpoint/search',
      free_tier_hint: 'AI 研究，含免費額度',
      best_for: 'AI 助理、研究整理、文件搜尋',
      setup_guide: ['在 Tavily 建立 API key。', '把 API key 貼到下方欄位。', '按下儲存後即可開始網路搜尋。'],
      usage_hint: '例如：網路搜尋 OpenAI API 最新文件',
      api_key_placeholder: 'tvly-...',
    },
    {
      id: 'exa',
      name: 'Exa',
      fields: ['api_key'],
      docs_url: 'https://exa.ai/docs/reference/search',
      free_tier_hint: '研究文檔，20k/月免費',
      best_for: '技術文件、文章、公司資料、研究型搜尋',
      setup_guide: ['到 Exa dashboard 產生 API key。', '把 API key 貼到下方欄位。', '按下儲存後即可開始網路搜尋。'],
      usage_hint: '例如：網路搜尋 最新 LLM 論文',
      api_key_placeholder: 'Exa API key',
    },
    {
      id: 'serpapi',
      name: 'SerpApi',
      fields: ['api_key'],
      docs_url: 'https://serpapi.com/search-api',
      free_tier_hint: 'Google 結果，250/月免費',
      best_for: '想拿 Google 風格搜尋結果資料',
      setup_guide: ['到 SerpApi 產生 private API key。', '把 API key 貼到下方欄位。', '按下儲存後即可開始網路搜尋。'],
      usage_hint: '例如：網路搜尋 台北 天氣',
      api_key_placeholder: 'SerpApi API key',
    },
    {
      id: 'brave',
      name: 'Brave',
      fields: ['api_key'],
      docs_url: 'https://api-dashboard.search.brave.com/documentation/guides/authentication',
      free_tier_hint: '一般網頁，需 API token',
      best_for: '通用網頁搜尋與一般資訊查找',
      setup_guide: ['在 Brave Search API 建立 subscription token。', '把 token 貼到下方欄位。', '按下儲存後即可開始網路搜尋。'],
      usage_hint: '例如：網路搜尋 AI agent search API',
      api_key_placeholder: 'Brave subscription token',
    },
  ];
}

// Tool tab routing --------------------------------------------------------

const toolTabById = {
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

// DAG status label --------------------------------------------------------
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

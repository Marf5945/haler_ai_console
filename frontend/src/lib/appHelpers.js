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

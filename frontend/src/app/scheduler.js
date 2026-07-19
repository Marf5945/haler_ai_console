import {t as _t} from '../locales/useI18n';

const schedulerWeekdayMap = [
  ['日', 0], ['天', 0], ['一', 1], ['二', 2], ['三', 3], ['四', 4], ['五', 5], ['六', 6],
];
const schedulerSlotLabelKeys = {name: 'chatSystem.scheduler.slotName', time: 'chatSystem.scheduler.slotTime'};
const schedulerChineseNumberMap = {
  零: 0,
  〇: 0,
  一: 1,
  二: 2,
  兩: 2,
  三: 3,
  四: 4,
  五: 5,
  六: 6,
  七: 7,
  八: 8,
  九: 9,
};

export function isSchedulerAffirmation(text) {
  return /^(可以|可以了|好|好的|確認|確定|沒問題|ok|okay|yes|建立|寫入|送出|對|對的)$/i.test(String(text || '').trim());
}

export function isSchedulerCancellation(text) {
  return /^(取消|不要了|先不用|算了|停止)$/i.test(String(text || '').trim());
}

export function schedulerDefaultPayload(title, actionText = '', summary = '') {
  const cleanTitle = title || actionText || '提醒';
  const cleanAction = actionText || cleanTitle;
  const cleanSummary = summary || `在排程時間執行：${cleanAction}`;
  return JSON.stringify({
    event_name: 'scheduler:reminder',
    data: {
      title: cleanTitle,
      action: cleanAction,
      summary: cleanSummary,
    },
  });
}

export function parseSchedulerChineseNumber(text) {
  const raw = String(text || '').trim();
  if (!raw) return null;
  if (/^\d+$/.test(raw)) return Number(raw);
  if (Object.prototype.hasOwnProperty.call(schedulerChineseNumberMap, raw)) return schedulerChineseNumberMap[raw];
  if (raw === '十') return 10;
  const tenIndex = raw.indexOf('十');
  if (tenIndex < 0) return null;
  const left = raw.slice(0, tenIndex);
  const right = raw.slice(tenIndex + 1);
  const tens = left ? schedulerChineseNumberMap[left] : 1;
  const ones = right ? schedulerChineseNumberMap[right] : 0;
  if (tens == null || ones == null) return null;
  return tens * 10 + ones;
}

export function normalizeSchedulerTimeNumerals(text) {
  return String(text || '').replace(
    /([零〇一二兩三四五六七八九十]{1,3})(?=\s*(?:點|:|：|分|日|號))/g,
    (match) => {
      const parsed = parseSchedulerChineseNumber(match);
      return parsed == null ? match : String(parsed);
    },
  );
}

export function hasSchedulerClockText(text) {
  return /(?:上午|早上|中午|下午|晚上)?\s*\d{1,2}(?:[:：]\d{2}|點(?:半|\d{1,2}分?)?)/.test(normalizeSchedulerTimeNumerals(text));
}

export function hasSchedulerTimeText(text) {
  const raw = normalizeSchedulerTimeNumerals(text);
  const hasClock = hasSchedulerClockText(raw);
  const hasWeekly = /(?:每\s*)?(?:週|周|星期|禮拜)[日天一二三四五六]/.test(raw);
  const hasMonthly = /每\s*月\s*\d{1,2}\s*(?:日|號)?/.test(raw);
  return /@(?:hourly|daily|weekly|monthly|yearly)/i.test(raw)
    || /\d{1,2}\s+\d{1,2}\s+\*|\*\s+\*\s+\*/.test(raw)
    || /每\s*(小時|一小時|個小時)|hourly/i.test(raw)
    || ((/每天|每日|每\s*(天|日)/.test(raw) || hasWeekly || hasMonthly) && hasClock);
}

export function stripSchedulerTimeText(text) {
  return normalizeSchedulerTimeNumerals(text)
    .replace(/@(?:hourly|daily|weekly|monthly|yearly)/ig, ' ')
    .replace(/(?:上午|早上|中午|下午|晚上)?\s*\d{1,2}(?:[:：]\d{2}|點(?:半|\d{1,2}分?)?)?/g, ' ')
    .replace(/每\s*(小時|一小時|個小時)|每天|每日|每週|每周|每月|星期[日天一二三四五六]|禮拜[日天一二三四五六]|週[日天一二三四五六]|周[日天一二三四五六]/g, ' ')
    .replace(/每\s*月\s*\d{1,2}\s*(?:日|號)?/g, ' ');
}

export function extractSchedulerDeliveryText(text) {
  const raw = String(text || '').trim();
  const match = raw.match(/(?:幫我|請|再|並|然後)?\s*(?:做成|整理成|產出|生成|製作成)\s*(?:一個|一份|一張|一套)?\s*([^，,。；;\n]+)/);
  if (!match?.[1]) return '';
  return `整理成${match[1].trim()}`;
}

export function removeSchedulerDeliveryClauses(text) {
  return String(text || '')
    .replace(/(?:，|,|。|；|;)?\s*(?:幫我|請|再|並|然後)?\s*(?:做成|整理成|產出|生成|製作成)\s*(?:一個|一份|一張|一套)?\s*[^，,。；;\n]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}

export function parseSchedulerTimeText(text, fallback = '') {
  const raw = normalizeSchedulerTimeNumerals(text);
  const shortcut = raw.match(/@(hourly|daily|weekly|monthly|yearly)/i);
  if (shortcut) return `@${shortcut[1].toLowerCase()}`;
  const cron = raw.match(/(?:^|\s)(\S+\s+\S+\s+\S+\s+\S+\s+\S+)(?:\s|$)/);
  if (cron && /^[\d*,\-\/]+\s+[\d*,\-\/]+\s+[\d*,\-\/]+\s+[\d*,\-\/]+\s+[\d*,\-\/]+$/.test(cron[1])) return cron[1];
  const hasClock = hasSchedulerClockText(raw);
  if (!hasSchedulerTimeText(raw) && !hasClock) return fallback;
  const timeMatch = raw.match(/(?:上午|早上|中午|下午|晚上)?\s*(\d{1,2})(?:[:：](\d{2})|點(?:(半)|(\d{1,2})分?)?)/);
  let hour = 9;
  let minute = 0;
  if (timeMatch) {
    hour = Math.min(23, Math.max(0, Number(timeMatch[1])));
    minute = timeMatch[2]
      ? Math.min(59, Math.max(0, Number(timeMatch[2])))
      : (timeMatch[3] ? 30 : (timeMatch[4] ? Math.min(59, Math.max(0, Number(timeMatch[4]))) : 0));
    if ((raw.includes('下午') || raw.includes('晚上')) && hour < 12) hour += 12;
    if (raw.includes('中午') && hour < 11) hour += 12;
  }
  if (/每\s*(小時|一小時|個小時)|hourly/i.test(raw)) return '@hourly';
  const weekdayToken = schedulerWeekdayMap.find(([label]) => new RegExp(`(?:每\\s*)?(?:週|周|星期|禮拜)${label}`).test(raw));
  if (weekdayToken) return `${minute} ${hour} * * ${weekdayToken[1]}`;
  const monthMatch = raw.match(/每\s*月\s*(\d{1,2})\s*(?:日|號)?/);
  if (monthMatch) return `${minute} ${hour} ${Math.min(31, Math.max(1, Number(monthMatch[1])))} * *`;
  if (/每天|每日|daily/i.test(raw)) return `${minute} ${hour} * * *`;
  if (hasClock) return `${minute} ${hour} * * *`;
  return fallback;
}

export function cleanSchedulerNameText(text) {
  const raw = removeSchedulerDeliveryClauses(stripSchedulerTimeText(text));
  const cleaned = String(raw || '')
    .replace(/我想要|想要|我要|我想|請|幫我|幫忙|新增|建立|規劃|排程|排定|安排|任務|提醒/g, ' ')
    .replace(/一個|一項|一筆|一下/g, ' ')
    .replace(/時間(改成|改為|為|是).*/g, ' ')
    .replace(/動作(改成|改為|為|是).*/g, ' ')
    .replace(/的$/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  return cleaned.slice(0, 32);
}

export function parseSchedulerActionText(text) {
  const raw = removeSchedulerDeliveryClauses(String(text || '').trim());
  const channelMatch = raw.match(/(?:要)?(?:用|以)\s*(.+?)\s*(?:提醒我|通知我|叫我)$/);
  if (channelMatch?.[1]) return `${channelMatch[1].trim()}提醒`;
  const remindMeSuffix = raw.match(/(.+?)\s*(?:提醒我|通知我|叫我)$/);
  if (remindMeSuffix?.[1]) return remindMeSuffix[1].trim();
  const notifyMatch = raw.match(/(?:提醒我|叫我|通知我)\s*(.+)$/);
  if (notifyMatch?.[1]) return notifyMatch[1].trim();
  const runMatch = raw.match(/(?:執行|做|跑)\s*(.+)$/);
  if (runMatch?.[1]) return runMatch[1].trim();
  const actionMatch = raw.match(/動作(?:改成|改為|為|是|:|：)\s*(.+)$/);
  if (actionMatch?.[1]) return actionMatch[1].trim();
  const quoted = raw.match(/[「『"]([^」』"]+)[」』"]/);
  if (quoted?.[1]) return quoted[1].trim();
  return cleanSchedulerNameText(raw);
}

export function formatSchedulerSummary(draft) {
  const delivery = draft?.deliveryText || extractSchedulerDeliveryText(draft?.sourceText || '');
  const task = String(draft?.actionText || draft?.name || '提醒').trim();
  const output = delivery ? `，${delivery}` : '';
  return `在設定時間執行「${task}」${output}。`;
}

export function normalizeSchedulerDraft(draft) {
  const actionText = String(draft.actionText || '').trim();
  const name = draft.name || actionText;
  const summary = String(draft.summary || '').trim();
  const finalSummary = summary || formatSchedulerSummary({...draft, name, actionText});
  return {
    name: String(name || '').trim(),
    cronExpr: String(draft.cronExpr || '').trim(),
    actionText,
    summary: finalSummary,
    deliveryText: String(draft.deliveryText || '').trim(),
    sourceText: String(draft.sourceText || '').trim(),
    actionPayload: draft.actionPayload || (name || actionText ? schedulerDefaultPayload(name || '提醒', actionText || name, finalSummary) : ''),
  };
}

export function schedulerMissingSlots(draft) {
  const normalized = normalizeSchedulerDraft(draft || {});
  return ['name', 'time'].filter((slot) => {
    if (slot === 'name') return !normalized.name;
    if (slot === 'time') return !normalized.cronExpr;
    return false;
  });
}

export function schedulerQuestionForMissing(missing) {
  const labels = missing.map((slot) => schedulerSlotLabelKeys[slot]).filter(Boolean).map((key) => _t(key));
  return `Ai:${_t('chatSystem.scheduler.missingSlots', {labels: labels.join(_t('chatSystem.listSeparator'))})}`;
}

export function schedulerConfirmationMessage(draft, mode = 'create', job = null) {
  const normalized = normalizeSchedulerDraft(draft);
  const title = mode === 'update'
    ? _t('chatSystem.scheduler.confirmUpdateTitle', {name: job?.name || normalized.name})
    : _t('chatSystem.scheduler.confirmCreateTitle');
  const displayTime = formatSchedulerCronNextTime(normalized.cronExpr);
  return `Ai:${_t('chatSystem.scheduler.confirmation', {title, name: normalized.name, time: displayTime, summary: normalized.summary})}`;
}

export function buildSchedulerComposerConfirmAction(conversation, busy = false) {
  if (!conversation || conversation.phase !== 'confirm') return null;
  const normalized = normalizeSchedulerDraft(conversation.draft);
  return {
    type: 'scheduler',
    title: conversation.mode === 'update' ? '確認修改排程' : '確認寫入排程',
    primaryLabel: busy ? '寫入中' : '確定',
    cancelLabel: '取消',
    busy,
    lines: [
      `標題：${normalized.name}`,
      `時間：${formatSchedulerCronNextTime(normalized.cronExpr)}`,
      `摘要：${normalized.summary}`,
    ],
    summary: normalized.summary,
  };
}

export function mergeSchedulerSlotsFromText(currentDraft, text, preferredSlot = '') {
  const raw = String(text || '').trim();
  const next = {...(currentDraft || {})};
  const actionText = parseSchedulerActionText(raw);
  const cleanedName = cleanSchedulerNameText(raw);
  const deliveryText = extractSchedulerDeliveryText(raw);
  const explicitAction = /提醒|提醒我|叫我|通知我|動作|執行|做|跑/.test(raw);

  if (preferredSlot === 'name' && raw) next.name = raw.slice(0, 32);
  if (preferredSlot === 'action' && raw) next.actionText = raw.slice(0, 80);
  if (preferredSlot === 'time' && (hasSchedulerTimeText(raw) || hasSchedulerClockText(raw))) next.cronExpr = parseSchedulerTimeText(raw, next.cronExpr || '');

  if (hasSchedulerTimeText(raw)) next.cronExpr = parseSchedulerTimeText(raw, next.cronExpr || '');
  if (/名稱|標題/.test(raw) && cleanedName) next.name = cleanedName;
  if (explicitAction && actionText) next.actionText = actionText.slice(0, 80);
  if (!next.name && (cleanedName || actionText)) next.name = (actionText || cleanedName).slice(0, 32);
  if (!next.actionText && (cleanedName || actionText)) next.actionText = (actionText || cleanedName).slice(0, 80);
  if (deliveryText) next.deliveryText = deliveryText;
  if (raw) next.sourceText = [next.sourceText, raw].filter(Boolean).join('\n').slice(-500);

  const normalized = normalizeSchedulerDraft(next);
  normalized.actionPayload = normalized.actionText
    ? schedulerDefaultPayload(normalized.name || '提醒', normalized.actionText, normalized.summary)
    : '';
  return normalized;
}

export function findSchedulerJobFromText(text, jobs = []) {
  const raw = String(text || '');
  const indexWords = [
    ['第一', 0], ['第1', 0], ['1', 0],
    ['第二', 1], ['第2', 1], ['2', 1],
    ['第三', 2], ['第3', 2], ['3', 2],
    ['第四', 3], ['第4', 3], ['4', 3],
    ['第五', 4], ['第5', 4], ['5', 4],
  ];
  const indexed = indexWords.find(([word]) => raw.includes(`${word}個任務`) || raw.includes(`${word}個排程`) || raw.includes(`${word}任務`) || raw.includes(`${word}排程`));
  if (indexed) {
    const targetNo = indexed[1] + 1;
    return jobs.find((job, index) => schedulerJobNo(job, index) === targetNo) || null;
  }
  const hashNo = raw.match(/#\s*(\d+)|編號\s*(\d+)/);
  if (hashNo) {
    const targetNo = Number(hashNo[1] || hashNo[2]);
    return jobs.find((job, index) => schedulerJobNo(job, index) === targetNo) || null;
  }
  return jobs.find((job) => job?.name && raw.includes(job.name)) || null;
}

export function parseSchedulerConversationIntent(text, jobs = []) {
  const raw = String(text || '').trim();
  if (!/(排程|排定|定時|提醒|規劃|每小時|每天|每日|每週|每周|禮拜|星期)/.test(raw)) return null;
  const isUpdate = /(修改|更改|改成|改為|變更|調整)/.test(raw);
  if (isUpdate) {
    const job = findSchedulerJobFromText(raw, jobs);
    if (!job) return {type: 'open', message: '我找不到你想修改的排程，先打開清單讓你確認編號。'};
    const nextName = /名稱/.test(raw) ? (cleanSchedulerNameText(raw) || job.name) : job.name;
    const nextCron = hasSchedulerTimeText(raw)
      ? parseSchedulerTimeText(raw, job.cron_expr)
      : job.cron_expr;
    const nextActionText = /動作/.test(raw) ? parseSchedulerActionText(raw) : '';
    const nextPayload = nextActionText ? schedulerDefaultPayload(nextName, nextActionText) : job.action_payload;
    return {
      type: 'update',
      job,
      patch: {
        name: nextName,
        cronExpr: nextCron,
        actionType: job.action_type || 'event',
        actionPayload: nextPayload,
      },
    };
  }
  const draft = mergeSchedulerSlotsFromText({}, raw);
  return {
    type: 'create',
    draft,
  };
}

export function formatSchedulerTime(value) {
  if (!value) return '尚未排定';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return formatSchedulerDateKey(date);
}

export function formatSchedulerDateKey(value) {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return String(value || '');
  const pad = (number) => String(number).padStart(2, '0');
  return [
    date.getFullYear(),
    pad(date.getMonth() + 1),
    pad(date.getDate()),
    pad(date.getHours()),
    pad(date.getMinutes()),
  ].join('-');
}

export function schedulerCronPartNumber(part) {
  const raw = String(part || '').trim();
  return /^\d{1,2}$/.test(raw) ? Number(raw) : null;
}

export function schedulerCronPartMatches(part, value) {
  const raw = String(part || '').trim();
  if (raw === '*' || raw === '') return true;
  const number = schedulerCronPartNumber(raw);
  return number == null ? false : number === value;
}

export function nextSchedulerFireDate(cronExpr, baseDate = new Date()) {
  const raw = String(cronExpr || '').trim();
  if (!raw) return null;
  const base = new Date(baseDate);
  if (Number.isNaN(base.getTime())) return null;
  if (/^@hourly$/i.test(raw)) {
    const next = new Date(base);
    next.setMinutes(0, 0, 0);
    next.setHours(next.getHours() + 1);
    return next;
  }
  if (/^@daily$/i.test(raw)) return nextSchedulerFireDate('0 9 * * *', base);
  if (/^@weekly$/i.test(raw)) return nextSchedulerFireDate('0 9 * * 1', base);
  if (/^@monthly$/i.test(raw)) return nextSchedulerFireDate('0 9 1 * *', base);
  const parts = raw.split(/\s+/);
  if (parts.length !== 5) return null;
  const minute = schedulerCronPartNumber(parts[0]);
  const hour = schedulerCronPartNumber(parts[1]);
  if (minute == null || hour == null) return null;
  const start = new Date(base);
  start.setSeconds(0, 0);
  for (let offset = 0; offset <= 370; offset += 1) {
    const candidate = new Date(start);
    candidate.setDate(start.getDate() + offset);
    candidate.setHours(hour, minute, 0, 0);
    if (candidate <= start) continue;
    const dayOfMonth = candidate.getDate();
    const month = candidate.getMonth() + 1;
    const weekday = candidate.getDay();
    if (!schedulerCronPartMatches(parts[2], dayOfMonth)) continue;
    if (!schedulerCronPartMatches(parts[3], month)) continue;
    if (!schedulerCronPartMatches(parts[4], weekday)) continue;
    return candidate;
  }
  return null;
}

export function formatSchedulerCronNextTime(cronExpr, fallback = '') {
  const next = nextSchedulerFireDate(cronExpr);
  return next ? formatSchedulerDateKey(next) : (fallback || cronExpr || '尚未排定');
}

export function formatSchedulerJobNextTime(job) {
  const nextFire = job?.next_fire || job?.nextFire || '';
  if (nextFire) return formatSchedulerTime(nextFire);
  return formatSchedulerCronNextTime(job?.cron_expr || job?.cronExpr || '');
}

// 排程清單／摘要只顯示「下次會啟動的時間」：停用或無 next_fire 一律留空，
// 讓使用者一眼看出沒有排程；每日/每週/每月等 cron 規則不顯示給使用者。
export function schedulerActiveNextLabel(job) {
  if (!job || job.enabled === false) return '';
  const nextFire = job.next_fire || job.nextFire || '';
  return nextFire ? formatSchedulerTime(nextFire) : '';
}

export function formatSchedulerTimeLong(value) {
  if (!value) return '尚未排定';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-TW', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function formatSchedulerPayload(payload, fallback = '提醒') {
  const parsed = parseSchedulerPayload(payload);
  return parsed.action || parsed.title || parsed.event || payload || fallback;
}

export function formatSchedulerPayloadSummary(payload, fallback = '尚無摘要') {
  const parsed = parseSchedulerPayload(payload);
  return parsed.summary || parsed.action || parsed.title || payload || fallback;
}

export function parseSchedulerPayload(payload) {
  try {
    const parsed = JSON.parse(payload || '{}');
    return {
      title: parsed?.data?.title || '',
      action: parsed?.data?.action || '',
      summary: parsed?.data?.summary || '',
      event: parsed?.event_name || '',
    };
  } catch {
    return {title: '', action: String(payload || ''), summary: '', event: ''};
  }
}

export function schedulerJobNo(job, fallbackIndex = 0) {
  const no = Number(job?.schedule_no ?? job?.scheduleNo ?? 0);
  return no > 0 ? no : fallbackIndex + 1;
}

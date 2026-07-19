import {t as translate, tForLanguage} from '../locales/useI18n';

const composerPendingInitialText = translate('composer.pendingInitial');
export const composerPendingSlowText = translate('composer.pendingSlow');
export const composerPendingVerySlowText = translate('composer.pendingVerySlow');
const composerPendingMarker = '\u2063pending:';

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

export function updateComposerPendingMessage(messages, traceId, replacement) {
  const needle = `${composerPendingMarker}${traceId}`;
  const index = messages.findIndex((message) => String(message || '').includes(needle));
  if (index < 0) return messages;
  const next = [...messages];
  next[index] = replacement;
  return next;
}

export function localizeChatSystemMessage(message, chatLocale = 'zh-TW') {
  const raw = String(message || '');
  const body = raw.replace(/^Ai:/, '').trim();
  const ai = raw.startsWith('Ai:');
  const localized = (key, params = {}) => `${ai ? 'Ai:' : ''}${tForLanguage(chatLocale, key, params)}`;
  if (ai && body === '請選擇：') return localized('chatSystem.choose');

  // 舊版對話是已渲染的中文純字串；相容層讓已落盤訊息也能隨面板語系切換。
  let match;
  if (ai && (match = body.match(/^我還讀不到上一段示範：([\s\S]*)$/))) {
    return localized('chatSystem.learning.replayReadFailed', {error: match[1]});
  }
  if (ai && (match = body.match(/^我收到 replay 指令，但讀不到上一段示範：([\s\S]*)$/))) {
    return localized('chatSystem.learning.replayDirectiveReadFailed', {error: match[1]});
  }
  if (ai && (match = body.match(/^我讀不到已保存操作 catalog：([\s\S]*)$/))) {
    return localized('chatSystem.learning.catalogReadFailed', {error: match[1]});
  }
  if (ai && body === 'Replay 尚未執行；我已保留上面的安全 plan，確認內容正確後可以再輸入「請按照我剛剛的示範」。') {
    return localized('chatSystem.learning.replayCancelled');
  }
  if (ai && body === '已中斷本次回覆。') return localized('chatSystem.replyInterrupted');
  if (ai && (match = body.match(/^任務已開始：([\s\S]+)$/))) {
    return localized('chatSystem.taskStarted', {title: match[1]});
  }
  if (ai && body === '好，我先取消這次排程確認。') return localized('chatSystem.scheduler.confirmationCancelled');
  if (ai && body === '好，我先取消這次排程設定。') return localized('chatSystem.scheduler.setupCancelled');
  if (ai && body === '我先打開排程清單，請確認要看或修改哪一筆。') return localized('chatSystem.scheduler.openList');
  if (ai && body === '我找不到你想修改的那一筆排程，先打開清單讓你確認編號。') return localized('chatSystem.scheduler.updateNotFound');
  if (!ai && raw === '[系統] 已把這次示範存成候選流程。') return localized('chatSystem.learning.candidateSaved');
  if (!ai && raw === '[系統] 已放棄這次示範。') return localized('chatSystem.learning.demoDiscarded');
  if (!ai && raw === '[系統] 學習模式關閉，生態系閒置整理已停止。') return localized('chatSystem.learning.modeOff');
  if (!ai && (match = raw.match(/^\[系統\] 學習模式開啟。閒置 (\d+) 分鐘後/))) {
    return localized('chatSystem.learning.modeOn', {minutes: match[1]});
  }
  return raw;
}

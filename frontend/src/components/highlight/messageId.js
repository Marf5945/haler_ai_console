// messageId.js — 穩定訊息 id（content hash），取代易位移的 index。
//
// 同一段訊息文字 → 同一 id，刪 / 插其他訊息不影響，標註與暗標都能正確再錨定。
// 含「相同文字第幾次出現」以避免重複訊息撞 id。
// MessageRow（data-message-id）、useSystemMarks、刪訊息級聯清除三處共用，
// 必須回傳一致結果。

export function hashStr(s) {
  let h = 5381;
  const str = String(s || '');
  for (let i = 0; i < str.length; i++) {
    h = ((h << 5) + h) ^ str.charCodeAt(i);
  }
  return (h >>> 0).toString(36);
}

export function messageDomId(message, index, allMessages = []) {
  const m = String(message || '');
  let occ = 0;
  for (let i = 0; i < index && i < allMessages.length; i++) {
    if (String(allMessages[i] || '') === m) occ++;
  }
  return `m_${hashStr(m)}_${occ}`;
}

export default messageDomId;

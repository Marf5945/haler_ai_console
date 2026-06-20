import { t as _t } from '../locales/useI18n';

// Panel/style/font/voice settings domain — extracted from App.jsx so the
// settings components and App shell share a single source of truth.

export const STYLE_KEYS = ['default', 'passiveWhite', 'pinkBetrayal', 'forgiveMeGreen', 'defeatBlue'];
export const STYLE_KEY_I18N = {
  default: 'style.default',
  passiveWhite: 'style.options.passiveWhite',
  pinkBetrayal: 'style.options.pinkBetrayal',
  forgiveMeGreen: 'style.options.forgiveMeGreen',
  defeatBlue: 'style.options.defeatBlue',
};
export const defaultPanelStyle = 'default';
export const styleOptions = STYLE_KEYS;
// Panel font scale is a UI-wide preference; CSS consumes it through --ui-font-scale.
export const fontScaleOptions = ['50%', '80%', '90%', '100%', '110%', '120%'];
export const styleOptionOrderStorageKey = 'ai-console.style-option-order';

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
  '한국어': 'settings.langKo',
  '韓文': 'settings.langKo',
  '韓語': 'settings.langKo',
  'Korean': 'settings.langKo',
  'ko': 'settings.langKo',
  // self-heal: raw i18n keys leaked by an older build
  'settings.langPt': 'settings.langPt',
  'settings.langEs': 'settings.langEs',
  'settings.langTh': 'settings.langTh',
  'settings.langKo': 'settings.langKo',
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
  '한국어': 'settings.langKo',
  '韓文': 'settings.langKo',
  '韓語': 'settings.langKo',
  'Korean': 'settings.langKo',
  'ko': 'settings.langKo',
};
export const _fontPresetLabelMap = {
  // 預設（沿用系統/語言字型，不覆寫）
  '預設': 'settings.fontDefault', 'Default': 'settings.fontDefault', '標準': 'settings.fontDefault',
  'Predefinido': 'settings.fontDefault', 'Predeterminado': 'settings.fontDefault', 'ค่าเริ่มต้น': 'settings.fontDefault',
  // 普通（中性內建無襯線）
  '普通': 'settings.fontNormal', 'Normal': 'settings.fontNormal', '通常': 'settings.fontNormal', 'ปกติ': 'settings.fontNormal',
  // 手寫
  '手寫': 'settings.fontHand', 'Handwriting': 'settings.fontHand', '手書き': 'settings.fontHand',
  'Manuscrito': 'settings.fontHand', 'Manuscrita': 'settings.fontHand', 'ลายมือ': 'settings.fontHand',
  // 書法 / 毛筆
  '書法': 'settings.fontCalli', 'Calligraphy': 'settings.fontCalli', '毛筆': 'settings.fontCalli',
  'Caligrafia': 'settings.fontCalli', 'Caligrafía': 'settings.fontCalli', 'พู่กัน': 'settings.fontCalli',
  // 圓潤 / 普普
  '圓潤': 'settings.fontRound', 'Rounded': 'settings.fontRound', '丸ゴシック': 'settings.fontRound',
  'Arredondado': 'settings.fontRound', 'Redondeada': 'settings.fontRound', 'มน': 'settings.fontRound',
  // 等寬
  '等寬': 'settings.fontMono', 'Monospace': 'settings.fontMono', '等幅': 'settings.fontMono',
  'Monoespaçado': 'settings.fontMono', 'Monoespaciada': 'settings.fontMono', 'ความกว้างคงที่': 'settings.fontMono',
};

// Any historical/localized label (zh-TW / en / ja, current + legacy) → stable key.
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

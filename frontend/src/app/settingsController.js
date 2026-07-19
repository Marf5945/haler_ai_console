import useI18n, {t as _t, tForLanguage} from '../locales/useI18n';
import {
  _fontPresetLabelMap,
  _panelLangLabelMap,
  _roleLangLabelMap,
  defaultPanelStyle,
  localizeBackendLabel,
  normalizePanelStyle,
  styleKeyOf,
} from '../lib/panelSettings';
import {
  BUILT_IN_PERSONA_BY_ID,
  LOCKED_PERSONA_ID,
  createBuiltInPersonaSeeds,
} from './personas/registry';

const STYLE_KEY_THEME = {
  default: 'onanegiku', passiveWhite: 'white', pinkBetrayal: 'pink-black',
  forgiveMeGreen: 'green', defeatBlue: 'blue',
};

const FONT_PRESET_STACKS = {
  'settings.fontNormal': "'Inter','Noto Sans TC','Noto Sans JP',sans-serif",
  'settings.fontHand': "'Caveat','Klee One','LXGW WenKai','Noto Sans TC','Noto Sans JP',cursive",
  'settings.fontCalli': "'Dancing Script','LXGW WenKai','Yuji Syuku','Noto Serif TC','Noto Sans JP',serif",
  'settings.fontRound': "'Fredoka','jf-openhuninn','Noto Sans JP','Noto Sans TC',sans-serif",
  'settings.fontMono': "'JetBrains Mono','Noto Sans TC','Noto Sans JP',monospace",
};

export const fallbackSettings = {
  panel: {
    panelLanguage: _t('settings.langZhTW'), roleLanguage: _t('settings.roleLangAuto'),
    fontPreset: _t('settings.fontDefault'), fontScale: '100%', panelStyle: defaultPanelStyle,
  },
  activePersonaId: LOCKED_PERSONA_ID,
  personas: createBuiltInPersonaSeeds(_t),
  removedDefaultPersonaIds: [],
};

export function panelLanguageLabelToLocale(displayLabel, {allowAuto = true} = {}) {
  const map = {
    [_t('settings.langZhTW')]: 'zh-TW', [_t('settings.langEn')]: 'en', [_t('settings.langJa')]: 'ja',
    [_t('settings.langPt')]: 'pt-PT', [_t('settings.langEs')]: 'es', [_t('settings.langTh')]: 'th',
    [_t('settings.langKo')]: 'ko', [_t('settings.langAr')]: 'ar',
    '繁中': 'zh-TW', '中文': 'zh-TW', '英文': 'en', '日文': 'ja',
    'Traditional Chinese': 'zh-TW', 'Chinese': 'zh-TW', 'English': 'en', 'Japanese': 'ja',
    '中': 'zh-TW', en: 'en', ja: 'ja', pt: 'pt-PT', 'pt-PT': 'pt-PT', es: 'es', th: 'th', ko: 'ko',
    'Português': 'pt-PT', '葡萄牙文': 'pt-PT', 'Español': 'es', '西班牙文': 'es',
    'ไทย': 'th', '泰文': 'th', '한국어': 'ko', '韓文': 'ko', '韓語': 'ko', Korean: 'ko',
    ar: 'ar', Arabic: 'ar', 'العربية': 'ar', '阿拉伯文': 'ar', '阿拉伯語': 'ar',
    'settings.langPt': 'pt-PT', 'settings.langEs': 'es', 'settings.langTh': 'th',
    'settings.langKo': 'ko', 'settings.langAr': 'ar',
  };
  if (!allowAuto) {
    const autoLabels = new Set([_t('settings.roleLangAuto'), _t('settings.roleLanguageAuto'), '自動', 'Auto', 'auto']);
    if (autoLabels.has(displayLabel)) return null;
  }
  return map[displayLabel] || null;
}

export function personaLocaleFromPanel(panel = {}) {
  return panelLanguageLabelToLocale(panel.panelLanguage) || useI18n.getState().language || 'zh-TW';
}

function personaI18n(locale, key, params) {
  return locale ? tForLanguage(locale, key, params) : _t(key, params);
}

export function fontPresetKey(value) {
  if (!value) return 'settings.fontDefault';
  if (_fontPresetLabelMap[value]) return _fontPresetLabelMap[value];
  for (const key of new Set(Object.values(_fontPresetLabelMap))) {
    if (_t(key) === value) return key;
  }
  return 'settings.fontDefault';
}

export function fontPresetVars(value) {
  const stack = FONT_PRESET_STACKS[fontPresetKey(value)];
  return stack ? {'--font-console': stack, '--i18n-font': stack} : {};
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

function isLegacyDefaultCopy(value, legacyValues = []) {
  const text = String(value || '').trim();
  return text === '' || legacyValues.includes(text);
}

export function normalizeBuiltInPersonaCopy(persona = {}, personaLocale = '') {
  const copy = BUILT_IN_PERSONA_BY_ID[persona.id]?.copy;
  if (!copy) return persona;
  return {
    ...persona,
    name: isLegacyDefaultCopy(persona.name, copy.legacyNames) ? personaI18n(personaLocale, copy.nameKey) : persona.name,
    identity: isLegacyDefaultCopy(persona.identity, copy.legacyIdentities) ? personaI18n(personaLocale, copy.identityKey) : persona.identity,
    personality: isLegacyDefaultCopy(persona.personality, copy.legacyPersonalities)
      ? (copy.personalityKey ? personaI18n(personaLocale, copy.personalityKey) : '')
      : persona.personality,
  };
}

function appendMissingDefaultPersonas(personas = [], removedDefaultPersonaIds = [], personaLocale = '') {
  const known = new Set(personas.map((persona) => persona.id));
  const removed = new Set(removedDefaultPersonaIds || []);
  const seeds = fallbackSettings.personas.filter((persona) => (
    persona.id !== LOCKED_PERSONA_ID && !known.has(persona.id) && !removed.has(persona.id)
  )).map((persona) => normalizeBuiltInPersonaCopy(persona, personaLocale));
  return seeds.length > 0 ? [...personas, ...seeds] : personas;
}

export function normalizeLockedPersonas(personas = [], removedDefaultPersonaIds = [], personaLocale = '') {
  const normalized = personas.map((persona) => {
    const withPatrolDialogue = {...persona, patrolDialogue: persona?.patrolDialogue ?? persona?.patrol_dialogue ?? ''};
    const localizedCopy = normalizeBuiltInPersonaCopy(withPatrolDialogue, personaLocale);
    return persona.id === LOCKED_PERSONA_ID
      ? {...localizedCopy, name: personaI18n(personaLocale, 'persona.lockedName')}
      : localizedCopy;
  });
  if (!normalized.some((persona) => persona.id === LOCKED_PERSONA_ID)) {
    const locked = normalizeBuiltInPersonaCopy(fallbackSettings.personas[0], personaLocale);
    return appendMissingDefaultPersonas([locked, ...normalized], removedDefaultPersonaIds, personaLocale);
  }
  return appendMissingDefaultPersonas(normalized, removedDefaultPersonaIds, personaLocale);
}

export function normalizeSettingsState(settingsState = {}, fallback = fallbackSettings) {
  const merged = {
    ...fallback, ...settingsState,
    panel: {...(fallback.panel || fallbackSettings.panel), ...(settingsState.panel || {})},
  };
  const personas = normalizeLockedPersonas(
    merged.personas || fallbackSettings.personas,
    merged.removedDefaultPersonaIds || [],
    personaLocaleFromPanel(merged.panel),
  );
  const activePersonaId = personas.some((persona) => persona.id === merged.activePersonaId)
    ? merged.activePersonaId : LOCKED_PERSONA_ID;
  return {...merged, activePersonaId, personas, panel: normalizePanelSettings(merged.panel)};
}

export function panelStyleTheme(style) {
  return STYLE_KEY_THEME[styleKeyOf(style)] || 'onanegiku';
}

export function fontScaleValue(fontScale) {
  const value = Number.parseInt(String(fontScale || '100%').replace('%', ''), 10);
  const normalized = Number.isNaN(value) ? 100 : value;
  return Math.min(120, Math.max(50, normalized)) / 100;
}

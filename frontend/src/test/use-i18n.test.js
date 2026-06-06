import { beforeEach, describe, expect, it } from 'vitest';
import useI18n from '../locales/useI18n';

describe('useI18n', () => {
  beforeEach(() => {
    window.localStorage.clear();
    useI18n.getState().setLanguage('zh-TW');
  });

  it('switches language in place without requiring a page reload', () => {
    useI18n.getState().setLanguage('en');

    expect(useI18n.getState().language).toBe('en');
    expect(window.localStorage.getItem('i18n_language')).toBe('en');
    expect(document.documentElement.lang).toBe('en');
    expect(document.documentElement.dir).toBe('ltr');
    expect(useI18n.getState().t('settings.panelSettings')).toBe('Panel Settings');
  });
});

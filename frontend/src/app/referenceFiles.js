import {t as translate} from '../locales/useI18n';

export function referenceFileKey(file) {
  return String(file?.path || file?.name || '');
}

export function sharedSourceKey(link) {
  return String(link?.id || link?.ID || link?.url || link?.URL || '');
}

export function referenceFileStatusLabel(status, t = translate) {
  return {
    importing: t('rightRail.refStatusImporting'),
    checking: t('rightRail.refStatusChecking'),
    ready: t('rightRail.refStatusReady'),
    error: t('rightRail.refStatusError'),
  }[status] || t('rightRail.refStatusAdded');
}

export function shouldShowReferenceFileDetail(file) {
  if (!file?.detail) return false;
  return !(file.source === 'library' && file.status === 'ready');
}

export function isVideoPath(path) {
  return /\.(mp4|mov|m4v|webm|mkv|avi|wmv|flv|mpe?g|3gp|ogv)$/i.test(String(path || ''));
}

export function fileExtLabel(name) {
  const match = /\.([A-Za-z0-9]+)$/.exec(String(name || ''));
  return match ? match[1].toLowerCase() : '';
}

export function fileBaseName(path) {
  return String(path || '').split(/[\\/]/).filter(Boolean).pop() || '';
}

export function twoLineFileName(name, fallback = 'Unnamed File') {
  const chars = Array.from(String(name || fallback));
  const first = chars.slice(0, 20).join('');
  const second = chars.slice(20, 40).join('');
  return second ? [first, second] : [first];
}
